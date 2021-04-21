package proxy

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"net"
	"strconv"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/net-byte/opensocks/common/cipher"
	"github.com/net-byte/opensocks/common/constant"
	"github.com/net-byte/opensocks/config"
	"github.com/net-byte/opensocks/counter"
)

type UDPServer struct {
	UDPConn *net.UDPConn
	Config  config.Config
}

func (u *UDPServer) Start() {
	udpAddr, _ := net.ResolveUDPAddr("udp", u.Config.LocalAddr)
	udpConn, err := net.ListenUDP("udp", udpAddr)
	if err != nil {
		log.Printf("[udp] failed to listen udp %v", err)
		return
	}
	u.UDPConn = udpConn
	defer u.UDPConn.Close()
	log.Printf("[udp] server started on %v", u.Config.LocalAddr)
	u.forward()
}

func (u *UDPServer) forward() {
	uf := &UDPForward{serverConn: u.UDPConn, config: u.Config}
	uf.toRemote()
}

type UDPForward struct {
	serverConn *net.UDPConn
	dstHeader  sync.Map
	wsConnPool sync.Map
	config     config.Config
}

func (uf *UDPForward) toRemote() {
	buf := make([]byte, constant.BufferSize)
	for {
		uf.serverConn.SetReadDeadline(time.Now().Add(time.Duration(constant.Timeout) * time.Second))
		n, cliAddr, err := uf.serverConn.ReadFromUDP(buf)
		if err != nil || err == io.EOF || n == 0 {
			continue
		}
		b := buf[:n]
		dstAddr, header, data := uf.getAddr(b)
		if dstAddr == nil || header == nil || data == nil {
			continue
		}
		key := getKey(cliAddr, dstAddr)
		var wsConn *websocket.Conn
		if value, ok := uf.wsConnPool.Load(key); ok {
			wsConn = value.(*websocket.Conn)
		} else {
			wsConn = ConnectWS("udp", dstAddr.IP.String(), strconv.Itoa(dstAddr.Port), uf.config)
			if wsConn == nil {
				continue
			}
			uf.wsConnPool.Store(key, wsConn)
			uf.dstHeader.Store(key, header)
			go uf.toClient(wsConn, cliAddr, dstAddr)
		}
		cipher.Encrypt(&data)
		wsConn.WriteMessage(websocket.BinaryMessage, data)
		counter.IncrWriteByte(n)
	}
}

func (uf *UDPForward) toClient(wsConn *websocket.Conn, cliAddr *net.UDPAddr, dstAddr *net.UDPAddr) {
	defer CloseWS(wsConn)
	key := getKey(cliAddr, dstAddr)
	for {
		wsConn.SetReadDeadline(time.Now().Add(time.Duration(constant.Timeout) * time.Second))
		_, buffer, err := wsConn.ReadMessage()
		n := len(buffer)
		if err != nil || err == io.EOF || n == 0 {
			break
		}
		if header, ok := uf.dstHeader.Load(key); ok {
			cipher.Decrypt(&buffer)
			var data bytes.Buffer
			data.Write(header.([]byte))
			data.Write(buffer)
			uf.serverConn.WriteToUDP(data.Bytes(), cliAddr)
			counter.IncrReadByte(n)
		}
	}
	uf.dstHeader.Delete(key)
	uf.wsConnPool.Delete(key)
}

func getKey(cliAddr *net.UDPAddr, dstAddr *net.UDPAddr) string {
	return fmt.Sprintf("%v->%v", cliAddr.String(), dstAddr.String())
}

func (uf *UDPForward) getAddr(b []byte) (dstAddr *net.UDPAddr, header []byte, data []byte) {
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
