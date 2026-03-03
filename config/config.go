package config

import "github.com/caarlos0/env/v11"

// Config holds all application configuration, loaded from environment variables.
type Config struct {
	AppEnv    string `env:"APP_ENV" envDefault:"development"`
	Port      string `env:"PORT" envDefault:"5000"`
	SecretKey string `env:"SECRET_KEY,required"`
	DevDBURL  string `env:"DEV_DATABASE_URL" envDefault:"data-dev.sqlite"`
	ProdDBURL string `env:"DATABASE_URL" envDefault:"postgres://user:password@localhost:5432/appdb"`
}

// Load parses environment variables into a Config struct.
func Load() (Config, error) {
	var cfg Config
	if err := env.Parse(&cfg); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

// DatabaseDSN returns the appropriate database connection string based on AppEnv.
func (c Config) DatabaseDSN() string {
	if c.AppEnv == "production" {
		return c.ProdDBURL
	}
	return c.DevDBURL
}
