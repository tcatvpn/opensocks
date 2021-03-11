package main

import (
	"flag"

	"github.com/net-byte/opensocks/client"
	"github.com/net-byte/opensocks/config"
	"github.com/net-byte/opensocks/server"
)

func main() {
	config := config.Config{}
	flag.StringVar(&config.LocalAddr, "l", "0.0.0.0:1081", "local address")
	flag.StringVar(&config.ServerAddr, "s", "0.0.0.0:8081", "server address")
	flag.StringVar(&config.Username, "u", "admin", "username")
	flag.StringVar(&config.Password, "p", "pass@123456", "password")
	flag.BoolVar(&config.ServerMode, "S", false, "server mode")
	flag.BoolVar(&config.Wss, "wss", true, "wss/ws scheme")
	flag.Parse()
	if config.ServerMode {
		server.Start(config)
	} else {
		client.Start(config)
	}
}
