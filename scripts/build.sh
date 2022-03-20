#!bin/bash
export GO111MODULE=on

#Linux
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o ./bin/opensocks-linux-amd64 ./main.go
#Linux arm
CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -o ./bin/opensocks-linux-arm64 ./main.go
#Mac OS
CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -o ./bin/opensocks-darwin-amd64 ./main.go
#Windows
CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -o ./bin/opensocks-windows-amd64.exe ./main.go
#Operwrt
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -ldflags="-s -w" -o ./bin/opensocks-openwrt-amd64 ./main.go

echo "DONE!!!"
