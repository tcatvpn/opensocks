package proxy

import (
	"io"
	"net"
	"strconv"
	"time"

	"github.com/gobwas/ws/wsutil"
	"github.com/net-byte/opensocks/common/cipher"
	"github.com/net-byte/opensocks/common/constant"
	"github.com/net-byte/opensocks/config"
	"github.com/net-byte/opensocks/counter"
)

func TCPProxy(conn net.Conn, config config.Config, data []byte) {
	host, port := getAddr(data)
	if host == "" || port == "" {
		return
	}
	// bypass private ip
	if config.Bypass && net.ParseIP(host) != nil && net.ParseIP(host).IsPrivate() {
		DirectProxy(conn, host, port, config)
		return
	}

	wsconn := connectServer("tcp", host, port, config)
	if wsconn == nil {
		ResponseTCP(conn, constant.ConnectionRefused)
		return
	}

	ResponseTCP(conn, constant.SuccessReply)
	go toRemote(config, wsconn, conn)
	go toLocal(config, wsconn, conn)
}

func toRemote(config config.Config, wsconn net.Conn, tcpconn net.Conn) {
	defer wsconn.Close()
	defer tcpconn.Close()
	buffer := make([]byte, constant.BufferSize)
	for {
		tcpconn.SetReadDeadline(time.Now().Add(time.Duration(constant.Timeout) * time.Second))
		n, err := tcpconn.Read(buffer)
		if err != nil || err == io.EOF || n == 0 {
			break
		}
		var b []byte
		if config.Obfuscate {
			b = cipher.XOR(buffer[:n])
		} else {
			b = buffer[:n]
		}
		counter.IncrWrittenBytes(n)
		wsutil.WriteClientBinary(wsconn, b)
	}
}

func toLocal(config config.Config, wsconn net.Conn, tcpconn net.Conn) {
	defer wsconn.Close()
	defer tcpconn.Close()
	for {
		wsconn.SetReadDeadline(time.Now().Add(time.Duration(constant.Timeout) * time.Second))
		buffer, err := wsutil.ReadServerBinary(wsconn)
		n := len(buffer)
		if err != nil || err == io.EOF || n == 0 {
			break
		}
		if config.Obfuscate {
			buffer = cipher.XOR(buffer)
		}
		counter.IncrReadBytes(n)
		tcpconn.Write(buffer[:])
	}
}

func getAddr(b []byte) (host string, port string) {
	/**
	  +----+-----+-------+------+----------+----------+
	  |VER | CMD |  RSV  | ATYP | DST.ADDR | DST.PORT |
	  +----+-----+-------+------+----------+----------+
	  | 1  |  1  | X'00' |  1   | Variable |    2     |
	  +----+-----+-------+------+----------+----------+
	*/
	len := len(b)
	switch b[3] {
	case constant.Ipv4Address:
		host = net.IPv4(b[4], b[5], b[6], b[7]).String()
	case constant.FqdnAddress:
		host = string(b[5 : len-2])
	case constant.Ipv6Address:
		host = net.IP(b[4:19]).String()
	default:
		return "", ""
	}
	port = strconv.Itoa(int(b[len-2])<<8 | int(b[len-1]))
	return host, port
}
