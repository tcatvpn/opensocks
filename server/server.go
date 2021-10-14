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

	"github.com/gorilla/websocket"
	"github.com/inhies/go-bytesize"
	"github.com/net-byte/opensocks/common/cipher"
	"github.com/net-byte/opensocks/common/constant"
	"github.com/net-byte/opensocks/config"
	"github.com/net-byte/opensocks/counter"
	"github.com/net-byte/opensocks/proxy"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  constant.BufferSize,
	WriteBufferSize: constant.BufferSize,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

// Start starts server
func Start(config config.Config) {
	log.Printf("opensocks server started on %s", config.ServerAddr)
	http.HandleFunc(constant.WSPath, func(w http.ResponseWriter, r *http.Request) {
		wsConn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		// handshake
		ok, req := handshake(config, wsConn)
		if !ok {
			wsConn.Close()
			return
		}
		// connect real server
		conn, err := net.DialTimeout(req.Network, net.JoinHostPort(req.Host, req.Port), time.Duration(constant.Timeout)*time.Second)
		if err != nil {
			wsConn.Close()
			log.Printf("[server] failed to dial the real server%v", err)
			return
		}
		// forward data
		go proxy.WSToTCP(config, wsConn, conn)
		go proxy.TCPToWS(config, wsConn, conn)
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
		resp := fmt.Sprintf("download %v upload %v", bytesize.New(float64(counter.TotalWriteByte)).String(), bytesize.New(float64(counter.TotalReadByte)).String())
		io.WriteString(w, resp)
	})

	http.ListenAndServe(config.ServerAddr, nil)
}

func handshake(config config.Config, wsConn *websocket.Conn) (bool, proxy.RequestAddr) {
	var req proxy.RequestAddr
	_, buffer, err := wsConn.ReadMessage()
	if err != nil {
		return false, req
	}
	if config.Obfuscate {
		buffer = cipher.XOR(buffer)
	}
	if req.UnmarshalBinary(buffer) != nil {
		log.Printf("[server] failed to unmarshal binary %v", err)
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
