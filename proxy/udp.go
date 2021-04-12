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

func UDPProxy(tcpConn net.Conn, config config.Config) {
	defer tcpConn.Close()
	//start local udp server
	localTCPAddr, _ := net.ResolveTCPAddr("tcp", tcpConn.LocalAddr().String())
	localUDPAddr := fmt.Sprintf("%s:%d", localTCPAddr.IP.String(), 0)
	udpAddr, _ := net.ResolveUDPAddr("udp", localUDPAddr)
	udpConn, err := net.ListenUDP("udp", udpAddr)
	if err != nil {
		log.Printf("[udp] listen error:%v", err)
		return
	}
	defer udpConn.Close()
	bindAddr, _ := net.ResolveUDPAddr("udp", udpConn.LocalAddr().String())
	//response to client
	ResponseUDPAddr(tcpConn, bindAddr)
	//forward udp
	done := make(chan bool, 0)
	go keepUDPAlive(tcpConn.(*net.TCPConn), done)
	go forwardUDP(udpConn, config)
	<-done
}

func keepUDPAlive(tcpConn *net.TCPConn, done chan<- bool) {
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

func forwardUDP(udpConn *net.UDPConn, config config.Config) {
	udpServer := &UDPServer{serverConn: udpConn, config: config}
	udpServer.forwardRemote()
}

type UDPServer struct {
	clientAddr   *net.UDPAddr
	serverConn   *net.UDPConn
	dstAddrCache sync.Map
	wsConnCache  sync.Map
	config       config.Config
}

func (udpServer *UDPServer) forwardRemote() {
	buf := make([]byte, constant.BufferSize)
	for {
		udpServer.serverConn.SetReadDeadline(time.Now().Add(60 * time.Second))
		n, clientAddr, err := udpServer.serverConn.ReadFromUDP(buf)
		if err != nil || err == io.EOF || n == 0 {
			break
		}
		if udpServer.clientAddr == nil {
			udpServer.clientAddr = clientAddr
		}
		b := buf[:n]
		dstAddr, data := udpServer.getAddr(b)
		if dstAddr == nil || data == nil {
			continue
		}
		var wsConn *websocket.Conn
		if value, ok := udpServer.wsConnCache.Load(dstAddr.String()); ok {
			wsConn = value.(*websocket.Conn)
		} else {
			wsConn = ConnectWS("udp", dstAddr.IP.String(), strconv.Itoa(dstAddr.Port), udpServer.config)
			if wsConn == nil {
				break
			}
			udpServer.wsConnCache.Store(dstAddr.String(), wsConn)
			go udpServer.forwardClient(wsConn, dstAddr)
		}
		cipher.Encrypt(&data)
		wsConn.WriteMessage(websocket.BinaryMessage, data)
		counter.IncrWriteByte(n)
	}
}

func (udpServer *UDPServer) forwardClient(wsConn *websocket.Conn, dstAddr *net.UDPAddr) {
	defer CloseWS(wsConn)
	for {
		wsConn.SetReadDeadline(time.Now().Add(60 * time.Second))
		_, buffer, err := wsConn.ReadMessage()
		n := len(buffer)
		if err != nil || err == io.EOF || n == 0 {
			break
		}
		if header, ok := udpServer.dstAddrCache.Load(dstAddr.String()); ok {
			cipher.Decrypt(&buffer)
			var data bytes.Buffer
			data.Write([]byte(header.(string)))
			data.Write(buffer)
			udpServer.serverConn.WriteToUDP(data.Bytes(), udpServer.clientAddr)
			counter.IncrReadByte(n)
		}
	}
}

func (udpServer *UDPServer) getAddr(b []byte) (dstAddr *net.UDPAddr, data []byte) {
	/*
	   +----+------+------+----------+----------+----------+
	   |RSV | FRAG | ATYP | DST.ADDR | DST.PORT |   DATA   |
	   +----+------+------+----------+----------+----------+
	   |  2 |   1  |   1  | Variable |     2    | Variable |
	   +----+------+------+----------+----------+----------+
	*/
	if b[2] != 0x00 {
		log.Printf("[udp] frag %v do not support", b[2])
		return nil, nil
	}
	switch b[3] {
	case constant.Ipv4Address:
		dstAddr = &net.UDPAddr{
			IP:   net.IPv4(b[4], b[5], b[6], b[7]),
			Port: int(b[8])<<8 | int(b[9]),
		}
		udpServer.dstAddrCache.LoadOrStore(dstAddr.String(), string(b[0:10]))
		data = b[10:]
	case constant.FqdnAddress:
		domainLength := int(b[4])
		domain := string(b[5 : 5+domainLength])
		ipAddr, err := net.ResolveIPAddr("ip", domain)
		if err != nil {
			log.Printf("[udp] dns resolve %s error:%v", domain, err)
			return nil, nil
		}
		dstAddr = &net.UDPAddr{
			IP:   ipAddr.IP,
			Port: int(b[5+domainLength])<<8 | int(b[6+domainLength]),
		}
		udpServer.dstAddrCache.LoadOrStore(dstAddr.String(), string(b[0:7+domainLength]))
		data = b[7+domainLength:]
	case constant.Ipv6Address:
		{
			dstAddr = &net.UDPAddr{
				IP:   net.IP(b[4:19]),
				Port: int(b[20])<<8 | int(b[21]),
			}
			udpServer.dstAddrCache.LoadOrStore(dstAddr.String(), string(b[0:22]))
			data = b[22:]
		}
	default:
		return nil, nil
	}
	return dstAddr, data
}
