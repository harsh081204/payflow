package config

import (
	"shared/config"
)

// AppConfig extends BaseConfig with API-Gateway specific settings
type AppConfig struct {
	config.BaseConfig
	JWTSecret         string `env:"JWT_SECRET" envDefault:"supersecretkey"`
	UserServiceURL    string `env:"USER_SERVICE_URL" envDefault:"http://localhost:8081"`
	OrderServiceURL   string `env:"ORDER_SERVICE_URL" envDefault:"http://localhost:8082"`
	PaymentServiceURL string `env:"PAYMENT_SERVICE_URL" envDefault:"http://localhost:8083"`
}

// Load loads the environment variables into AppConfig
func Load() (*AppConfig, error) {
	return config.Load[AppConfig]()
}
