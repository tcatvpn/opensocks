package config

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
