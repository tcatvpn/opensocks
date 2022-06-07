# Opensocks

A multiplexing socks5 proxy that supports websocket and kcp.

[![Travis](https://travis-ci.com/net-byte/opensocks.svg?branch=main)](https://github.com/net-byte/opensocks)
[![Go Report Card](https://goreportcard.com/badge/github.com/net-byte/opensocks)](https://goreportcard.com/report/github.com/net-byte/opensocks)
![image](https://img.shields.io/badge/License-MIT-orange)
![image](https://img.shields.io/badge/License-Anti--996-red)
![image](https://img.shields.io/github/downloads/net-byte/opensocks/total.svg)

# Usage
```
Usage of opensocks:
  -S	server mode
  -bypass
    	bypass private ip
  -k string
    	encryption key (default "6w9z$C&F)J@NcRfUjXn2r4u7x!A%D*G-")
  -l string
    	local address (default "127.0.0.1:1080")
  -obfs
    	obfuscation mode
  -p string
    	protocol ws/wss/kcp (default "wss")
  -s string
    	server address (default ":8081")

```
# Run
## Run client
```
./opensocks-linux-amd64 -s=YOUR_DOMIAN:8081 -l=127.0.0.1:1080 -k=123456 -p ws -obfs
```

## Run server
```
./opensocks-linux-amd64 -S -k=123456 -obfs
```

# Docker

## Run client
```
docker run -d --restart=always  --network=host \
--name opensocks-client netbyte/opensocks -s=YOUR_DOMIAN:8081 -l=127.0.0.1:1080 -k=123456 -p ws -obfs
```

## Run server
```
docker run  -d --restart=always --net=host \
--name opensocks-server netbyte/opensocks -S -k=123456 -obfs
```

## Reverse proxy server
add tls for opensocks ws server(8081) via nginx/caddy(443)

## Server settings
settings for kcp with good performance
```
ulimit -n 65535
vi /etc/sysctl.conf
net.core.rmem_max=26214400 // BDP - bandwidth delay product
net.core.rmem_default=26214400
net.core.wmem_max=26214400
net.core.wmem_default=26214400
net.core.netdev_max_backlog=2048 // proportional to -rcvwnd
sysctl -p /etc/sysctl.conf
```

# License
[The MIT License (MIT)](https://raw.githubusercontent.com/net-byte/opensocks/main/LICENSE)

## Credits

This repo relies on the following third-party projects:
- [gobwas](https://github.com/gobwas/ws)
- [kcp-go](https://github.com/xtaci/kcp-go)
- [smux](https://github.com/xtaci/smux)
- [bpool](https://github.com/oxtoacart/bpool)
