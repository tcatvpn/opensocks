package proxy

import (
	"context"
	"crypto/sha1"
	"fmt"
	"io"
	"log"
	"net"
	"strconv"
	"time"

	"github.com/gobwas/ws"
	"github.com/net-byte/opensocks/common/cipher"
	"github.com/net-byte/opensocks/common/enum"
	"github.com/net-byte/opensocks/config"
	"github.com/net-byte/opensocks/proto"
	"github.com/xtaci/kcp-go/v5"
	"golang.org/x/crypto/pbkdf2"
)

// Connect connects to the server
func connectServer(config config.Config) net.Conn {
	if config.Protocol == "kcp" {
		key := pbkdf2.Key([]byte(config.Key), []byte("opensocks@2022"), 1024, 32, sha1.New)
		block, _ := kcp.NewAESBlockCrypt(key)
		c, err := kcp.DialWithOptions(config.ServerAddr, block, 10, 3)
		if err != nil {
			log.Printf("[client] failed to dial kcp server %s %v", config.ServerAddr, err)
			return nil
		}
		c.SetWindowSize(enum.SndWnd, enum.RcvWnd)
		if err := c.SetReadBuffer(enum.SockBuf); err != nil {
			log.Println("[client] failed to set read buffer:", err)
		}
		log.Printf("[client] kcp server connected %s", config.ServerAddr)
		return c

	} else {
		url := fmt.Sprintf("%s://%s%s", config.Protocol, config.ServerAddr, enum.WSPath)
		dialer := &ws.Dialer{ReadBufferSize: enum.BufferSize, WriteBufferSize: enum.BufferSize, Timeout: time.Duration(enum.Timeout) * time.Second}
		c, _, _, err := dialer.Dial(context.Background(), url)
		if err != nil {
			log.Printf("[client] failed to dial websocket %s %v", url, err)
			return nil
		}
		log.Printf("[client] ws server connected %s", url)
		return c
	}
}

// Handshake handshake with the server
func handshake(stream io.ReadWriteCloser, network string, host string, port string, key string, obfs bool) bool {
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
	encode, err := proto.Encode(data)
	if err != nil {
		log.Println(err)
		return false
	}
	_, err = stream.Write(encode)
	if err != nil {
		log.Println(err)
		return false
	}
	return true
}
