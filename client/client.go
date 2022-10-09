package client

import (
	"context"
	"log"
	"net/http"
	"net/url"

	"github.com/net-byte/opensocks/common/util"
	"github.com/net-byte/opensocks/config"
	"github.com/net-byte/opensocks/proxy"
	p "golang.org/x/net/proxy"
)

var _tcpServer proxy.TCPServer
var _udpServer proxy.UDPServer
var _httpServer http.Server

// Start starts the client
func Start(config config.Config) {
	util.PrintStats(config.Verbose)
	// start http server
	if config.HttpProxy {
		go startHttpServer(config)
	}
	// start udp server
	_udpServer = proxy.UDPServer{Config: config}
	udpConn := _udpServer.Start()
	// start tcp server
	_tcpServer = proxy.TCPServer{Config: config, Tproxy: &proxy.TCPProxy{Config: config}, Uproxy: &proxy.UDPProxy{Config: config}, UDPConn: udpConn}
	_tcpServer.Start()
}

// Stop stops the client
func Stop() {
	if _tcpServer.Listener != nil {
		if err := _tcpServer.Listener.Close(); err != nil {
			log.Printf("failed to shutdown tcp server: %v", err)
		}
	}
	if _udpServer.UDPConn != nil {
		if err := _udpServer.UDPConn.Close(); err != nil {
			log.Printf("failed to shutdown udp server: %v", err)
		}
	}
	if _httpServer.Addr != "" && _httpServer.Handler != nil {
		if err := _httpServer.Shutdown(context.Background()); err != nil {
			log.Printf("failed to shutdown http server: %v", err)
		}
	}
}

func startHttpServer(config config.Config) {
	socksURL, err := url.Parse("socks5://" + config.LocalAddr)
	if err != nil {
		log.Fatalln("proxy url parse error:", err)
	}
	socks5Dialer, err := p.FromURL(socksURL, p.Direct)
	if err != nil {
		log.Fatalln("failed to make proxy dialer:", err)
	}
	log.Printf("opensocks [http] client started on %s", config.LocalHttpProxyAddr)
	_httpServer = http.Server{
		Addr:    config.LocalHttpProxyAddr,
		Handler: &proxy.HttpProxyHandler{Dialer: socks5Dialer},
	}
	if err := _httpServer.ListenAndServe(); err != nil {
		log.Fatalln("failed to start http server:", err)
	}
}
