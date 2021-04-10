package client

import (
	"io"
	"log"
	"net"

	"github.com/net-byte/opensocks/common/constant"
	"github.com/net-byte/opensocks/config"
	"github.com/net-byte/opensocks/proxy"
)

//Start starts server
func Start(config config.Config) {
	config.Init()
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
	buf := make([]byte, constant.BufferSize)
	//read the version
	n, err := conn.Read(buf[0:])
	if err != nil || err == io.EOF {
		return
	}
	b := buf[0:n]
	if b[0] != constant.Socks5Version {
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
	switch b[1] {
	case constant.ConnectCommand:
		proxy.TCPProxy(conn, config, b)
		return
	case constant.AssociateCommand:
		proxy.UDPProxy(conn, config)
		return
	case constant.BindCommand:
		proxy.Response(conn, constant.CommandNotSupported)
		return
	default:
		proxy.Response(conn, constant.CommandNotSupported)
		return
	}
}
