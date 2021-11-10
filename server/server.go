package server

import (
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gobwas/ws"
	"github.com/gobwas/ws/wsutil"
	"github.com/inhies/go-bytesize"
	"github.com/net-byte/opensocks/common/cipher"
	"github.com/net-byte/opensocks/common/constant"
	"github.com/net-byte/opensocks/config"
	"github.com/net-byte/opensocks/counter"
	"github.com/net-byte/opensocks/proxy"
)

// Start server
func Start(config config.Config) {
	http.HandleFunc(constant.WSPath, func(w http.ResponseWriter, r *http.Request) {
		conn, _, _, err := ws.UpgradeHTTP(r, w)
		if err != nil {
			log.Printf("[server] failed to upgrade http %v", err)
			return
		}
		wsHandler(conn, config)
	})

	http.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {
		io.WriteString(w, "Hello，世界！")
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
		resp := fmt.Sprintf("download %v upload %v", bytesize.New(float64(counter.TotalWrittenBytes)).String(), bytesize.New(float64(counter.TotalReadBytes)).String())
		io.WriteString(w, resp)
	})

	log.Printf("opensocks server started on %s", config.ServerAddr)
	http.ListenAndServe(config.ServerAddr, nil)
}

func wsHandler(wsconn net.Conn, config config.Config) {
	// handshake
	ok, req := handshake(config, wsconn)
	if !ok {
		wsconn.Close()
		return
	}
	// connect real server
	// log.Printf("[server] dial to server %v %v:%v", req.Network, req.Host, req.Port)
	conn, err := net.DialTimeout(req.Network, net.JoinHostPort(req.Host, req.Port), time.Duration(constant.Timeout)*time.Second)
	if err != nil {
		wsconn.Close()
		log.Printf("[server] failed to dial server %v", err)
		return
	}
	// forward data
	go toRemote(config, wsconn, conn)
	toLocal(config, wsconn, conn)
}

func handshake(config config.Config, conn net.Conn) (bool, proxy.RequestAddr) {
	var req proxy.RequestAddr
	buffer, err := wsutil.ReadClientText(conn)
	if err != nil {
		return false, req
	}
	if config.Obfuscate {
		buffer = cipher.XOR(buffer)
	}
	if req.UnmarshalBinary(buffer) != nil {
		log.Printf("[server] failed to decode request %v", err)
		return false, req
	}
	reqTime, _ := strconv.ParseInt(req.Timestamp, 10, 64)
	if time.Now().Unix()-reqTime > int64(constant.Timeout) {
		log.Printf("[server] timestamp expired %v", reqTime)
		return false, req
	}
	if config.Key != req.Key {
		log.Printf("[server] error key %s", req.Key)
		return false, req
	}
	return true, req
}

func toLocal(config config.Config, wsconn net.Conn, tcpconn net.Conn) {
	defer wsconn.Close()
	defer tcpconn.Close()
	buffer := make([]byte, constant.BufferSize)
	for {
		tcpconn.SetReadDeadline(time.Now().Add(time.Duration(constant.Timeout) * time.Second))
		n, err := tcpconn.Read(buffer)
		if err != nil || err == io.EOF || n == 0 {
			break
		}
		var b []byte
		if config.Obfuscate {
			b = cipher.XOR(buffer[:n])
		} else {
			b = buffer[:n]
		}
		counter.IncrWrittenBytes(n)
		wsutil.WriteServerBinary(wsconn, b)
	}
}

func toRemote(config config.Config, wsconn net.Conn, tcpconn net.Conn) {
	defer wsconn.Close()
	defer tcpconn.Close()
	for {
		wsconn.SetReadDeadline(time.Now().Add(time.Duration(constant.Timeout) * time.Second))
		buffer, err := wsutil.ReadClientBinary(wsconn)
		n := len(buffer)
		if err != nil || err == io.EOF || n == 0 {
			break
		}
		if config.Obfuscate {
			buffer = cipher.XOR(buffer)
		}
		counter.IncrReadBytes(n)
		tcpconn.Write(buffer[:])
	}
}
