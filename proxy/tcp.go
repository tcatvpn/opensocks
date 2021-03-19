package proxy

import (
	"net"
	"strconv"

	"github.com/net-byte/opensocks/config"
	"github.com/net-byte/opensocks/utils"
)

func TCPProxy(conn net.Conn, config config.Config, data []byte) {
	host, port := getAddr(data)
	if host == "" || port == "" {
		return
	}
	if config.Bypass && utils.IsPrivateIP(net.ParseIP(host)) {
		DirectProxy(conn, host, port, config)
		return
	}

	wsConn := ConnectWS("tcp", host, port, config)
	if wsConn == nil {
		Response(conn, ConnectionRefused)
		return
	}

	Response(conn, SuccessReply)
	go ForwardRemote(wsConn, conn, config)
	go ForwardClient(wsConn, conn, config)

}

func getAddr(b []byte) (host string, port string) {
	/**
	  +----+-----+-------+------+----------+----------+
	  |VER | CMD |  RSV  | ATYP | DST.ADDR | DST.PORT |
	  +----+-----+-------+------+----------+----------+
	  | 1  |  1  | X'00' |  1   | Variable |    2     |
	  +----+-----+-------+------+----------+----------+
	*/
	len := len(b)
	switch b[3] {
	case Ipv4Address:
		host = net.IPv4(b[4], b[5], b[6], b[7]).String()
		break
	case FqdnAddress:
		host = string(b[5 : len-2])
		break
	case Ipv6Address:
		host = net.IP(b[4:19]).String()
		break
	default:
		return "", ""
	}
	port = strconv.Itoa(int(b[len-2])<<8 | int(b[len-1]))
	return host, port
}
