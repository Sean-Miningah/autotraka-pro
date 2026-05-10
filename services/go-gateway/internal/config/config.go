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
	JWTSecret   string `mapstructure:"JWT_SECRET"`
	ServiceToken string `mapstructure:"SERVICE_TOKEN"`

	// WhatsApp channel credentials (temporary — per-channel config in DB will replace these).
	WhatsAppVerifyToken   string `mapstructure:"META_WHATSAPP_VERIFY_TOKEN"`
	WhatsAppAppSecret     string `mapstructure:"META_WHATSAPP_APP_SECRET"`
	WhatsAppAccessToken   string `mapstructure:"META_WHATSAPP_ACCESS_TOKEN"`
	WhatsAppPhoneNumberID string `mapstructure:"META_WHATSAPP_PHONE_NUMBER_ID"`
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
	v.SetDefault("JWT_SECRET", "change-me-in-production-at-least-32-chars")
	v.SetDefault("SERVICE_TOKEN", "internal-service-token-change-me")
	v.SetDefault("META_WHATSAPP_VERIFY_TOKEN", "change-me")
	v.SetDefault("META_WHATSAPP_APP_SECRET", "")
	v.SetDefault("META_WHATSAPP_ACCESS_TOKEN", "")
	v.SetDefault("META_WHATSAPP_PHONE_NUMBER_ID", "")

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
