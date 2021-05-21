package proxy

import (
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/gorilla/websocket"
	"github.com/net-byte/opensocks/common/cipher"
	"github.com/net-byte/opensocks/common/constant"
	"github.com/net-byte/opensocks/config"
	"github.com/net-byte/opensocks/counter"
)

func ConnectWS(network string, host string, port string, config config.Config) *websocket.Conn {
	scheme := "ws"
	if config.Wss {
		scheme = "wss"
	}
	u := url.URL{Scheme: scheme, Host: config.ServerAddr, Path: constant.WSPath}
	header := make(http.Header)
	header.Set("user-agent", constant.UserAgent)
	c, _, err := websocket.DefaultDialer.Dial(u.String(), header)
	if err != nil {
		log.Printf("[client] failed to dial websocket %v", err)
		return nil
	}
	req := &RequestAddr{}
	req.Network = network
	req.Host = host
	req.Port = port
	req.Key = config.Key
	req.Timestamp = strconv.FormatInt(time.Now().Unix(), 10)
	req.Random = cipher.Random()
	data, err := req.MarshalBinary()
	if err != nil {
		log.Printf("[client] failed to marshal binary %v", err)
		return nil
	}
	data = cipher.Encrypt(data)
	c.WriteMessage(websocket.BinaryMessage, data)
	return c
}

func CloseWS(wsConn *websocket.Conn) {
	wsConn.WriteControl(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""), time.Now().Add(time.Second*5))
	wsConn.Close()
}

func TCPToWS(wsConn *websocket.Conn, conn net.Conn) {
	defer CloseWS(wsConn)
	defer conn.Close()
	buffer := make([]byte, constant.BufferSize)
	for {
		conn.SetReadDeadline(time.Now().Add(time.Duration(constant.Timeout) * time.Second))
		n, err := conn.Read(buffer)
		if err != nil || err == io.EOF || n == 0 {
			break
		}
		b := cipher.Encrypt(buffer[:n])
		wsConn.WriteMessage(websocket.BinaryMessage, b)
		counter.IncrWriteByte(n)
	}
}

func WSToTCP(wsConn *websocket.Conn, conn net.Conn) {
	defer CloseWS(wsConn)
	defer conn.Close()
	for {
		wsConn.SetReadDeadline(time.Now().Add(time.Duration(constant.Timeout) * time.Second))
		_, buffer, err := wsConn.ReadMessage()
		n := len(buffer)
		if err != nil || err == io.EOF || n == 0 {
			break
		}
		buffer = cipher.Decrypt(buffer)
		conn.Write(buffer[:])
		counter.IncrReadByte(n)
	}
}
