package proxy

import (
	"bytes"
	"io"
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
	"github.com/net-byte/opensocks/proto"
)

type UDPProxy struct {
	Config config.Config
}

func (u *UDPProxy) Proxy(tcpConn net.Conn, udpConn *net.UDPConn) {
	defer tcpConn.Close()
	if udpConn == nil {
		log.Printf("[udp] failed to start udp server on %v", u.Config.LocalAddr)
		return
	}
	bindAddr, _ := net.ResolveUDPAddr("udp", udpConn.LocalAddr().String())
	// response to client
	ResponseUDP(tcpConn, bindAddr)
	// keep client alive
	done := make(chan bool)
	go u.keepTCPAlive(tcpConn.(*net.TCPConn), done)
	<-done
}

func (u *UDPProxy) keepTCPAlive(tcpConn *net.TCPConn, done chan<- bool) {
	tcpConn.SetKeepAlive(true)
	buf := make([]byte, enum.BufferSize)
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
	streamMap sync.Map
	Session   *yamux.Session
	Lock      sync.Mutex
}

func (u *UDPRelay) Start() *net.UDPConn {
	udpAddr, _ := net.ResolveUDPAddr("udp", u.Config.LocalAddr)
	udpConn, err := net.ListenUDP("udp", udpAddr)
	if err != nil {
		log.Printf("[udp] failed to listen udp %v", err)
		return nil
	}
	u.UDPConn = udpConn
	go u.toServer()
	log.Printf("opensocks [udp] client started on %v", u.Config.LocalAddr)
	return u.UDPConn
}

func (u *UDPRelay) toServer() {
	defer u.UDPConn.Close()
	buf := u.Config.BytePool.Get()
	defer u.Config.BytePool.Put(buf)
	for {
		u.UDPConn.SetReadDeadline(time.Now().Add(time.Duration(enum.Timeout) * time.Second))
		n, cliAddr, err := u.UDPConn.ReadFromUDP(buf)
		if err != nil || err == io.EOF || n == 0 {
			continue
		}
		b := buf[:n]
		dstAddr, header, data := u.getAddr(b)
		if dstAddr == nil || header == nil || data == nil {
			continue
		}
		key := cliAddr.String()
		var stream net.Conn
		if value, ok := u.streamMap.Load(key); !ok {
			if u.Session == nil {
				u.Lock.Lock()
				var err error
				wsconn := connectServer(u.Config)
				if wsconn == nil {
					u.Lock.Unlock()
					continue
				}
				u.Session, err = yamux.Client(wsconn, nil)
				if err != nil || u.Session == nil {
					log.Println(err)
					u.Lock.Unlock()
					continue
				}
				u.Lock.Unlock()
			}
			stream, err = u.Session.Open()
			if err != nil {
				u.Session = nil
				log.Println(err)
				continue
			}
			ok := handshake(stream, "udp", dstAddr.IP.String(), strconv.Itoa(dstAddr.Port), u.Config.Key, u.Config.Obfs)
			if !ok {
				u.Session = nil
				log.Println("[udp] failed to handshake")
				continue
			}
			u.streamMap.Store(key, stream)
			u.headerMap.Store(key, header)
			go u.toClient(stream, cliAddr)
		} else {
			stream = value.(net.Conn)
		}
		if u.Config.Obfs {
			data = cipher.XOR(data)
		}
		edate, err := proto.Encode(data)
		if err != nil {
			break
		}
		stream.Write(edate)
		counter.IncrWrittenBytes(n)
	}
}

func (u *UDPRelay) toClient(stream net.Conn, cliAddr *net.UDPAddr) {
	key := cliAddr.String()
	buffer := u.Config.BytePool.Get()
	defer u.Config.BytePool.Put(buffer)
	defer stream.Close()
	for {
		stream.SetReadDeadline(time.Now().Add(time.Duration(enum.Timeout) * time.Second))
		n, err := stream.Read(buffer)
		if err != nil || n == 0 {
			break
		}
		if header, ok := u.headerMap.Load(key); ok {
			b := buffer[:n]
			if u.Config.Obfs {
				b = cipher.XOR(b)
			}
			var data bytes.Buffer
			data.Write(header.([]byte))
			data.Write(b)
			_, err = u.UDPConn.WriteToUDP(data.Bytes(), cliAddr)
			if err != nil {
				break
			}
			counter.IncrReadBytes(n)
		}
	}
	u.headerMap.Delete(key)
	u.streamMap.Delete(key)
}

func (u *UDPRelay) getAddr(b []byte) (dstAddr *net.UDPAddr, header []byte, data []byte) {
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
	case enum.Ipv4Address:
		dstAddr = &net.UDPAddr{
			IP:   net.IPv4(b[4], b[5], b[6], b[7]),
			Port: int(b[8])<<8 | int(b[9]),
		}
		header = b[0:10]
		data = b[10:]
	case enum.FqdnAddress:
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
	case enum.Ipv6Address:
		{
			dstAddr = &net.UDPAddr{
				IP:   net.IP(b[4:20]),
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
