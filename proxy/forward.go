package proxy

import (
	"io"
	"log"
	"net"
	"net/url"
	"strconv"
	"time"

	"github.com/gorilla/websocket"
	"github.com/net-byte/opensocks/config"
	"github.com/net-byte/opensocks/counter"
	"github.com/net-byte/opensocks/utils"
)

func ConnectWS(network string, host string, port string, config config.Config) *websocket.Conn {
	scheme := "ws"
	if config.Wss {
		scheme = "wss"
	}
	u := url.URL{Scheme: scheme, Host: config.ServerAddr, Path: WSPath}
	c, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
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
	utils.Encrypt(&data, config.Key)
	c.WriteMessage(websocket.BinaryMessage, data)
	return c
}

func ForwardRemote(wsConn *websocket.Conn, conn net.Conn, config config.Config) {
	defer wsConn.Close()
	defer conn.Close()
	buffer := make([]byte, BufferSize)
	for {
		conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		n, err := conn.Read(buffer)
		if err != nil || err == io.EOF || n == 0 {
			break
		}
		b := buffer[:n]
		utils.Encrypt(&b, config.Key)
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
		utils.Decrypt(&buffer, config.Key)
		conn.Write(buffer[:])
		counter.IncrReadByte(n)
	}
}
