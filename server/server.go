package server

import (
	"bufio"
	"context"
	"crypto/sha1"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gobwas/ws"
	"github.com/golang/snappy"
	"github.com/net-byte/opensocks/common/cipher"
	"github.com/net-byte/opensocks/common/enum"
	"github.com/net-byte/opensocks/common/pool"
	"github.com/net-byte/opensocks/common/util"
	"github.com/net-byte/opensocks/config"
	"github.com/net-byte/opensocks/counter"
	"github.com/net-byte/opensocks/proto"
	"github.com/net-byte/opensocks/proxy"
	"github.com/xtaci/kcp-go/v5"
	"github.com/xtaci/smux"
	"golang.org/x/crypto/pbkdf2"
)

var _defaultPage = []byte(`
<!DOCTYPE html>
<html>
<head>
<title>Welcome to nginx!</title>
<style>
	body {
		width: 35em;
		margin: 0 auto;
		font-family: Tahoma, Verdana, Arial, sans-serif;
	}
</style>
</head>
<body>
<h1>Welcome to nginx!</h1>
<p>If you see this page, the nginx web server is successfully installed and
working. Further configuration is required.</p>

<p>For online documentation and support please refer to
<a href="http://nginx.org/">nginx.org</a>.<br/>
Commercial support is available at
<a href="http://nginx.com/">nginx.com</a>.</p>

<p><em>Thank you for using nginx.</em></p>
</body>
</html>`)

var _wsServer http.Server
var _tcpListener net.Listener
var _kcpListener *kcp.Listener
var _serverType string

// Start starts the server
func Start(config config.Config) {
	util.PrintStats(config.Verbose)
	switch config.Protocol {
	case "kcp":
		_serverType = "kcp"
		startKCPServer(config)
	case "tcp":
		_serverType = "tcp"
		startTCPServer(config)
	default:
		_serverType = "ws"
		startWSServer(config)
	}
}

// Stop starts the server
func Stop() {
	switch _serverType {
	case "kcp":
		if err := _kcpListener.Close(); err != nil {
			log.Printf("failed to shutdown kcp server: %v", err)
		}
	case "tcp":
		if err := _tcpListener.Close(); err != nil {
			log.Printf("failed to shutdown tcp server: %v", err)
		}
	default:
		if err := _wsServer.Shutdown(context.Background()); err != nil {
			log.Printf("failed to shutdown ws server: %v", err)
		}
	}
}

func startWSServer(config config.Config) {
	http.HandleFunc(enum.WSPath, func(w http.ResponseWriter, r *http.Request) {
		conn, _, _, err := ws.UpgradeHTTP(r, w)
		if err != nil {
			log.Printf("[server] failed to upgrade http %v", err)
			return
		}
		muxHandler(conn, config)
	})

	http.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {
		w.WriteHeader(200)
		w.Header().Set("Server", "nginx/1.18.0 (Ubuntu)")
		w.Header().Set("Content-Type", "text/html")
		w.Header().Set("Content-Length", strconv.Itoa(len(_defaultPage)))
		w.Header().Set("Connection", " keep-alive")
		w.Header().Set("Accept-Ranges", "bytes")
		w.Write(_defaultPage)
	})

	http.HandleFunc("/ip", func(w http.ResponseWriter, req *http.Request) {
		ip := req.Header.Get("X-Forwarded-For")
		if ip == "" {
			ip = strings.Split(req.RemoteAddr, ":")[0]
		}
		resp := fmt.Sprintf("%v", ip)
		io.WriteString(w, resp)
	})

	http.HandleFunc("/stats", func(w http.ResponseWriter, req *http.Request) {
		io.WriteString(w, counter.PrintBytes())
	})

	log.Printf("opensocks ws server started on %s", config.ServerAddr)
	_wsServer = http.Server{
		Addr: config.ServerAddr,
	}
	_wsServer.ListenAndServe()
}

func startKCPServer(config config.Config) {
	key := pbkdf2.Key([]byte(config.Key), []byte("opensocks@2022"), 1024, 32, sha1.New)
	block, _ := kcp.NewAESBlockCrypt(key)
	var err error
	if _kcpListener, err = kcp.ListenWithOptions(config.ServerAddr, block, 10, 3); err == nil {
		log.Printf("opensocks kcp server started on %s", config.ServerAddr)
		for {
			conn, err := _kcpListener.AcceptKCP()
			if err != nil {
				break
			}
			conn.SetWindowSize(enum.SndWnd, enum.RcvWnd)
			if err := conn.SetReadBuffer(enum.SockBuf); err != nil {
				log.Println("[server] failed to set read buffer:", err)
			}
			go muxHandler(conn, config)
		}
	}
}

