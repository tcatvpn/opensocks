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
	"github.com/net-byte/opensocks/common/osutil"
	"github.com/net-byte/opensocks/config"
	"github.com/net-byte/opensocks/counter"
	"github.com/net-byte/opensocks/proxy"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:    constant.BufferSize,
	WriteBufferSize:   constant.BufferSize,
	EnableCompression: true,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

// Start starts server
func Start(config config.Config) {
	osutil.SetSysMaxLimit()
	log.Printf("opensocks server started on %s", config.ServerAddr)
	http.HandleFunc(constant.WSPath, func(w http.ResponseWriter, r *http.Request) {
		wsConn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		_, buffer, err := wsConn.ReadMessage()
		if err != nil {
			return
		}
		cipher.Decrypt(&buffer, config.Key)
		var req proxy.RequestAddr
		if req.UnmarshalBinary(buffer) != nil {
			log.Printf("[server] unmarshal binary error:%v", err)
			return
		}
		//log.Printf("[server] client request: %v", req)
		reqTime, _ := strconv.ParseInt(req.Timestamp, 10, 64)
		if time.Now().Unix()-reqTime > 30 {
			log.Printf("[server] timestamp expired: %v", reqTime)
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
		go proxy.ForwardClient(wsConn, conn, config)
		go proxy.ForwardRemote(wsConn, conn, config)

	})

	http.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {
		resp := fmt.Sprintf("Hello，世界！")
		io.WriteString(w, resp)
	})

	http.HandleFunc("/ip", func(w http.ResponseWriter, req *http.Request) {
		ip := req.Header.Get("X-Forwarded-For")
		if "" == ip {
			ip = strings.Split(req.RemoteAddr, ":")[0]
		}
		resp := fmt.Sprintf("%v", ip)
		io.WriteString(w, resp)
	})

	http.HandleFunc("/sys", func(w http.ResponseWriter, req *http.Request) {
		resp := fmt.Sprintf("download %v upload %v", bytesize.New(float64(counter.TotalReadByte)).String(), bytesize.New(float64(counter.TotalWriteByte)).String())
		io.WriteString(w, resp)
	})

	http.ListenAndServe(config.ServerAddr, nil)
}

func formatByteSize(size int64) string {
	return bytesize.New(float64(size)).String()
}
