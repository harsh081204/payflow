package config

import (
	"fmt"

	"github.com/caarlos0/env/v11"
)

// BaseConfig contains common environment variables used across multiple microservices.
type BaseConfig struct {
	Port        int    `env:"PORT" envDefault:"8080"`
	Environment string `env:"ENVIRONMENT" envDefault:"development"` // development, staging, production
	DatabaseURL string `env:"DATABASE_URL,required"`
	RedisAddr   string `env:"REDIS_ADDR,required"`
	RedisPass   string `env:"REDIS_PASS"`
}

// Load loads environment variables into a struct.
// You can pass a service-specific struct that embeds BaseConfig.
func Load[T any]() (*T, error) {
	var cfg T
	if err := env.Parse(&cfg); err != nil {
		return nil, fmt.Errorf("failed to parse environment variables: %w", err)
	}
	return &cfg, nil
}