func startTCPServer(config config.Config) {
	var err error
	if _tcpListener, err = net.Listen("tcp", config.ServerAddr); err == nil {
		log.Printf("opensocks tcp server started on %s", config.ServerAddr)
		for {
			conn, err := _tcpListener.Accept()
			if err != nil {
				break
			}
			go muxHandler(conn, config)
		}
	}
}

func muxHandler(w net.Conn, config config.Config) {
	defer w.Close()
	smuxConfig := smux.DefaultConfig()
	smuxConfig.Version = enum.SmuxVer
	smuxConfig.MaxReceiveBuffer = enum.SmuxBuf
	smuxConfig.MaxStreamBuffer = enum.StreamBuf
	session, err := smux.Server(w, smuxConfig)
	if err != nil {
		log.Printf("[server] failed to initialise yamux session: %s", err)
		return
	}
	defer session.Close()
	for {
		stream, err := session.AcceptStream()
		if err != nil {
			util.PrintLog(config.Verbose, "[server] failed to accept steam %v", err)
			break
		}
		go func() {
			defer stream.Close()
			reader := bufio.NewReader(stream)
			// handshake
			ok, req := handshake(config, reader)
			if !ok {
				return
			}
			util.PrintLog(config.Verbose, "[server] dial to server %v", req.Network, req.Host, req.Port)
			conn, err := net.DialTimeout(req.Network, net.JoinHostPort(req.Host, req.Port), time.Duration(enum.Timeout)*time.Second)
			if err != nil {
				util.PrintLog(config.Verbose, "[server] failed to dial server %v", err)
				return
			}
			// forward data
			go toServer(config, reader, conn)
			toClient(config, stream, conn)
		}()
	}
}

func handshake(config config.Config, reader *bufio.Reader) (bool, proxy.RequestAddr) {
	var req proxy.RequestAddr
	b, _, err := proto.Decode(reader)
	if err != nil {
		return false, req
	}
	if config.Obfs {
		b = cipher.XOR(b)
	}
	if req.UnmarshalBinary(b) != nil {
		util.PrintLog(config.Verbose, "[server] failed to decode request %v", err)
		return false, req
	}
	reqTime, _ := strconv.ParseInt(req.Timestamp, 10, 64)
	if time.Now().Unix()-reqTime > int64(enum.Timeout) {
		util.PrintLog(config.Verbose, "[server] timestamp expired %v", reqTime)
		return false, req
	}
	if config.Key != req.Key {
		util.PrintLog(config.Verbose, "[server] error key %s", req.Key)
		return false, req
	}
	return true, req
}

func toClient(config config.Config, stream net.Conn, conn net.Conn) {
	defer conn.Close()
	buffer := pool.BytePool.Get()
	defer pool.BytePool.Put(buffer)
	for {
		conn.SetReadDeadline(time.Now().Add(time.Duration(enum.Timeout) * time.Second))
		n, err := conn.Read(buffer)
		if err != nil {
			break
		}
		b := buffer[:n]
		if config.Obfs {
			b = cipher.XOR(b)
		}
		if config.Compress {
			b = snappy.Encode(nil, b)
		}
		_, err = stream.Write(b)
		if err != nil {
			break
		}
		counter.IncrWrittenBytes(n)
	}
}

func toServer(config config.Config, reader *bufio.Reader, conn net.Conn) {
	defer conn.Close()
	buffer := pool.BytePool.Get()
	defer pool.BytePool.Put(buffer)
	for {
		n, err := reader.Read(buffer)
		if err != nil {
			break
		}
		b := buffer[:n]
		if config.Compress {
			b, err = snappy.Decode(nil, b)
			if err != nil {
				util.PrintLog(config.Verbose, "failed to decode:%v", err)
				break
			}
		}
		if config.Obfs {
			b = cipher.XOR(b)
		}
		_, err = conn.Write(b)
		if err != nil {
			break
		}
		counter.IncrReadBytes(int(n))
	}
}
