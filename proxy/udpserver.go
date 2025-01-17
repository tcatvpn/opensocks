package proxy

import (
	"bytes"
	"io"
	"log"
	"net"
	"strconv"
	"sync"

	"github.com/golang/snappy"
	"github.com/net-byte/opensocks/common/cipher"
	"github.com/net-byte/opensocks/common/enum"
	"github.com/net-byte/opensocks/common/pool"
	"github.com/net-byte/opensocks/common/util"
	"github.com/net-byte/opensocks/config"
	"github.com/net-byte/opensocks/counter"
	"github.com/xtaci/smux"
)

// The UDP server struct
type UDPServer struct {
	UDPConn   *net.UDPConn
	Config    config.Config
	headerMap sync.Map
	streamMap sync.Map
	Session   *smux.Session
	Lock      sync.Mutex
}

// Start the UDP server
func (u *UDPServer) Start() *net.UDPConn {
	udpAddr, _ := net.ResolveUDPAddr("udp", u.Config.LocalAddr)
	var err error
	u.UDPConn, err = net.ListenUDP("udp", udpAddr)
	if err != nil {
		log.Panicf("[udp] failed to listen udp %v", err)
	}
	go u.toServer()
	log.Printf("opensocks [udp] client started on %v", u.Config.LocalAddr)
	return u.UDPConn
}

// toServer handle the udp packet from client
func (u *UDPServer) toServer() {
	defer u.UDPConn.Close()
	buf := pool.BytePool.Get()
	defer pool.BytePool.Put(buf)
	for {
		n, cliAddr, err := u.UDPConn.ReadFromUDP(buf)
		if err != nil {
			break
		}
		b := buf[:n]
		dstAddr, header, data := u.getAddr(b)
		if dstAddr == nil || header == nil || data == nil {
			continue
		}
		key := cliAddr.String()
		var stream io.ReadWriteCloser
		if value, ok := u.streamMap.Load(key); !ok {
			u.Lock.Lock()
			if u.Session == nil {
				var err error
				wsconn := connectServer(u.Config)
				if wsconn == nil {
					u.Lock.Unlock()
					continue
				}
				smuxConfig := smux.DefaultConfig()
				smuxConfig.Version = enum.SmuxVer
				smuxConfig.MaxReceiveBuffer = enum.SmuxBuf
				smuxConfig.MaxStreamBuffer = enum.StreamBuf
				u.Session, err = smux.Client(wsconn, smuxConfig)
				if err != nil || u.Session == nil {
					log.Println(err)
					u.Lock.Unlock()
					continue
				}
			}
			u.Lock.Unlock()
			stream, err = u.Session.Open()
			if err != nil {
				u.Session = nil
				util.PrintLog(u.Config.Verbose, "failed to open session:%v", err)
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
			stream = value.(io.ReadWriteCloser)
		}
		if u.Config.Obfs {
			data = cipher.XOR(data)
		}
		if u.Config.Compress {
			data = snappy.Encode(nil, data)
		}
		stream.Write(data)
		counter.IncrWrittenBytes(n)
	}
}

// toClient handle the udp packet from server
func (u *UDPServer) toClient(stream io.ReadWriteCloser, cliAddr *net.UDPAddr) {
	key := cliAddr.String()
	buffer := pool.BytePool.Get()
	defer pool.BytePool.Put(buffer)
	defer stream.Close()
	for {
		n, err := stream.Read(buffer)
		if err != nil {
			break
		}
		if header, ok := u.headerMap.Load(key); ok {
			b := buffer[:n]
			if u.Config.Compress {
				b, err = snappy.Decode(nil, b)
				if err != nil {
					util.PrintLog(u.Config.Verbose, "failed to decode:%v", err)
					break
				}
			}
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

// getAddr get the dst addr and header from the packet
func (u *UDPServer) getAddr(b []byte) (dstAddr *net.UDPAddr, header []byte, data []byte) {
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
		dlen := int(b[4])
		domain := string(b[5 : 5+dlen])
		ipAddr, err := net.ResolveIPAddr("ip", domain)
		if err != nil {
			log.Printf("[udp] failed to resolve dns %s:%v", domain, err)
			return nil, nil, nil
		}
		dstAddr = &net.UDPAddr{
			IP:   ipAddr.IP,
			Port: int(b[5+dlen])<<8 | int(b[6+dlen]),
		}
		header = b[0 : 7+dlen]
		data = b[7+dlen:]
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
