package proxy

import (
	"io"
	"log"
	"net"
	"time"

	"github.com/net-byte/opensocks/common/constant"
	"github.com/net-byte/opensocks/config"
)

func DirectProxy(conn net.Conn, host string, port string, config config.Config) {
	remoteConn, err := net.DialTimeout("tcp", net.JoinHostPort(host, port), time.Duration(constant.Timeout)*time.Second)
	if err != nil {
		log.Printf("[tcp] failed to dial tcp %v", err)
		ResponseTCP(conn, constant.ConnectionRefused)
		return
	}

	ResponseTCP(conn, constant.SuccessReply)
	go copy(remoteConn, conn)
	go copy(conn, remoteConn)
}

func copy(to io.WriteCloser, from io.ReadCloser) {
	defer to.Close()
	defer from.Close()
	io.Copy(to, from)
}
