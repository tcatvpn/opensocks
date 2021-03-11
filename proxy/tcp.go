package proxy

import (
	"net"

	"github.com/net-byte/opensocks/config"
)

func TcpProxy(conn net.Conn, addrType uint8, host string, port string, config config.Config) {
	if host == "" || port == "" {
		return
	}

	//log.Printf("remote proxy %s", host)
	wsConn := ConnectWS("tcp", host, port, config)
	if wsConn == nil {
		conn.Write([]byte{Socks5Version, ConnectionRefused, 0x00, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00})
		return
	}

	Response(conn, SuccessReply)
	go ForwardRemote(wsConn, conn)
	go ForwardClient(wsConn, conn)

}
