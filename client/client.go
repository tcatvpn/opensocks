package client

import (
	"github.com/net-byte/opensocks/config"
	"github.com/net-byte/opensocks/proxy"
)

//Start client
func Start(config config.Config) {
	// start udp server
	udpServer := &proxy.UDPServer{Config: config}
	udpConn := udpServer.Start()
	// start tcp server
	tcpServer := &proxy.TCPServer{Config: config, Tproxy: &proxy.TCPProxy{Config: config}, Uproxy: &proxy.UDPProxy{Config: config}, UDPConn: udpConn}
	tcpServer.Start()
}
