@ECHO OFF
ECHO start building
:: Linux amd64
go env -w GOARCH=amd64
go env -w GOOS=linux
go build -o .\bin\opensocks-linux-amd64 .\main.go
:: Linux arm64
go env -w GOARCH=arm64
go env -w GOOS=linux
go build -o .\bin\opensocks-linux-arm64 .\main.go
:: Mac amd64
go env -w GOARCH=amd64
go env -w GOOS=darwin
go build -o .\bin\opensocks-darwin-amd64 .\main.go
:: Mac arm64
go env -w GOARCH=arm64
go env -w GOOS=darwin
go build -o .\bin\opensocks-darwin-arm64 .\main.go
:: Windows arm64
go env -w GOARCH=arm64
go env -w GOOS=windows
go build -o .\bin\opensocks-windows-arm64.exe .\main.go
:: Windows amd64
go env -w GOARCH=amd64
go env -w GOOS=windows
go build -o .\bin\opensocks-windows-amd64.exe .\main.go

ECHO done