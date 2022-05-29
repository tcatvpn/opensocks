package config

import (
	"github.com/net-byte/opensocks/common/cipher"
	"github.com/oxtoacart/bpool"
)

type Config struct {
	LocalAddr  string
	ServerAddr string
	Key        string
	Scheme     string
	ServerMode bool
	Bypass     bool
	Obfs       bool
	BytePool   *bpool.BytePool
}

func (config *Config) Init() {
	cipher.GenerateKey(config.Key)
	config.BytePool = bpool.NewBytePool(128, 256*1024)
}
