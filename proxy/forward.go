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
)

func ConnectWS(network string, host string, port string, config config.Config) *websocket.Conn {
	scheme := "ws"
	if config.Wss {
		scheme = "wss"
	}
	u := url.URL{Scheme: scheme, Host: config.ServerAddr, Path: "/opensocks"}
	c, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		log.Println(err)
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
		log.Println(err)
		return nil
	}
	c.WriteMessage(websocket.BinaryMessage, data)
	return c
}

func ForwardRemote(wsConn *websocket.Conn, conn net.Conn) {
	defer wsConn.Close()
	defer conn.Close()
	buffer := make([]byte, BufferSize)
	for {
		conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		n, err := conn.Read(buffer)
		if err != nil || err == io.EOF || n == 0 {
			break
		}
		wsConn.WriteMessage(websocket.BinaryMessage, buffer[:n])
	}
}

func ForwardClient(wsConn *websocket.Conn, conn net.Conn) {
	defer wsConn.Close()
	defer conn.Close()
	for {
		wsConn.SetReadDeadline(time.Now().Add(60 * time.Second))
		_, buffer, err := wsConn.ReadMessage()
		if err != nil || err == io.EOF || len(buffer) == 0 {
			break
		}
		conn.Write(buffer[:])
	}
}

func Response(conn net.Conn, rep byte) {
	/**
	  +----+-----+-------+------+----------+----------+
	  |VER | REP |  RSV  | ATYP | BND.ADDR | BND.PORT |
	  +----+-----+-------+------+----------+----------+
	  | 1  |  1  | X'00' |  1   | Variable |    2     |
	  +----+-----+-------+------+----------+----------+
	*/
	conn.Write([]byte{Socks5Version, rep, 0x00, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00})
}
