package config

import (
	"fmt"

	"github.com/spf13/viper"
)

// Config holds application configuration.
type Config struct {
	Port        string `mapstructure:"PORT"`
	Env         string `mapstructure:"ENV"`
	DatabaseURL string `mapstructure:"DATABASE_URL"`
	RedisURL    string `mapstructure:"REDIS_URL"`
	NATSURL     string `mapstructure:"NATS_URL"`
	MetaBaseURL string `mapstructure:"META_BASE_URL"`
}

// Load reads configuration from environment variables and, in development, a .env file.
func Load() (*Config, error) {
	v := viper.New()

	v.SetDefault("PORT", "8080")
	v.SetDefault("ENV", "development")
	v.SetDefault("DATABASE_URL", "postgres://devuser:devpass@localhost:5432/wacrm?sslmode=disable")
	v.SetDefault("REDIS_URL", "redis://localhost:6379/0")
	v.SetDefault("NATS_URL", "nats://localhost:4222")
	v.SetDefault("META_BASE_URL", "http://localhost:1080")

	v.AutomaticEnv()

	// Load .env file only in development for local convenience.
	if v.GetString("ENV") == "development" {
		v.SetConfigFile(".env")
		// Intentionally ignore error; .env is optional.
		_ = v.ReadInConfig()
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("unmarshal config: %w", err)
	}
	return &cfg, nil
}
