package proxy

import (
	"bytes"
	"io"
	"log"
	"net"
	"strconv"
	"sync"
	"time"

	"github.com/gobwas/ws/wsutil"
	"github.com/net-byte/opensocks/common/cipher"
	"github.com/net-byte/opensocks/common/constant"
	"github.com/net-byte/opensocks/config"
	"github.com/net-byte/opensocks/counter"
)

func UDPProxy(tcpConn net.Conn, udpConn *net.UDPConn, config config.Config) {
	defer tcpConn.Close()
	if udpConn == nil {
		log.Printf("[udp] failed to start udp server on %v", config.LocalAddr)
		return
	}
	bindAddr, _ := net.ResolveUDPAddr("udp", udpConn.LocalAddr().String())
	// response to client
	ResponseUDP(tcpConn, bindAddr)
	// keep client alive
	done := make(chan bool)
	go keepTCPAlive(tcpConn.(*net.TCPConn), done)
	<-done
}

func keepTCPAlive(tcpConn *net.TCPConn, done chan<- bool) {
	tcpConn.SetKeepAlive(true)
	buf := make([]byte, constant.BufferSize)
	for {
		_, err := tcpConn.Read(buf[0:])
		if err != nil {
			break
		}
	}
	done <- true
}

// UDP Relay
type UDPRelay struct {
	UDPConn   *net.UDPConn
	Config    config.Config
	headerMap sync.Map
	wsconnMap sync.Map
}

func (u *UDPRelay) Start() *net.UDPConn {
	udpAddr, _ := net.ResolveUDPAddr("udp", u.Config.LocalAddr)
	udpConn, err := net.ListenUDP("udp", udpAddr)
	if err != nil {
		log.Printf("[udp] failed to listen udp %v", err)
		return nil
	}
	u.UDPConn = udpConn
	go u.toRemote()
	log.Printf("opensocks [udp] client started on %v", u.Config.LocalAddr)
	return u.UDPConn
}

func (proxy *UDPRelay) toRemote() {
	defer proxy.UDPConn.Close()
	buf := make([]byte, constant.BufferSize)
	for {
		proxy.UDPConn.SetReadDeadline(time.Now().Add(time.Duration(constant.Timeout) * time.Second))
		n, cliAddr, err := proxy.UDPConn.ReadFromUDP(buf)
		if err != nil || err == io.EOF || n == 0 {
			continue
		}
		b := buf[:n]
		dstAddr, header, data := proxy.getAddr(b)
		if dstAddr == nil || header == nil || data == nil {
			continue
		}
		key := cliAddr.String()
		var wsconn net.Conn
		if value, ok := proxy.wsconnMap.Load(key); ok {
			wsconn = value.(net.Conn)
		} else {
			wsconn = connectServer("udp", dstAddr.IP.String(), strconv.Itoa(dstAddr.Port), proxy.Config)
			if wsconn == nil {
				continue
			}
			proxy.wsconnMap.Store(key, wsconn)
			proxy.headerMap.Store(key, header)
			go proxy.toLocal(wsconn, cliAddr)
		}
		if proxy.Config.Obfuscate {
			data = cipher.XOR(data)
		}
		counter.IncrWrittenBytes(n)
		wsutil.WriteClientBinary(wsconn, data)
	}
}

func (proxy *UDPRelay) toLocal(wsconn net.Conn, cliAddr *net.UDPAddr) {
	defer wsconn.Close()
	key := cliAddr.String()
	for {
		wsconn.SetReadDeadline(time.Now().Add(time.Duration(constant.Timeout) * time.Second))
		buffer, err := wsutil.ReadServerBinary(wsconn)
		n := len(buffer)
		if err != nil || err == io.EOF || n == 0 {
			break
		}
		if header, ok := proxy.headerMap.Load(key); ok {
			if proxy.Config.Obfuscate {
				buffer = cipher.XOR(buffer)
			}
			var data bytes.Buffer
			data.Write(header.([]byte))
			data.Write(buffer)
			counter.IncrReadBytes(n)
			proxy.UDPConn.WriteToUDP(data.Bytes(), cliAddr)
		}
	}
	proxy.headerMap.Delete(key)
	proxy.wsconnMap.Delete(key)
}

func (proxy *UDPRelay) getAddr(b []byte) (dstAddr *net.UDPAddr, header []byte, data []byte) {
	/*
	   +----+------+------+----------+----------+----------+
	   |RSV | FRAG | ATYP | DST.ADDR | DST.PORT |   DATA   |
	   +----+------+------+----------+----------+----------+
	   |  2 |   1  |   1  | Variable |     2    | Variable |
	   +----+------+------+----------+----------+----------+
	*/
	if b[2] != 0x00 {
		log.Printf("[udp] not support frag %v", b[2])
		return nil, nil, nil
	}
	switch b[3] {
	case constant.Ipv4Address:
		dstAddr = &net.UDPAddr{
			IP:   net.IPv4(b[4], b[5], b[6], b[7]),
			Port: int(b[8])<<8 | int(b[9]),
		}
		header = b[0:10]
		data = b[10:]
	case constant.FqdnAddress:
		domainLength := int(b[4])
		domain := string(b[5 : 5+domainLength])
		ipAddr, err := net.ResolveIPAddr("ip", domain)
		if err != nil {
			log.Printf("[udp] failed to resolve dns %s:%v", domain, err)
			return nil, nil, nil
		}
		dstAddr = &net.UDPAddr{
			IP:   ipAddr.IP,
			Port: int(b[5+domainLength])<<8 | int(b[6+domainLength]),
		}
		header = b[0 : 7+domainLength]
		data = b[7+domainLength:]
	case constant.Ipv6Address:
		{
			dstAddr = &net.UDPAddr{
				IP:   net.IP(b[4:19]),
				Port: int(b[20])<<8 | int(b[21]),
			}
			header = b[0:22]
			data = b[22:]
		}
	default:
		return nil, nil, nil
	}
	return dstAddr, header, data
}
