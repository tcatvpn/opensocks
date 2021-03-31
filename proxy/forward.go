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
	header.Set("user-agent", "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/88.0.4324.182 Safari/537.36")
	c, _, err := websocket.DefaultDialer.Dial(u.String(), header)
	if err != nil {
		log.Printf("[client] websocket dial error:%v", err)
		return nil
	}
	req := &RequestAddr{}
	req.Network = network
	req.Host = host
	req.Port = port
	req.Username = config.Username
	req.Password = config.Password
	req.Timestamp = strconv.FormatInt(time.Now().UnixNano(), 10)
	data, err := req.MarshalBinary()
	if err != nil {
		log.Printf("[client] MarshalBinary error:%v", err)
		return nil
	}
	cipher.Encrypt(&data, config.Key)
	c.WriteMessage(websocket.BinaryMessage, data)
	return c
}

func ForwardRemote(wsConn *websocket.Conn, conn net.Conn, config config.Config) {
	defer wsConn.Close()
	defer conn.Close()
	buffer := make([]byte, constant.BufferSize)
	for {
		conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		n, err := conn.Read(buffer)
		if err != nil || err == io.EOF || n == 0 {
			break
		}
		b := buffer[:n]
		cipher.Encrypt(&b, config.Key)
		wsConn.WriteMessage(websocket.BinaryMessage, b)
		counter.IncrWriteByte(n)
	}
}

func ForwardClient(wsConn *websocket.Conn, conn net.Conn, config config.Config) {
	defer wsConn.Close()
	defer conn.Close()
	for {
		wsConn.SetReadDeadline(time.Now().Add(60 * time.Second))
		_, buffer, err := wsConn.ReadMessage()
		n := len(buffer)
		if err != nil || err == io.EOF || n == 0 {
			break
		}
		cipher.Decrypt(&buffer, config.Key)
		conn.Write(buffer[:])
		counter.IncrReadByte(n)
	}
}
