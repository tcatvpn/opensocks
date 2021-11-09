package proxy

import (
	"context"
	"fmt"
	"io"
	"log"
	"net"
	"strconv"
	"time"

	"github.com/gobwas/ws"
	"github.com/gobwas/ws/wsutil"
	"github.com/net-byte/opensocks/common/cipher"
	"github.com/net-byte/opensocks/common/constant"
	"github.com/net-byte/opensocks/config"
	"github.com/net-byte/opensocks/counter"
)

func ConnectWS(network string, host string, port string, config config.Config) net.Conn {
	url := fmt.Sprintf("%s://%s%s", config.Scheme, config.ServerAddr, constant.WSPath)
	c, _, _, err := ws.DefaultDialer.Dial(context.Background(), url)
	if err != nil {
		log.Printf("[client] failed to dial websocket %s %v", url, err)
		return nil
	}
	// handshake
	req := &RequestAddr{}
	req.Network = network
	req.Host = host
	req.Port = port
	req.Key = config.Key
	req.Timestamp = strconv.FormatInt(time.Now().Unix(), 10)
	req.Random = cipher.Random()
	data, err := req.MarshalBinary()
	if err != nil {
		log.Printf("[client] failed to marshal binary %v", err)
		return nil
	}
	if config.Obfuscate {
		data = cipher.XOR(data)
	}
	wserr := wsutil.WriteClientText(c, data)
	if wserr != nil {
		log.Printf("[client] failed to write message %v", wserr)
		return nil
	}
	return c
}

func TCPToWS(config config.Config, wsconn net.Conn, tcpconn net.Conn) {
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
		counter.IncrWriteByte(n)
		wsutil.WriteClientBinary(wsconn, b)
	}
}

func WSToTCP(config config.Config, wsconn net.Conn, tcpconn net.Conn) {
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
		counter.IncrReadByte(n)
		tcpconn.Write(buffer[:])
	}
}
