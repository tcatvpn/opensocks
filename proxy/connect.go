package proxy

import (
	"context"
	"fmt"
	"log"
	"net"
	"strconv"
	"time"

	"github.com/gobwas/ws"
	"github.com/gobwas/ws/wsutil"
	"github.com/net-byte/opensocks/common/cipher"
	"github.com/net-byte/opensocks/common/constant"
	"github.com/net-byte/opensocks/config"
)

func connectServer(network string, host string, port string, config config.Config) net.Conn {
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
		log.Printf("[client] failed to encode request %v", err)
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
