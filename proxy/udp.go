package proxy

import (
	"net"

	"github.com/net-byte/opensocks/config"
)

type UDPProxy struct {
	Config config.Config
}

func (u *UDPProxy) Proxy(tcpConn net.Conn, udpConn *net.UDPConn) {
	defer tcpConn.Close()
	udpAddr, _ := net.ResolveUDPAddr("udp", udpConn.LocalAddr().String())
	respSuccess(tcpConn, udpAddr.IP.To4(), udpAddr.Port)
	// keep tcp conn alive
	done := make(chan bool)
	go u.keepTCPAlive(tcpConn.(*net.TCPConn), done)
	<-done
}

func (u *UDPProxy) keepTCPAlive(tcpConn *net.TCPConn, done chan<- bool) {
	tcpConn.SetKeepAlive(true)
	buf := u.Config.BytePool.Get()
	defer u.Config.BytePool.Put(buf)
	for {
		_, err := tcpConn.Read(buf[0:])
		if err != nil {
			break
		}
	}
	done <- true
}
