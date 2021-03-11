package client

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"net"
	"net/url"
	"strconv"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/net-byte/opensocks/config"
)

const (
	socks5Version = uint8(5)
)

const (
	connectCommand   = uint8(1)
	bindCommand      = uint8(2)
	associateCommand = uint8(3)
	ipv4Address      = uint8(1)
	fqdnAddress      = uint8(3)
	ipv6Address      = uint8(4)
)

const (
	successReply uint8 = iota
	serverFailure
	ruleFailure
	networkUnreachable
	hostUnreachable
	connectionRefused
	ttlExpired
	commandNotSupported
	addrTypeNotSupported
)

const (
	noAuth          = uint8(0)
	noAcceptable    = uint8(255)
	userPassAuth    = uint8(2)
	userAuthVersion = uint8(1)
	authSuccess     = uint8(0)
	authFailure     = uint8(1)
)

const (
	bufferSize int = 4096
)

//Start starts server
func Start(config config.Config) {
	log.Printf("opensocks client started on %s", config.LocalAddr)
	l, err := net.Listen("tcp", config.LocalAddr)
	if err != nil {
		log.Panic(err)
	}
	for {
		conn, err := l.Accept()
		if err != nil {
			continue
		}
		go connHandler(conn, config)
	}
}

func connHandler(conn net.Conn, config config.Config) {
	buf := make([]byte, bufferSize)
	//read the version
	n, err := conn.Read(buf[0:])
	if err != nil || err == io.EOF {
		return
	}
	b := buf[0:n]
	if !checkVersionAndAuth(conn, b) {
		return
	}
	//read the cmd
	n, err = conn.Read(buf[0:])
	if err != nil || err == io.EOF {
		return
	}
	b = buf[0:n]
	if !checkCmd(conn, b) {
		return
	}

	switch b[1] {
	case connectCommand:
		//read the addr
		host, port, addrType := getAddr(conn, b)
		tcpProxy(conn, addrType, host, port, config)
		return
	case associateCommand:
		udpProxy(conn, config)
		return
	case bindCommand:
		return
	default:
		return
	}

}

func tcpProxy(conn net.Conn, addrType uint8, host string, port string, config config.Config) {
	if host == "" || port == "" {
		return
	}

	//log.Printf("remote proxy %s", host)
	wsConn := connectWS("tcp", host, port, config)
	if wsConn == nil {
		conn.Write([]byte{socks5Version, connectionRefused, 0x00, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00})
		return
	}

	response(conn, successReply)
	go forwardRemote(wsConn, conn)
	go forwardClient(wsConn, conn)

}

func udpProxy(conn net.Conn, config config.Config) {
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
	log.Printf("bind udp addr %v %v", bindUDPAddr.IP.To4().String(), bindUDPAddr.Port)

	//reply to client
	response := []byte{socks5Version, successReply, 0x00, 0x01}
	buffer := bytes.NewBuffer(response)
	binary.Write(buffer, binary.BigEndian, bindUDPAddr.IP.To4())
	binary.Write(buffer, binary.BigEndian, uint16(bindUDPAddr.Port))
	conn.Write(buffer.Bytes())

	done := make(chan bool, 0)
	go keepUDPAlive(conn.(*net.TCPConn), done)
	go replyUDP(udpConn, done, config)
	<-done
}

func response(conn net.Conn, rep byte) {
	/**
	  +----+-----+-------+------+----------+----------+
	  |VER | REP |  RSV  | ATYP | BND.ADDR | BND.PORT |
	  +----+-----+-------+------+----------+----------+
	  | 1  |  1  | X'00' |  1   | Variable |    2     |
	  +----+-----+-------+------+----------+----------+
	*/
	conn.Write([]byte{socks5Version, rep, 0x00, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00})
}

