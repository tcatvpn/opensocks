package config

import (
	"github.com/net-byte/opensocks/common/cipher"
)

type Config struct {
	LocalAddr  string
	ServerAddr string
	Key        string
	Protocol   string
	ServerMode bool
	Bypass     bool
	Obfs       bool
}

func (config *Config) Init() {
	cipher.GenerateKey(config.Key)
}
