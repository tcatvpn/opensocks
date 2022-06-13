package proxy

import (
	"net"

	"github.com/net-byte/opensocks/common/pool"
	"github.com/net-byte/opensocks/config"
)

// The UDPProxy struct
type UDPProxy struct {
	Config config.Config
}

// Proxy handles the udp connection
func (u *UDPProxy) Proxy(tcpConn net.Conn, udpConn *net.UDPConn) {
	defer tcpConn.Close()
	udpAddr, _ := net.ResolveUDPAddr("udp", udpConn.LocalAddr().String())
	respSuccess(tcpConn, udpAddr.IP.To4(), udpAddr.Port)
	// keep tcp conn alive
	done := make(chan bool)
	go u.keepTCPAlive(tcpConn.(*net.TCPConn), done)
	<-done
}

// keepTCPAlive keeps the tcp connection alive
func (u *UDPProxy) keepTCPAlive(tcpConn *net.TCPConn, done chan<- bool) {
	tcpConn.SetKeepAlive(true)
	buf := pool.BytePool.Get()
	defer pool.BytePool.Put(buf)
	for {
		_, err := tcpConn.Read(buf[0:])
		if err != nil {
			break
		}
	}
	done <- true
}
