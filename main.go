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
	flag.StringVar(&config.Scheme, "scheme", "wss", "scheme ws/wss")
	flag.BoolVar(&config.Bypass, "bypass", true, "bypass private ip")
	flag.Parse()

	if config.ServerMode {
		server.Start(config)
	} else {
		client.Start(config)
	}
}
