# Opensocks

A socks5 proxy over websocket.

[![Travis](https://travis-ci.com/net-byte/opensocks.svg?branch=main)](https://github.com/net-byte/opensocks)
[![Go Report Card](https://goreportcard.com/badge/github.com/net-byte/opensocks)](https://goreportcard.com/report/github.com/net-byte/opensocks)
![image](https://img.shields.io/badge/License-MIT-orange)
![image](https://img.shields.io/badge/License-Anti--996-red)

# Usage
```
Usage of /main:
  -k string
        encryption key (default "6w9z$C&F)J@NcRfUjXn2r4u7x!A%D*G-")
  -l string
        local address (default "127.0.0.1:1080")
  -s string
        server address (default ":8081")
  -scheme string
        scheme ws/wss (default "wss")
  -bypass
        bypass private ip
  -S    server mode
  -o    data obfuscation
```

# Docker

## Run client
```
docker run -d --restart=always  --network=host \
--name opensocks-client netbyte/opensocks -s=YOUR_DOMIAN:443 -l=127.0.0.1:1080 -k=123456
```

## Run server
```
docker run  -d --restart=always --net=host \
--name opensocks-server netbyte/opensocks -S -s=:8080 -k=123456
```

## Reverse proxy
reverse proxy server(8080) via nginx/caddy(443)

# Cross-platform client
[opensocks-gui](https://github.com/net-byte/opensocks-gui)

# Deploy server
[opensocks-cloud](https://github.com/net-byte/opensocks-cloud)

# License
[The MIT License (MIT)](https://raw.githubusercontent.com/net-byte/opensocks/main/LICENSE)


