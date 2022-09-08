package app

import (
	"encoding/json"
	"log"

	"github.com/net-byte/opensocks/client"
	"github.com/net-byte/opensocks/config"
	"github.com/net-byte/opensocks/counter"
	"github.com/net-byte/opensocks/server"
)

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

func GetTotalReadBytes() uint64 {
	return counter.TotalReadBytes
}

func GetTotalWrittenBytes() uint64 {
	return counter.TotalWrittenBytes
}
