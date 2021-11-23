package client

import (
	"io"
	"log"
	"net"

	"github.com/net-byte/opensocks/common/constant"
	"github.com/net-byte/opensocks/config"
	"github.com/net-byte/opensocks/proxy"
)

//Start client
func Start(config config.Config) {
	log.Printf("opensocks [tcp] client started on %s", config.LocalAddr)
	udpConn := handleUDP(config)
	handleTCP(config, udpConn)
}

func handleTCP(config config.Config, udpConn *net.UDPConn) {
	l, err := net.Listen("tcp", config.LocalAddr)
	if err != nil {
		log.Panicf("[tcp] failed to listen tcp %v", err)
	}
	for {
		tcpConn, err := l.Accept()
		if err != nil {
			continue
		}
		go tcpHandler(tcpConn, udpConn, config)
	}
}

func handleUDP(config config.Config) *net.UDPConn {
	udpRelay := &proxy.UDPRelay{Config: config}
	return udpRelay.Start()
}

func tcpHandler(tcpConn net.Conn, udpConn *net.UDPConn, config config.Config) {
	buf := make([]byte, constant.BufferSize)
	//read version
	n, err := tcpConn.Read(buf[0:])
	if err != nil || err == io.EOF {
		return
	}
	b := buf[0:n]
	if b[0] != constant.Socks5Version {
		return
	}
	//no auth
	proxy.ResponseNoAuth(tcpConn)
	//read cmd
	n, err = tcpConn.Read(buf[0:])
	if err != nil || err == io.EOF {
		return
	}
	b = buf[0:n]
	switch b[1] {
	case constant.ConnectCommand:
		proxy.TCPProxy(tcpConn, config, b)
		return
	case constant.AssociateCommand:
		proxy.UDPProxy(tcpConn, udpConn, config)
		return
	case constant.BindCommand:
		proxy.ResponseTCP(tcpConn, constant.CommandNotSupported)
		return
	default:
		proxy.ResponseTCP(tcpConn, constant.CommandNotSupported)
		return
	}
}
