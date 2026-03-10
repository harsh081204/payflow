package config

import (
	"shared/config"
)

// AppConfig extends BaseConfig with User-Service specific settings
type AppConfig struct {
	config.BaseConfig
	JWTSecret string `env:"JWT_SECRET" envDefault:"supersecretkey"` // Should be securely injected in production
}

// Load loads the environment variables into AppConfig
func Load() (*AppConfig, error) {
	return config.Load[AppConfig]()
}
