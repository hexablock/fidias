package fidias

import (
	"time"

	"github.com/hexablock/hexalog"
	"github.com/hexablock/hexaring"
	"github.com/hexablock/hexatype"
)

// Config hold the guac config along with the underlying log and ring config
type Config struct {
	Ring             *hexaring.Config
	Hexalog          *hexalog.Config
	RebalanceBufSize int           // Rebalance request buffer size
	Replicas         int           // Number of replicas for a key
	RetryInterval    time.Duration // interval to wait before retrying
	StableThreshold  time.Duration // Threshold after ring event to consider we are stable
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

// DefaultConfig returns a default sane config setting the hostname on the log and ring
// configs
func DefaultConfig(hostname string) *Config {
	cfg := &Config{
		Replicas:         3,
		RebalanceBufSize: 64,
		Ring:             hexaring.DefaultConfig(hostname),
		Hexalog:          hexalog.DefaultConfig(hostname),
		StableThreshold:  5 * time.Minute,
		RetryInterval:    10 * time.Millisecond,
	}

	return cfg
}
