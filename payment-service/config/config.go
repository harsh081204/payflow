package config

import (
	"shared/config"
)

// AppConfig extends BaseConfig with Payment-Service specific settings
type AppConfig struct {
	config.BaseConfig
	// Specific configs like Merchant IDs, API keys, etc.
}

// Load loads the environment variables into AppConfig
func Load() (*AppConfig, error) {
	return config.Load[AppConfig]()
}
