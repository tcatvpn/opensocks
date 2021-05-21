package proxy

import (
	"net"
	"strconv"

	"github.com/net-byte/opensocks/common/constant"
	"github.com/net-byte/opensocks/common/iputil"
	"github.com/net-byte/opensocks/config"
)

func TCPProxy(conn net.Conn, config config.Config, data []byte) {
	host, port := getAddr(data)
	if host == "" || port == "" {
		return
	}
	if config.Bypass && iputil.IsPrivateIP(net.ParseIP(host)) {
		DirectProxy(conn, host, port, config)
		return
	}

	wsConn := ConnectWS("tcp", host, port, config)
	if wsConn == nil {
		Response(conn, constant.ConnectionRefused)
		return
	}

	Response(conn, constant.SuccessReply)
	go TCPToWS(wsConn, conn)
	go WSToTCP(wsConn, conn)

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
	case constant.Ipv4Address:
		host = net.IPv4(b[4], b[5], b[6], b[7]).String()
	case constant.FqdnAddress:
		host = string(b[5 : len-2])
	case constant.Ipv6Address:
		host = net.IP(b[4:19]).String()
	default:
		return "", ""
	}
	port = strconv.Itoa(int(b[len-2])<<8 | int(b[len-1]))
	return host, port
}
