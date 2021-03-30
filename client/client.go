package client

import (
	"io"
	"log"
	"net"

	"github.com/net-byte/opensocks/common/cipher"
	"github.com/net-byte/opensocks/config"
	"github.com/net-byte/opensocks/proxy"
)

//Start starts server
func Start(config config.Config) {
	config.Key = cipher.CreateHash(config.Username + config.Password)
	log.Printf("opensocks client started on %s", config.LocalAddr)
	l, err := net.Listen("tcp", config.LocalAddr)
	if err != nil {
		log.Panicf("[tcp] listen error:%v", err)
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
	if !checkVersion(conn, b) {
		return
	}
	//no auth
	proxy.ResponseNoAuth(conn)
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
		proxy.TCPProxy(conn, config, b)
		return
	case proxy.AssociateCommand:
		proxy.UDPProxy(conn, config)
		return
	case proxy.BindCommand:
		return
	default:
		return
	}

}

func checkVersion(conn net.Conn, b []byte) bool {
	if b[0] != proxy.Socks5Version {
		return false
	}
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
