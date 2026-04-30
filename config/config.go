package config

import "os"

type Config struct {
	Port             string
	DatabaseURL      string
	RedisURL         string
	JWTSecret        string
	TelegramBotToken string
	TwilioAccountSID string
	TwilioAuthToken  string
	TwilioFromNumber string
}

func Load() *Config {
	return &Config{
		Port:             env("PORT", "8080"),
		DatabaseURL:      env("DATABASE_URL", "postgres://localhost/subman?sslmode=disable"),
		RedisURL:         env("REDIS_URL", "redis://localhost:6379"),
		JWTSecret:        env("JWT_SECRET", "change-me-in-production"),
		TelegramBotToken: env("TELEGRAM_BOT_TOKEN", ""),
		TwilioAccountSID: env("TWILIO_ACCOUNT_SID", ""),
		TwilioAuthToken:  env("TWILIO_AUTH_TOKEN", ""),
		TwilioFromNumber: env("TWILIO_FROM_NUMBER", "whatsapp:+14155238886"),
	}
}

func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
