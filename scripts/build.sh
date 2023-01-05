#!bin/bash
export GO111MODULE=on

#Linux amd64
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o ./bin/opensocks-linux-amd64 ./main.go
#Linux arm64
CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -o ./bin/opensocks-linux-arm64 ./main.go
#MacOS amd64
CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -o ./bin/opensocks-darwin-amd64 ./main.go
#MacOS arm64
CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build -o ./bin/opensocks-darwin-arm64 ./main.go
#Windows amd64
CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -o ./bin/opensocks-windows-amd64.exe ./main.go
#Windows arm64
CGO_ENABLED=0 GOOS=windows GOARCH=arm64 go build -o ./bin/opensocks-windows-arm64.exe ./main.go

echo "DONE!!!"
