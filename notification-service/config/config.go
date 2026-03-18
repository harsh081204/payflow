package config

import (
	"shared/config"
)

// AppConfig extends BaseConfig with Notification-Service specific settings
type AppConfig struct {
	config.BaseConfig
	KafkaBrokers string `env:"KAFKA_BROKERS" envDefault:"localhost:9092"`
	KafkaGroupID string `env:"KAFKA_GROUP_ID" envDefault:"notification-service-group"`
	WorkerCount  int    `env:"WORKER_COUNT" envDefault:"10"`
}

// Load loads the environment variables into AppConfig
func Load() (*AppConfig, error) {
	return config.Load[AppConfig]()
}
