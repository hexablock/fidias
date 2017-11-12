package fidias

import "github.com/hexablock/phi"

// Config is the fidias config
type Config struct {
	// KV prefix for the WAL
	KVPrefix string

	Phi *phi.Config

	Peers []string
}

func DefaultConfig() *Config {
	return &Config{
		KVPrefix: "kv/",
		Phi:      phi.DefaultConfig(),
	}
}
