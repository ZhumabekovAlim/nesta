package config

import (
	"os"
	"strconv"
	"strings"
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
	Robokassa          RobokassaConfig
}

type RobokassaConfig struct {
	MerchantLogin   string
	Password1       string
	Password2       string
	TestPassword1   string
	TestPassword2   string
	IsTest          bool
	HashAlgorithm   string
	PaymentURL      string
	ResultURL       string
	SuccessURL      string
	FailURL         string
	DefaultCurrency string
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
		Robokassa: RobokassaConfig{
			MerchantLogin:   getEnv("ROBOKASSA_MERCHANT_LOGIN", ""),
			Password1:       getEnv("ROBOKASSA_PASSWORD_1", ""),
			Password2:       getEnv("ROBOKASSA_PASSWORD_2", ""),
			TestPassword1:   getEnv("ROBOKASSA_TEST_PASSWORD_1", ""),
			TestPassword2:   getEnv("ROBOKASSA_TEST_PASSWORD_2", ""),
			IsTest:          getBoolEnv("ROBOKASSA_IS_TEST", false),
			HashAlgorithm:   strings.ToUpper(getEnv("ROBOKASSA_HASH_ALGO", "MD5")),
			PaymentURL:      getEnv("ROBOKASSA_PAYMENT_URL", "https://auth.robokassa.kz/Merchant/Index.aspx"),
			ResultURL:       getEnv("ROBOKASSA_RESULT_URL", "/api/v1/payments/robokassa/result"),
			SuccessURL:      getEnv("ROBOKASSA_SUCCESS_URL", "/api/v1/payments/robokassa/success"),
			FailURL:         getEnv("ROBOKASSA_FAIL_URL", "/api/v1/payments/robokassa/fail"),
			DefaultCurrency: getEnv("ROBOKASSA_DEFAULT_CURRENCY", "KZT"),
		},
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

func getBoolEnv(key string, fallback bool) bool {
	value := strings.TrimSpace(strings.ToLower(os.Getenv(key)))
	if value == "" {
		return fallback
	}
	return value == "1" || value == "true" || value == "yes"
}
