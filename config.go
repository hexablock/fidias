package fidias

import (
	"io/ioutil"
	"time"

	yaml "gopkg.in/yaml.v1"

	"github.com/hexablock/go-chord"
	"github.com/hexablock/hexalog"
	"github.com/hexablock/hexaring"
	"github.com/hexablock/hexatype"
)

// Config hold the guac config along with the underlying log and ring config
type Config struct {
	// This is internally set on bootstrap
	Version string
	// Chord ring config
	Ring *chord.Config
	// Hexalog config
	Hexalog *hexalog.Config
	// Relocate request buffer size
	RelocateBufSize int
	// Interval to wait before retrying a proposal
	RetryInterval time.Duration

	// Threshold after ring event to consider we are stable
	//StableThreshold time.Duration

	// Web UI directory
	UIDir string
	// Hexalog key namespaces
	Namespaces *NSConfig
}

// NSConfig holds a namespace config
type NSConfig struct {
	// Hexalog namespace for key value pairs
	KeyValue string
	// Hexalog namespace for filesystem paths
	FileSystem string
}

// Hostname returns the configured hostname. The assumption here is the log and ring
// hostnames are the same as they should be checked and set prior to using this call
func (conf *Config) Hostname() string {
	return conf.Ring.Hostname
}

// Hasher returns the log hasher.  This is a helper function
func (conf *Config) Hasher() hexatype.Hasher {
	return conf.Hexalog.Hasher
}

// LoadConfig loads a yaml config from a file
func LoadConfig(filename string) (*Config, error) {
	b, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	cfg := &Config{}
	if err = yaml.Unmarshal(b, cfg); err != nil {
		return nil, err
	}

	return cfg, err
}

// DefaultConfig returns a default sane config setting the hostname on the log and ring
// configs
func DefaultConfig(hostname string) *Config {
	cfg := &Config{
		RelocateBufSize: 64,
		Ring:            hexaring.DefaultConfig(hostname),
		Hexalog:         hexalog.DefaultConfig(hostname),
		//StableThreshold: 5 * time.Minute,
		RetryInterval: 10 * time.Millisecond,
		Namespaces: &NSConfig{
			KeyValue:   "keyvs/",
			FileSystem: "fs/",
		},
	}

	return cfg
}