func checkVersionAndAuth(conn net.Conn, b []byte) bool {
	// Only socks5
	if b[0] != socks5Version {
		return false
	}
	// No auth
	/**
	  +----+--------+
	  |VER | METHOD |
	  +----+--------+
	  | 1  |   1    |
	  +----+--------+
	*/
	conn.Write([]byte{socks5Version, noAuth})
	return true
}

func checkCmd(conn net.Conn, b []byte) bool {
	switch b[1] {
	case connectCommand:
		return true
	case associateCommand:
		return true
	case bindCommand:
		response(conn, commandNotSupported)
		return false
	default:
		return false
	}
}

func getAddr(conn net.Conn, b []byte) (host string, port string, addrType uint8) {
	/**
	  +----+-----+-------+------+----------+----------+
	  |VER | CMD |  RSV  | ATYP | DST.ADDR | DST.PORT |
	  +----+-----+-------+------+----------+----------+
	  | 1  |  1  | X'00' |  1   | Variable |    2     |
	  +----+-----+-------+------+----------+----------+
	*/
	len := len(b)
	switch b[3] {
	case ipv4Address:
		host = net.IPv4(b[4], b[5], b[6], b[7]).String()
		break
	case fqdnAddress:
		host = string(b[5 : len-2])
		break
	case ipv6Address:
		host = net.IP(b[4:19]).String()
		break
	default:
		return "", "0", b[3]
	}
	port = strconv.Itoa(int(b[len-2])<<8 | int(b[len-1]))
	//log.Printf("host %v port %v addType %v", host, port, b[3])
	return host, port, b[3]
}

func connectWS(network string, host string, port string, config config.Config) *websocket.Conn {
	scheme := "ws"
	if config.Wss {
		scheme = "wss"
	}
	u := url.URL{Scheme: scheme, Host: config.ServerAddr, Path: "/opensocks"}
	c, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		log.Println(err)
		return nil
	}
	// Send host addr to proxy server side
	var data bytes.Buffer
	data.WriteString(host)
	data.WriteString("||")
	data.WriteString(port)
	data.WriteString("||")
	data.WriteString(config.Username)
	data.WriteString("||")
	data.WriteString(config.Password)
	data.WriteString("||")
	data.WriteString(network)
	data.WriteString("||")
	data.WriteString(strconv.FormatInt(time.Now().UnixNano(), 10))
	c.WriteMessage(websocket.BinaryMessage, data.Bytes())
	return c
}

func forwardRemote(wsConn *websocket.Conn, conn net.Conn) {
	defer wsConn.Close()
	defer conn.Close()
	for {
		buffer := make([]byte, bufferSize)
		n, err := conn.Read(buffer)
		if err != nil || err == io.EOF {
			break
		}
		wsConn.WriteMessage(websocket.BinaryMessage, buffer[:n])
	}
}

func forwardClient(wsConn *websocket.Conn, conn net.Conn) {
	defer wsConn.Close()
	defer conn.Close()
	for {
		_, buffer, err := wsConn.ReadMessage()
		if err != nil || err == io.EOF {
			break
		}
		conn.Write(buffer[:])
	}
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
	buf := make([]byte, bufferSize)
	var wsConn *websocket.Conn
	var dstAddr *net.UDPAddr
	var data []byte
	for {
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
		case ipv4Address:
			dstAddr = &net.UDPAddr{
				IP:   net.IPv4(b[4], b[5], b[6], b[7]),
				Port: int(b[8])*256 + int(b[9]),
			}
			udpServer.dstAddrMap.LoadOrStore(dstAddr.String(), string(b[0:10]))
			data = b[10:]
			break
		case fqdnAddress:
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
		case ipv6Address:
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
		wsConn = connectWS("udp", dstAddr.IP.String(), strconv.Itoa(dstAddr.Port), config)
		if wsConn == nil {
			udpServer.exitSignal <- true
			return
		}
		wsConn.WriteMessage(websocket.BinaryMessage, data)
		go func() {
			bufCopy := make([]byte, bufferSize)
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
