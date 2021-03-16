package proxy

import (
	"bytes"
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
	// Send host addr to proxy server side
	var data bytes.Buffer
	data.WriteString(host)
	data.WriteString("||")
	data.WriteString(port)
	data.WriteString("||")
	data.WriteString(config.Username)
	data.WriteString("||")
	data.WriteString(config.Password)
	data.WriteString("||")
	data.WriteString(network)
	data.WriteString("||")
	data.WriteString(strconv.FormatInt(time.Now().UnixNano(), 10))
	c.WriteMessage(websocket.BinaryMessage, data.Bytes())
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
