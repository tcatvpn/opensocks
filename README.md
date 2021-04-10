# OpenSocks

OpenSocks is a net tool that helps you break the wall and see the world.

[![Travis](https://travis-ci.com/net-byte/opensocks.svg?branch=main)](https://github.com/net-byte/opensocks)
[![Go Report Card](https://goreportcard.com/badge/github.com/net-byte/opensocks)](https://goreportcard.com/report/github.com/net-byte/opensocks)
![image](https://img.shields.io/badge/License-MIT-orange)
![image](https://img.shields.io/badge/License-Anti--996-red)

# Features

* Support SOCKS5 protocol implements CONNECT/ASSOCIATE command
* Support websocket(wss/ws) for application layer
* Support data encryption with ChaCha20-Poly1305

# Docker

## Run client
```
docker run -d --restart=always  --network=host \
--name opensocks-client netbyte/opensocks -s=YOUR_DOMIAN:443
```

## Run server
```
docker run  -d --restart=always --net=host \
--name opensocks-server netbyte/opensocks -S
```

## Run caddy for reverse proxy
```
docker run -d --restart=always --network=host \
-v /data/caddy/Caddyfile:/etc/Caddyfile \
-v /data/caddy/.caddy:/root/.caddy \
-e ACME_AGREE=true --name caddy abiosoft/caddy
```

## Config Caddyfile
```
your.domain {
    gzip
    proxy / 0.0.0.0:8081 {
        websocket
        header_upstream -Origin
        header_upstream Host {host}
        header_upstream X-Real-IP {remote}
        header_upstream X-Forwarded-For {remote}
        header_upstream X-Forwarded-Port {server_port}
        header_upstream X-Forwarded-Proto {scheme}
    }
}
```

# Cross-platform client
[opensocks-gui](https://github.com/net-byte/opensocks-gui)

# Deploy server to cloud
[opensocks-cloud](https://github.com/net-byte/opensocks-cloud)

# License

[The MIT License (MIT)](https://raw.githubusercontent.com/net-byte/opensocks/main/LICENSE)


