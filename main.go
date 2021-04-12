package main

import (
	"flag"

	"github.com/net-byte/opensocks/client"
	"github.com/net-byte/opensocks/config"
	"github.com/net-byte/opensocks/server"
)

func main() {
	config := config.Config{}
	flag.StringVar(&config.LocalAddr, "l", "0.0.0.0:1080", "local address")
	flag.StringVar(&config.ServerAddr, "s", "0.0.0.0:8081", "server address")
	flag.StringVar(&config.Key, "k", "6w9z$C&F)J@NcRfUjXn2r4u7x!A%D*G-", "encryption key")
	flag.BoolVar(&config.ServerMode, "S", false, "server mode")
	flag.BoolVar(&config.Wss, "wss", true, "scheme wss(true)/ws(false)")
	flag.BoolVar(&config.Bypass, "bypass", true, "bypass private ip")
	flag.Parse()

	if config.ServerMode {
		server.Start(config)
	} else {
		client.Start(config)
	}
}
