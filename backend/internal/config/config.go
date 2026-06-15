package config

import (
	"os"
	"time"
)

type Config struct {
	Port, DatabasePath, JWTSecret, AllowedOrigin string
	AccessTTL, RefreshTTL                        time.Duration
}

func Load() Config {
	return Config{
		Port:          env("PORT", "8080"),
		DatabasePath:  env("DATABASE_PATH", "precious-metals.db"),
		JWTSecret:     env("JWT_SECRET", "development-secret-change-me-now"),
		AllowedOrigin: env("ALLOWED_ORIGIN", "http://localhost:5173"),
		AccessTTL:     duration("ACCESS_TOKEN_TTL", 15*time.Minute),
		RefreshTTL:    duration("REFRESH_TOKEN_TTL", 7*24*time.Hour),
	}
}

func env(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func duration(key string, fallback time.Duration) time.Duration {
	value, err := time.ParseDuration(os.Getenv(key))
	if err != nil {
		return fallback
	}
	return value
}
