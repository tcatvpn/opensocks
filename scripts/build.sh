#!bin/bash
export GO111MODULE=on

#Linux
GOOS=linux GOARCH=amd64 go build -o ./bin/opensocks-linux-amd64 ./main.go
#Linux arm
GOOS=linux GOARCH=arm64 go build -o ./bin/opensocks-linux-arm64 ./main.go
#Mac OS
GOOS=darwin GOARCH=amd64 go build -o ./bin/opensocks-darwin-amd64 ./main.go
#Windows
GOOS=windows GOARCH=amd64 go build -o ./bin/opensocks-windows-amd64.exe ./main.go
#Operwrt
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -ldflags="-s -w" -o ./bin/opensocks-openwrt-amd64 ./main.go

echo "done!"
