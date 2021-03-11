package server

import (
	"io"
	"log"
	"net"
	"net/http"
	"strings"
	"syscall"
	"time"

	"github.com/gorilla/websocket"
	"github.com/net-byte/opensocks/config"
)

const bufferSize int = 4096

var upgrader = websocket.Upgrader{
	ReadBufferSize:    bufferSize,
	WriteBufferSize:   bufferSize,
	EnableCompression: true,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

// Start starts server
func Start(config config.Config) {
	var rLimit syscall.Rlimit
	if err := syscall.Getrlimit(syscall.RLIMIT_NOFILE, &rLimit); err != nil {
		log.Panic(err)
	}
	rLimit.Cur = rLimit.Max
	if err := syscall.Setrlimit(syscall.RLIMIT_NOFILE, &rLimit); err != nil {
		log.Panic(err)
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
		// Read : host + "||" + port + "||" + username + "||" + password + "||" + network + "||" + UnixNano
		addr := string(buffer)
		log.Printf("remote addr:%v", addr)
		s := strings.Split(addr, "||")
		if len(s) < 5 {
			return
		}
		host, port, username, password := s[0], s[1], s[2], s[3]
		if !(config.Username == username && config.Password == password) {
			log.Println("Error username or password !")
			return
		}
		network := "tcp"
		if s[4] == "tcp" || s[4] == "udp" {
			network = s[4]
		}
		// Connect remote server
		conn, err := net.DialTimeout(network, net.JoinHostPort(host, port), 60*time.Second)
		if err != nil {
			log.Println(err)
			return
		}
		// Forward data
		go forwardClient(wsConn, conn)
		go forwardRemote(wsConn, conn)

	})

	http.ListenAndServe(config.ServerAddr, nil)
}

func forwardClient(wsConn *websocket.Conn, conn net.Conn) {
	defer wsConn.Close()
	defer conn.Close()
	for {
		buffer := make([]byte, bufferSize)
		n, err := conn.Read(buffer)
		if err != nil || err == io.EOF {
			break
		}
		wsConn.WriteMessage(websocket.BinaryMessage, buffer[:n])
	}
}

func forwardRemote(wsConn *websocket.Conn, conn net.Conn) {
	defer wsConn.Close()
	defer conn.Close()
	for {
		_, buffer, err := wsConn.ReadMessage()
		if err != nil || err == io.EOF {
			break
		}
		conn.Write(buffer[:])
	}
}
