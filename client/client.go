package client

import (
	"io"
	"log"
	"net"
	"strconv"

	"github.com/net-byte/opensocks/config"
	"github.com/net-byte/opensocks/proxy"
	"github.com/net-byte/opensocks/utils"
)

//Start starts server
func Start(config config.Config) {
	log.Printf("opensocks client started on %s", config.LocalAddr)
	l, err := net.Listen("tcp", config.LocalAddr)
	if err != nil {
		log.Panic(err)
	}
	for {
		conn, err := l.Accept()
		if err != nil {
			continue
		}
		go connHandler(conn, config)
	}
}

func connHandler(conn net.Conn, config config.Config) {
	buf := make([]byte, proxy.BufferSize)
	//read the version
	n, err := conn.Read(buf[0:])
	if err != nil || err == io.EOF {
		return
	}
	b := buf[0:n]
	if !checkVersionAndAuth(conn, b) {
		return
	}
	//read the cmd
	n, err = conn.Read(buf[0:])
	if err != nil || err == io.EOF {
		return
	}
	b = buf[0:n]
	if !checkCmd(conn, b) {
		return
	}

	switch b[1] {
	case proxy.ConnectCommand:
		//read the addr
		host, port, addrType := getAddr(conn, b)
		if config.Bypass && addrType != proxy.FqdnAddress && utils.IsPrivateIP(net.ParseIP(host)) {
			proxy.DirectProxy(conn, host, port, config)
			return
		}
		proxy.TcpProxy(conn, addrType, host, port, config)
		return
	case proxy.AssociateCommand:
		proxy.UdpProxy(conn, config)
		return
	case proxy.BindCommand:
		return
	default:
		return
	}

}

func checkVersionAndAuth(conn net.Conn, b []byte) bool {
	// Only socks5
	if b[0] != proxy.Socks5Version {
		return false
	}
	// No auth
	/**
	  +----+--------+
	  |VER | METHOD |
	  +----+--------+
	  | 1  |   1    |
	  +----+--------+
	*/
	conn.Write([]byte{proxy.Socks5Version, proxy.NoAuth})
	return true
}

func checkCmd(conn net.Conn, b []byte) bool {
	switch b[1] {
	case proxy.ConnectCommand:
		return true
	case proxy.AssociateCommand:
		return true
	case proxy.BindCommand:
		proxy.Response(conn, proxy.CommandNotSupported)
		return false
	default:
		return false
	}
}

func getAddr(conn net.Conn, b []byte) (host string, port string, addrType uint8) {
	/**
	  +----+-----+-------+------+----------+----------+
	  |VER | CMD |  RSV  | ATYP | DST.ADDR | DST.PORT |
	  +----+-----+-------+------+----------+----------+
	  | 1  |  1  | X'00' |  1   | Variable |    2     |
	  +----+-----+-------+------+----------+----------+
	*/
	len := len(b)
	switch b[3] {
	case proxy.Ipv4Address:
		host = net.IPv4(b[4], b[5], b[6], b[7]).String()
		break
	case proxy.FqdnAddress:
		host = string(b[5 : len-2])
		break
	case proxy.Ipv6Address:
		host = net.IP(b[4:19]).String()
		break
	default:
		return "", "", b[3]
	}
	port = strconv.Itoa(int(b[len-2])<<8 | int(b[len-1]))
	return host, port, b[3]
}
