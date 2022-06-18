package config

import (
	"github.com/net-byte/opensocks/common/cipher"
)

// The config struct
type Config struct {
	LocalAddr  string
	ServerAddr string
	Key        string
	Protocol   string
	ServerMode bool
	Bypass     bool
	Obfs       bool
	Compress   bool
}

func (config *Config) Init() {
	cipher.GenerateKey(config.Key)
}
