package config

import "github.com/net-byte/opensocks/common/cipher"

type Config struct {
	LocalAddr  string
	ServerAddr string
	Key        string
	Scheme     string
	ServerMode bool
	Bypass     bool
	Obfuscate  bool
}

func (config *Config) Init() {
	cipher.GenerateKey(config.Key)
}
