package main

import (
	"flag"
	"log"

	"github.com/net-byte/opensocks/client"
	"github.com/net-byte/opensocks/config"
	"github.com/net-byte/opensocks/server"
)

var _banner = `
___                                        _        
/ _ \   _ __   ___   _ _    ___  ___   __  | |__  ___
| (_) | | '_ \ / -_) | ' \  (_-< / _ \ / _| | / / (_-<
\___/  | .__/ \___| |_||_| /__/ \___/ \__| |_\_\ /__/
     |_|                                           
Source: https://github.com/net-byte/opensocks
Version: v1.6.9
`

func main() {
	config := config.Config{}
	flag.StringVar(&config.LocalAddr, "l", "127.0.0.1:1080", "local socks5 proxy address")
	flag.StringVar(&config.LocalHttpProxyAddr, "http", ":8008", "local http proxy address")
	flag.StringVar(&config.ServerAddr, "s", ":8081", "server address")
	flag.StringVar(&config.Key, "k", "6w9z$C&F)J@NcRfUjXn2r4u7x!A%D*G-", "encryption key")
	flag.BoolVar(&config.ServerMode, "S", false, "server mode")
	flag.StringVar(&config.Protocol, "p", "wss", "protocol ws/wss/kcp/tcp")
	flag.BoolVar(&config.Bypass, "bypass", false, "bypass private ip")
	flag.BoolVar(&config.Obfs, "obfs", false, "enable data obfuscation")
	flag.BoolVar(&config.Compress, "compress", false, "enable data compression")
	flag.BoolVar(&config.HttpProxy, "http-proxy", false, "enable http proxy")
	flag.BoolVar(&config.Verbose, "v", false, "enable verbose output")
	flag.Parse()
	log.Println(_banner)
	config.Init()
	if config.ServerMode {
		server.Start(config)
	} else {
		client.Start(config)
	}
}
