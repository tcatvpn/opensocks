package app

import (
	"encoding/json"
	"log"
	"strconv"

	"github.com/net-byte/opensocks/client"
	"github.com/net-byte/opensocks/config"
	"github.com/net-byte/opensocks/counter"
	"github.com/net-byte/opensocks/server"
)

// Start starts the app by json config
func Start(jsonConfig string) {
	config := config.Config{}
	err := json.Unmarshal([]byte(jsonConfig), &config)
	if err != nil {
		log.Panic("failed to decode config")
	}
	config.Init()
	if config.ServerMode {
		server.Start(config)
	} else {
		client.Start(config)
	}
}

// StopClient stops the client
func StopClient() {
	client.Stop()
}

// StopServer stops the server
func StopServer() {
	server.Stop()
}

// GetTotalReadBytes returns the total read bytes
func GetTotalReadBytes() string {
	return strconv.FormatUint(counter.TotalReadBytes, 10)
}

// GetTotalWrittenBytes returns the total written bytes
func GetTotalWrittenBytes() string {
	return strconv.FormatUint(counter.TotalWrittenBytes, 10)
}
