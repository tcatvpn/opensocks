package client

import (
	"log"
	"net"

	"github.com/net-byte/opensocks/common/enum"
	"github.com/net-byte/opensocks/config"
	"github.com/net-byte/opensocks/proxy"
)

//Start client
func Start(config config.Config) {
	log.Printf("opensocks [tcp] client started on %s", config.LocalAddr)
	tproxy := &proxy.TCPProxy{Config: config}
	uproxy := &proxy.UDPProxy{Config: config}
	cli := &Client{config: config, tproxy: tproxy, uproxy: uproxy}
	cli.start()
}

type Client struct {
	config config.Config
	tproxy *proxy.TCPProxy
	uproxy *proxy.UDPProxy
}

func (c *Client) start() {
	udpRelay := &proxy.UDPRelay{Config: c.config}
	udpConn := udpRelay.Start()
	l, err := net.Listen("tcp", c.config.LocalAddr)
	if err != nil {
		log.Panicf("[tcp] failed to listen tcp %v", err)
	}
	for {
		tcpConn, err := l.Accept()
		if err != nil {
			continue
		}
		go c.handler(tcpConn, udpConn)
	}
}

func (c *Client) handler(tcpConn net.Conn, udpConn *net.UDPConn) {
	buf := c.config.BytePool.Get()
	defer c.config.BytePool.Put(buf)
	//read version
	n, err := tcpConn.Read(buf[0:])
	if err != nil || n == 0 {
		return
	}
	b := buf[0:n]
	if b[0] != enum.Socks5Version {
		proxy.ResponseTCP(tcpConn, enum.ConnectionRefused)
		return
	}
	//no auth
	proxy.ResponseNoAuth(tcpConn)
	//read cmd
	n, err = tcpConn.Read(buf[0:])
	if err != nil || n == 0 {
		return
	}
	b = buf[0:n]
	switch b[1] {
	case enum.ConnectCommand:
		c.tproxy.Proxy(tcpConn, b)
		return
	case enum.AssociateCommand:
		c.uproxy.Proxy(tcpConn, udpConn)
		return
	case enum.BindCommand:
		proxy.ResponseTCP(tcpConn, enum.CommandNotSupported)
		return
	default:
		proxy.ResponseTCP(tcpConn, enum.CommandNotSupported)
		return
	}
}
