package proxy

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"net"
	"strconv"
	"sync"

	"github.com/gorilla/websocket"
	"github.com/net-byte/opensocks/config"
)

func UdpProxy(conn net.Conn, config config.Config) {
	//bind udp local server
	localTCPAddr, _ := net.ResolveTCPAddr("tcp", conn.LocalAddr().String())
	localUDPAddr := fmt.Sprintf("%s:%d", localTCPAddr.IP.String(), 0)
	udpAddr, _ := net.ResolveUDPAddr("udp", localUDPAddr)
	udpConn, err := net.ListenUDP("udp", udpAddr)
	if err != nil {
		log.Println(err)
		return
	}
	bindUDPAddr, _ := net.ResolveUDPAddr("udp", udpConn.LocalAddr().String())
	//log.Printf("bind udp addr %v %v", bindUDPAddr.IP.To4().String(), bindUDPAddr.Port)

	//reply to client
	response := []byte{Socks5Version, SuccessReply, 0x00, 0x01}
	buffer := bytes.NewBuffer(response)
	binary.Write(buffer, binary.BigEndian, bindUDPAddr.IP.To4())
	binary.Write(buffer, binary.BigEndian, uint16(bindUDPAddr.Port))
	conn.Write(buffer.Bytes())

	done := make(chan bool, 0)
	go keepUDPAlive(conn.(*net.TCPConn), done)
	go replyUDP(udpConn, done, config)
	<-done
}

type udpServer struct {
	exitSignal    chan bool
	clientUDPAddr *net.UDPAddr
	dstAddrMap    sync.Map
}

func (udpServer *udpServer) monitor() {
	for {
		select {
		case <-udpServer.exitSignal:
			return
		}
	}
}

func (udpServer *udpServer) forward(udpConn *net.UDPConn, config config.Config) {
	defer udpConn.Close()
	buf := make([]byte, BufferSize)
	for {
		var dstAddr *net.UDPAddr
		var data []byte
		n, udpAddr, err := udpConn.ReadFromUDP(buf)
		if err != nil || n == 0 {
			udpServer.exitSignal <- true
			return
		}
		if udpServer.clientUDPAddr == nil {
			udpServer.clientUDPAddr = udpAddr
		}
		b := buf[:n]
		/*
		   +----+------+------+----------+----------+----------+
		   |RSV | FRAG | ATYP | DST.ADDR | DST.PORT |   DATA   |
		   +----+------+------+----------+----------+----------+
		   |  2 |   1  |   1  | Variable |     2    | Variable |
		   +----+------+------+----------+----------+----------+
		*/
		if b[2] != 0x00 {
			log.Printf("FRAG do not support.\n")
			continue
		}

		switch b[3] {
		case Ipv4Address:
			dstAddr = &net.UDPAddr{
				IP:   net.IPv4(b[4], b[5], b[6], b[7]),
				Port: int(b[8])*256 + int(b[9]),
			}
			udpServer.dstAddrMap.LoadOrStore(dstAddr.String(), string(b[0:10]))
			data = b[10:]
			break
		case FqdnAddress:
			domainLength := int(b[4])
			domain := string(b[5 : 5+domainLength])
			ipAddr, err := net.ResolveIPAddr("ip", domain)
			if err != nil {
				log.Printf("dns resolve %s error:%v\n", domain, err)
				continue
			}
			dstAddr = &net.UDPAddr{
				IP:   ipAddr.IP,
				Port: int(b[5+domainLength])*256 + int(b[6+domainLength]),
			}
			udpServer.dstAddrMap.LoadOrStore(dstAddr.String(), string(b[0:7+domainLength]))
			data = b[7+domainLength:]
			break
		case Ipv6Address:
			{
				dstAddr = &net.UDPAddr{
					IP:   net.IP(b[4:19]),
					Port: int(b[20])*256 + int(b[21]),
				}
				udpServer.dstAddrMap.LoadOrStore(dstAddr.String(), string(b[0:22]))
				data = b[22:]
				break
			}
		default:
			continue
		}
		wsConn := ConnectWS("udp", dstAddr.IP.String(), strconv.Itoa(dstAddr.Port), config)
		if wsConn == nil {
			udpServer.exitSignal <- true
			return
		}
		wsConn.WriteMessage(websocket.BinaryMessage, data)
		go func() {
			defer wsConn.Close()
			bufCopy := make([]byte, BufferSize)
			for {
				_, buffer, err := wsConn.ReadMessage()
				if err != nil || err == io.EOF {
					udpServer.exitSignal <- true
					return
				}
				if header, ok := udpServer.dstAddrMap.Load(dstAddr.String()); ok {
					head := []byte(header.(string))
					headLength := len(head)
					copy(bufCopy[0:], head[0:headLength])
					copy(bufCopy[headLength:], buffer[0:])
					data := bufCopy[0 : headLength+len(buffer)]
					udpConn.WriteToUDP(data, udpServer.clientUDPAddr)
				}
			}
		}()
	}

}

func keepUDPAlive(tcpConn *net.TCPConn, done chan<- bool) {
	tcpConn.SetKeepAlive(true)
	buf := make([]byte, 1024)
	for {
		_, err := tcpConn.Read(buf[0:])
		if err != nil {
			done <- true
			return
		}
	}
}

func replyUDP(udpConn *net.UDPConn, done chan<- bool, config config.Config) {
	udpServer := &udpServer{
		exitSignal: make(chan bool, 0),
	}

	go udpServer.forward(udpConn, config)
	udpServer.monitor()
	done <- true
	return
}
