package proxy

import (
	"io"
	"log"
	"net"
	"time"

	"github.com/net-byte/opensocks/common/enum"
	"github.com/net-byte/opensocks/config"
)

// Direct is a direct proxy
func directProxy(conn net.Conn, host string, port string, config config.Config) {
	rconn, err := net.DialTimeout("tcp", net.JoinHostPort(host, port), time.Duration(enum.Timeout)*time.Second)
	if err != nil {
		log.Printf("[tcp] failed to dial tcp %v", err)
		resp(conn, enum.ConnectionRefused)
		return
	}

	resp(conn, enum.SuccessReply)
	go copy(rconn, conn)
	copy(conn, rconn)
}

// Copy copies data from src to dst
func copy(to io.WriteCloser, from io.ReadCloser) {
	defer to.Close()
	defer from.Close()
	io.Copy(to, from)
}
