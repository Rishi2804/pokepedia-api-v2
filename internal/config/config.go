package config

import (
	"fmt"
	"os"
	"time"
)

type Config struct {
	Port        string
	DatabaseURL string
	Env         string

	RedisURL       string        // empty = response cache disabled
	CacheTTL       time.Duration
	CacheOpTimeout time.Duration
}

func Load() (*Config, error) {
	cfg := &Config{
		Port:        getEnv("PORT", "8080"),
		DatabaseURL: getEnv("DATABASE_URL", ""),
		Env:         getEnv("APP_ENV", "local"),

		RedisURL:       getEnv("REDIS_URL", ""),
		CacheTTL:       getEnvDuration("CACHE_TTL", 24*time.Hour),
		CacheOpTimeout: getEnvDuration("CACHE_OP_TIMEOUT", 150*time.Millisecond),
	}

	if cfg.DatabaseURL == "" {
		return nil, fmt.Errorf("DATABASE_URL is required")
	}

	return cfg, nil
}

func getEnv(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return fallback
}

func getEnvDuration(key string, fallback time.Duration) time.Duration {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			return d
		}
	}
	return fallback
}
