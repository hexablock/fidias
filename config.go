package fidias

import (
	"crypto/sha256"
	"hash"

	"github.com/hashicorp/memberlist"
	kelips "github.com/hexablock/go-kelips"
	"github.com/hexablock/hexalog"
)

// Config is the fidias config
type Config struct {
	// KV prefix for the WAL
	KVPrefix string

	// Default block replicas
	Replicas int

	// Data directory
	DataDir string

	// Any existing peers. This will automatically cause the node to join the
	// cluster
	Peers []string

	Memberlist *memberlist.Config
	Hexalog    *hexalog.Config
	DHT        *kelips.Config
}

// HashFunc returns the hash function used for the fidias as a whole.  These
// will always match between the the dht, blox and hexalog
func (config *Config) HashFunc() hash.Hash {
	return config.Hexalog.Hasher()
}

// SetHashFunc sets the hash function for hexalog and the dht
func (config *Config) SetHashFunc(hf func() hash.Hash) {
	config.Hexalog.Hasher = hf
	config.DHT.HashFunc = hf
}

// DefaultConfig returns a minimally required config
func DefaultConfig() *Config {
	conf := &Config{
		Replicas: 2,
		KVPrefix: "kv/",
		Peers:    []string{},
		Hexalog:  hexalog.DefaultConfig(""),
		DHT:      kelips.DefaultConfig(""),
	}
	conf.Hexalog.Votes = 2
	conf.SetHashFunc(sha256.New)

	return conf
}
