package server

import (
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"strings"
	"syscall"
	"time"

	"github.com/gorilla/websocket"
	"github.com/net-byte/opensocks/config"
	"github.com/net-byte/opensocks/proxy"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:    proxy.BufferSize,
	WriteBufferSize:   proxy.BufferSize,
	EnableCompression: true,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

// Start starts server
func Start(config config.Config) {
	var rLimit syscall.Rlimit
	if err := syscall.Getrlimit(syscall.RLIMIT_NOFILE, &rLimit); err != nil {
		log.Panicf("[server] Getrlimit error:%v", err)
	}
	rLimit.Cur = rLimit.Max
	if err := syscall.Setrlimit(syscall.RLIMIT_NOFILE, &rLimit); err != nil {
		log.Panicf("[server] Setrlimit error:%v", err)
	}
	log.Printf("opensocks server started on %s", config.ServerAddr)

	http.HandleFunc("/opensocks", func(w http.ResponseWriter, r *http.Request) {
		wsConn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		_, buffer, err := wsConn.ReadMessage()
		if err != nil {
			return
		}
		var req proxy.RequestAddr
		if req.UnmarshalBinary(buffer) != nil {
			log.Printf("[server] UnmarshalBinary error:%v", err)
			return
		}
		if config.Username != req.Username || config.Password != req.Password {
			log.Printf("[server] error username %s,password %s", req.Username, req.Password)
			return
		}
		// Connect remote server
		conn, err := net.DialTimeout(req.Network, net.JoinHostPort(req.Host, req.Port), 60*time.Second)
		if err != nil {
			log.Printf("[server] dial error:%v", err)
			return
		}
		// Forward data
		go proxy.ForwardClient(wsConn, conn)
		go proxy.ForwardRemote(wsConn, conn)

	})

	http.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {
		ip := req.Header.Get("X-Forwarded-For")
		if "" == ip {
			ip = strings.Split(req.RemoteAddr, ":")[0]
		}
		resp := fmt.Sprintf("Hello,%v !", ip)
		io.WriteString(w, resp)
	})

	http.ListenAndServe(config.ServerAddr, nil)
}
