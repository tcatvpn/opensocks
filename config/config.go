package config

import "github.com/net-byte/opensocks/common/cipher"

type Config struct {
	LocalAddr  string
	ServerAddr string
	Username   string
	Password   string
	ServerMode bool
	Wss        bool
	Bypass     bool
	Key        []byte
}

func (config *Config) Init() {
	config.Key = cipher.CreateHash(config.Username + config.Password)
}
