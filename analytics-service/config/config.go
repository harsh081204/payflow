package config

import (
	"shared/config"
)

// AppConfig extends BaseConfig with Analytics-Service specific settings
type AppConfig struct {
	config.BaseConfig
	KafkaBrokers string `env:"KAFKA_BROKERS" envDefault:"localhost:9092"`
	KafkaGroupID string `env:"KAFKA_GROUP_ID" envDefault:"analytics-service-group"`
}

// Load loads the environment variables into AppConfig
func Load() (*AppConfig, error) {
	return config.Load[AppConfig]()
}
