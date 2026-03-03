package config

import (
	"github.com/caarlos0/env/v11"
)

// Config holds all application configuration loaded from environment variables.
type Config struct {
	// AppEnv selects which database DSN to use: "development" (default) or "production".
	AppEnv string `env:"APP_ENV" envDefault:"development"`

	// Port is the HTTP port the server listens on.
	Port string `env:"PORT" envDefault:"5000"`

	// SecretKey is used for JWT signing. Must be set in production.
	SecretKey string `env:"SECRET_KEY" envDefault:"your-secret-key-change-me"`

	// DevDBURL is the SQLite DSN used when AppEnv != "production".
	DevDBURL string `env:"DEV_DATABASE_URL" envDefault:"data-dev.sqlite"`

	// ProdDBURL is the PostgreSQL DSN used when AppEnv == "production".
	ProdDBURL string `env:"DATABASE_URL" envDefault:"postgresql://user:password@db:5432/appdb"`
}

// Load reads environment variables into a Config struct.
func Load() (Config, error) {
	var cfg Config
	if err := env.Parse(&cfg); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

// DatabaseDSN returns the database connection string based on the current environment.
func (c Config) DatabaseDSN() string {
	if c.AppEnv == "production" {
		return c.ProdDBURL
	}
	return c.DevDBURL
}
