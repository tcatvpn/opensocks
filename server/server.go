package server

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gobwas/ws"
	"github.com/inhies/go-bytesize"
	"github.com/net-byte/opensocks/common/cipher"
	"github.com/net-byte/opensocks/common/enum"
	"github.com/net-byte/opensocks/config"
	"github.com/net-byte/opensocks/counter"
	"github.com/net-byte/opensocks/proto"
	"github.com/net-byte/opensocks/proxy"
	"github.com/xtaci/smux"
)

// Start server
func Start(config config.Config) {
	http.HandleFunc(enum.WSPath, func(w http.ResponseWriter, r *http.Request) {
		conn, _, _, err := ws.UpgradeHTTP(r, w)
		if err != nil {
			log.Printf("[server] failed to upgrade http %v", err)
			return
		}
		wsHandler(conn, config)
	})

	http.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {
		io.WriteString(w, "Hello, world!\n")
	})

	http.HandleFunc("/ip", func(w http.ResponseWriter, req *http.Request) {
		ip := req.Header.Get("X-Forwarded-For")
		if ip == "" {
			ip = strings.Split(req.RemoteAddr, ":")[0]
		}
		resp := fmt.Sprintf("%v", ip)
		io.WriteString(w, resp)
	})

	http.HandleFunc("/sys", func(w http.ResponseWriter, req *http.Request) {
		resp := fmt.Sprintf("download %v upload %v \r\n", bytesize.New(float64(counter.TotalWrittenBytes)).String(), bytesize.New(float64(counter.TotalReadBytes)).String())
		io.WriteString(w, resp)
	})

	log.Printf("opensocks server started on %s", config.ServerAddr)
	http.ListenAndServe(config.ServerAddr, nil)
}

func wsHandler(w net.Conn, config config.Config) {
	defer w.Close()
	session, err := smux.Server(w, nil)
	if err != nil {
		log.Printf("[server] failed to initialise yamux session: %s", err)
		return
	}
	defer session.Close()
	for {
		stream, err := session.AcceptStream()
		if err != nil {
			log.Printf("[server] failed to accept steam %v", err)
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
			log.Printf("[server] dial to server %v %v:%v", req.Network, req.Host, req.Port)
			conn, err := net.DialTimeout(req.Network, net.JoinHostPort(req.Host, req.Port), time.Duration(enum.Timeout)*time.Second)
			if err != nil {
				log.Printf("[server] failed to dial server %v", err)
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
		log.Printf("[server] failed to decode request %v", err)
		return false, req
	}
	reqTime, _ := strconv.ParseInt(req.Timestamp, 10, 64)
	if time.Now().Unix()-reqTime > int64(enum.Timeout) {
		log.Printf("[server] timestamp expired %v", reqTime)
		return false, req
	}
	if config.Key != req.Key {
		log.Printf("[server] error key %s", req.Key)
		return false, req
	}
	return true, req
}

func toClient(config config.Config, stream net.Conn, conn net.Conn) {
	defer conn.Close()
	buffer := config.BytePool.Get()
	defer config.BytePool.Put(buffer)
	for {
		conn.SetReadDeadline(time.Now().Add(time.Duration(enum.Timeout) * time.Second))
		n, err := conn.Read(buffer)
		if err != nil || n == 0 {
			break
		}
		b := buffer[:n]
		if config.Obfs {
			b = cipher.XOR(b)
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
	buffer := config.BytePool.Get()
	defer config.BytePool.Put(buffer)
	for {
		n, err := reader.Read(buffer)
		if err != nil || n == 0 {
			break
		}
		b := buffer[:n]
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
