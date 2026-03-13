package config

import (
	"shared/config"
)

// AppConfig extends BaseConfig with Order-Service specific settings
type AppConfig struct {
	config.BaseConfig
	// Add specific configs here if needed, like Kafka/RabbitMQ URL
}

// Load loads the environment variables into AppConfig
func Load() (*AppConfig, error) {
	return config.Load[AppConfig]()
}
