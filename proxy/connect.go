package proxy

import (
	"context"
	"fmt"
	"log"
	"net"
	"strconv"
	"time"

	"github.com/gobwas/ws"
	"github.com/net-byte/opensocks/common/cipher"
	"github.com/net-byte/opensocks/common/enum"
	"github.com/net-byte/opensocks/config"
)

func connectServer(config config.Config) net.Conn {
	url := fmt.Sprintf("%s://%s%s", config.Scheme, config.ServerAddr, enum.WSPath)
	dialer := &ws.Dialer{ReadBufferSize: enum.BufferSize, WriteBufferSize: enum.BufferSize, Timeout: time.Duration(enum.Timeout) * time.Second}
	c, _, _, err := dialer.Dial(context.Background(), url)
	if err != nil {
		log.Printf("[client] failed to dial websocket %s %v", url, err)
		return nil
	}
	log.Printf("[client] server connected %s", url)
	return c
}

func handshake(stream net.Conn, network string, host string, port string, key string, obfs bool) bool {
	req := &RequestAddr{}
	req.Network = network
	req.Host = host
	req.Port = port
	req.Key = key
	req.Timestamp = strconv.FormatInt(time.Now().Unix(), 10)
	req.Random = cipher.Random()
	data, err := req.MarshalBinary()
	if err != nil {
		log.Printf("[client] failed to encode request %v", err)
		return false
	}
	if obfs {
		data = cipher.XOR(data)
	}
	_, err = stream.Write(data)
	if err != nil {
		log.Println(err)
		return false
	}
	return true
}
