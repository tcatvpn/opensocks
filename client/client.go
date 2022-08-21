package client

import (
	"log"
	"net/http"
	"net/url"

	"github.com/net-byte/opensocks/config"
	"github.com/net-byte/opensocks/proxy"
	p "golang.org/x/net/proxy"
)

//Start client
func Start(config config.Config) {
	// start http server
	if config.HttpProxy {
		go startHttpServer(config)
	}
	// start udp server
	udpServer := &proxy.UDPServer{Config: config}
	udpConn := udpServer.Start()
	// start tcp server
	tcpServer := &proxy.TCPServer{Config: config, Tproxy: &proxy.TCPProxy{Config: config}, Uproxy: &proxy.UDPProxy{Config: config}, UDPConn: udpConn}
	tcpServer.Start()
}

func startHttpServer(config config.Config) {
	socksURL, err := url.Parse("socks5://" + config.LocalAddr)
	if err != nil {
		log.Fatalln("proxy url parse error:", err)
	}
	socks5Dialer, err := p.FromURL(socksURL, p.Direct)
	if err != nil {
		log.Fatalln("can not make proxy dialer:", err)
	}
	log.Printf("opensocks [http] client started on %s", config.LocalHttpProxyAddr)
	if err := http.ListenAndServe(config.LocalHttpProxyAddr, &proxy.HttpProxyHandler{Dialer: socks5Dialer}); err != nil {
		log.Fatalln("can not start http server:", err)
	}
}
