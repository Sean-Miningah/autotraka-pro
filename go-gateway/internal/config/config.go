package config

import (
	"os"
)

// Config holds application configuration.
type Config struct {
	Port        string
	Env         string
	DatabaseURL string
	RedisURL    string
	NATSURL     string
	MetaBaseURL string
}

// Load reads configuration from environment variables.
func Load() Config {
	return Config{
		Port:        getEnv("PORT", "8080"),
		Env:         getEnv("ENV", "development"),
		DatabaseURL: getEnv("DATABASE_URL", "postgres://devuser:devpass@localhost:5432/wacrm?sslmode=disable"),
		RedisURL:    getEnv("REDIS_URL", "redis://localhost:6379/0"),
		NATSURL:     getEnv("NATS_URL", "nats://localhost:4222"),
		MetaBaseURL: getEnv("META_BASE_URL", "http://localhost:1080"),
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
