package main

import (
	"flag"

	"github.com/net-byte/opensocks/client"
	"github.com/net-byte/opensocks/config"
	"github.com/net-byte/opensocks/server"
)

func main() {
	config := config.Config{}
	flag.StringVar(&config.LocalAddr, "l", "127.0.0.1:1080", "local address")
	flag.StringVar(&config.ServerAddr, "s", ":8081", "server address")
	flag.StringVar(&config.Key, "k", "6w9z$C&F)J@NcRfUjXn2r4u7x!A%D*G-", "encryption key")
	flag.BoolVar(&config.ServerMode, "S", false, "server mode")
	flag.StringVar(&config.Protocol, "p", "wss", "protocol ws/wss/kcp")
	flag.BoolVar(&config.Bypass, "bypass", false, "bypass private ip")
	flag.BoolVar(&config.Obfs, "obfs", false, "enable data obfuscation")
	flag.BoolVar(&config.Compress, "compress", false, "enable data compression")
	flag.Parse()
	config.Init()
	if config.ServerMode {
		server.Start(config)
	} else {
		client.Start(config)
	}
}
