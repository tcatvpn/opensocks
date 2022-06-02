# Opensocks

A multiplexing socks5 proxy.

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
--name opensocks-client netbyte/opensocks -s=YOUR_DOMIAN:8081 -l=127.0.0.1:1080 -k=123456 -scheme ws -obfs
```

## Run server
```
docker run  -d --restart=always --net=host \
--name opensocks-server netbyte/opensocks -S -k=123456 -obfs
```

## Reverse proxy server
add tls for opensocks ws server(8081) via nginx/caddy(443)

# License
[The MIT License (MIT)](https://raw.githubusercontent.com/net-byte/opensocks/main/LICENSE)


