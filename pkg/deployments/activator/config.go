package activator

import (
	"time"

	"github.com/kelseyhightower/envconfig"
)

const envconfigPrefix = "ACTIVATOR"

// Config represents the configuration options for activator
// nolint: lll
type Config struct {
	ResyncInterval time.Duration `envconfig:"INFORMERS_RESYNC_INTERVAL" required:"true"`
}

// NewConfigWithDefaults returns a Config object with default values already
// applied. Callers are then free to set custom values for the remaining fields
// and/or override default values.
func NewConfigWithDefaults() Config {
	return Config{}
}

// GetConfigFromEnvironment returns configuration derived from environment
// variables
func GetConfigFromEnvironment() (Config, error) {
	c := NewConfigWithDefaults()
	err := envconfig.Process(envconfigPrefix, &c)
	return c, err
}
