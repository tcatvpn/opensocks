package proxy

import (
	"log"
	"net"
	"strconv"
	"sync"
	"time"

	"github.com/hashicorp/yamux"
	"github.com/net-byte/opensocks/common/cipher"
	"github.com/net-byte/opensocks/common/enum"
	"github.com/net-byte/opensocks/config"
	"github.com/net-byte/opensocks/counter"
)

type TCPProxy struct {
	Config  config.Config
	Session *yamux.Session
	Lock    sync.Mutex
}

func (t *TCPProxy) Proxy(conn net.Conn, data []byte) {
	host, port := t.getAddr(data)
	if host == "" || port == "" {
		return
	}
	// bypass private ip
	if t.Config.Bypass && net.ParseIP(host) != nil && net.ParseIP(host).IsPrivate() {
		DirectProxy(conn, host, port, t.Config)
		return
	}
	if t.Session == nil {
		t.Lock.Lock()
		var err error
		wsconn := connectServer(t.Config)
		if wsconn == nil {
			ResponseTCP(conn, enum.ConnectionRefused)
			return
		}
		t.Session, err = yamux.Client(wsconn, nil)
		if err != nil || t.Session == nil {
			log.Println(err)
			ResponseTCP(conn, enum.ConnectionRefused)
			return
		}
		t.Lock.Unlock()
	}
	stream, err := t.Session.Open()
	if err != nil {
		t.Session = nil
		log.Println(err)
		ResponseTCP(conn, enum.ConnectionRefused)
		return
	}
	ok := handshake(stream, "tcp", host, port, t.Config.Key, t.Config.Obfs)
	if !ok {
		t.Session = nil
		log.Println("[tcp] failed to handshake")
		ResponseTCP(conn, enum.ConnectionRefused)
		return
	}
	ResponseTCP(conn, enum.SuccessReply)
	go t.toServer(stream, conn)
	t.toClient(stream, conn)
}

func (t *TCPProxy) toServer(stream net.Conn, tcpconn net.Conn) {
	defer stream.Close()
	defer tcpconn.Close()
	buffer := t.Config.BytePool.Get()
	defer t.Config.BytePool.Put(buffer)
	for {
		tcpconn.SetReadDeadline(time.Now().Add(time.Duration(enum.Timeout) * time.Second))
		n, err := tcpconn.Read(buffer)
		if err != nil || n == 0 {
			break
		}
		b := buffer[:n]
		if t.Config.Obfs {
			b = cipher.XOR(b)
		}
		_, err = stream.Write(b)
		if err != nil {
			break
		}
		counter.IncrWrittenBytes(n)
	}
}

func (t *TCPProxy) toClient(stream net.Conn, tcpconn net.Conn) {
	defer stream.Close()
	defer tcpconn.Close()
	buffer := t.Config.BytePool.Get()
	defer t.Config.BytePool.Put(buffer)
	for {
		stream.SetReadDeadline(time.Now().Add(time.Duration(enum.Timeout) * time.Second))
		n, err := stream.Read(buffer)
		if err != nil || n == 0 {
			break
		}
		b := buffer[:n]
		if t.Config.Obfs {
			b = cipher.XOR(b)
		}
		_, err = tcpconn.Write(b)
		if err != nil {
			break
		}
		counter.IncrReadBytes(n)
	}
}

func (t *TCPProxy) getAddr(b []byte) (host string, port string) {
	/**
	  +----+-----+-------+------+----------+----------+
	  |VER | CMD |  RSV  | ATYP | DST.ADDR | DST.PORT |
	  +----+-----+-------+------+----------+----------+
	  | 1  |  1  | X'00' |  1   | Variable |    2     |
	  +----+-----+-------+------+----------+----------+
	*/
	len := len(b)
	switch b[3] {
	case enum.Ipv4Address:
		host = net.IPv4(b[4], b[5], b[6], b[7]).String()
	case enum.FqdnAddress:
		host = string(b[5 : len-2])
	case enum.Ipv6Address:
		host = net.IP(b[4:20]).String()
	default:
		return "", ""
	}
	port = strconv.Itoa(int(b[len-2])<<8 | int(b[len-1]))
	return host, port
}
