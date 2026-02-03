package config

import (
	"os"
	"strconv"
	"time"
)

type Config struct {
	Port               string
	DatabaseURL        string
	Env                string
	JWTSecret          string
	AccessTokenTTL     time.Duration
	RefreshTokenTTL    time.Duration
	OTPTTL             time.Duration
	OTPRateLimit       time.Duration
	OTPMaxAttempts     int
	SubscriptionPolicy string
}

func Load() Config {
	return Config{
		Port:               getEnv("PORT", "8080"),
		DatabaseURL:        getEnv("DB_URL", "postgres://postgres:1@localhost:5432/postgres?sslmode=disable&options=-c%20search_path%3Dnesta"),
		Env:                getEnv("APP_ENV", "development"),
		JWTSecret:          getEnv("JWT_SECRET", "dev-secret"),
		AccessTokenTTL:     getDurationEnv("ACCESS_TOKEN_TTL", 15*time.Minute),
		RefreshTokenTTL:    getDurationEnv("REFRESH_TOKEN_TTL", 720*time.Hour),
		OTPTTL:             getDurationEnv("OTP_TTL", 5*time.Minute),
		OTPRateLimit:       getDurationEnv("OTP_RATE_LIMIT", time.Minute),
		OTPMaxAttempts:     getIntEnv("OTP_MAX_ATTEMPTS", 5),
		SubscriptionPolicy: getEnv("SUBSCRIPTION_CANCEL_POLICY", "immediate"),
	}
}

func getEnv(key, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}

func getDurationEnv(key string, fallback time.Duration) time.Duration {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	parsed, err := time.ParseDuration(value)
	if err != nil {
		return fallback
	}
	return parsed
}

func getIntEnv(key string, fallback int) int {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}
	return parsed
}
