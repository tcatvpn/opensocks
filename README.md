# opensocks

Socks5(tcp/udp) over websocket,it helps you see the world.

![image](https://img.shields.io/badge/License-MIT-orange)
![image](https://img.shields.io/badge/License-Anti--996-red)

# Usage  

```
Usage of ./opensocks:
  -S	server mode
  -l string
    	local address (default "0.0.0.0:1080")
  -p string
    	password (default "pass@123456")
  -s string
    	server address (default "0.0.0.0:8081")
  -u string
    	username (default "admin")
  -wss
    	wss/ws scheme (default true)
  -bypass
    	bypass private ip (default true)


```

# Run on Docker  


## Run client
```
docker run -d --restart=always  --network=host --name opensocks-client -p 1081:1081 netbyte/opensocks -s=YOUR_DOMIAN:443
```  

## Run caddy  
```
docker run -d --restart=always --network=host -v /data/caddy/Caddyfile:/etc/Caddyfile -v /data/caddy/.caddy:/root/.caddy -e ACME_AGREE=true --name caddy abiosoft/caddy
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

## Run server
```
docker run  -d --restart=always --net=host --name opensocks-server -p 8081:8081 netbyte/opensocks -S
```


