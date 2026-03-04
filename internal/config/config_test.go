package config

import (
	"testing"
	"time"
)

func TestLoad_OTPSettingsFallbackToDefaultsWhenNonPositive(t *testing.T) {
	t.Setenv("OTP_TTL", "0s")
	t.Setenv("OTP_RATE_LIMIT", "-1s")
	t.Setenv("OTP_MAX_ATTEMPTS", "0")

	cfg := Load()

	if cfg.OTPTTL != 5*time.Minute {
		t.Fatalf("expected OTPTTL to fallback to 5m, got %s", cfg.OTPTTL)
	}
	if cfg.OTPRateLimit != time.Minute {
		t.Fatalf("expected OTPRateLimit to fallback to 1m, got %s", cfg.OTPRateLimit)
	}
	if cfg.OTPMaxAttempts != 5 {
		t.Fatalf("expected OTPMaxAttempts to fallback to 5, got %d", cfg.OTPMaxAttempts)
	}
}

func TestLoad_OTPSettingsUseProvidedPositiveValues(t *testing.T) {
	t.Setenv("OTP_TTL", "10m")
	t.Setenv("OTP_RATE_LIMIT", "30s")
	t.Setenv("OTP_MAX_ATTEMPTS", "7")

	cfg := Load()

	if cfg.OTPTTL != 10*time.Minute {
		t.Fatalf("expected OTPTTL 10m, got %s", cfg.OTPTTL)
	}
	if cfg.OTPRateLimit != 30*time.Second {
		t.Fatalf("expected OTPRateLimit 30s, got %s", cfg.OTPRateLimit)
	}
	if cfg.OTPMaxAttempts != 7 {
		t.Fatalf("expected OTPMaxAttempts 7, got %d", cfg.OTPMaxAttempts)
	}
}

func TestLoad_AdminCredentials(t *testing.T) {
	t.Setenv("ADMIN_AUTH_CREDENTIALS", "admin1:$2a$10$hash1; admin2 : $2a$10$hash2 ; invalid ; :missinglogin")

	cfg := Load()

	if len(cfg.AdminAuth.Credentials) != 2 {
		t.Fatalf("expected 2 credentials, got %d", len(cfg.AdminAuth.Credentials))
	}
	if cfg.AdminAuth.Credentials["admin1"] != "$2a$10$hash1" {
		t.Fatalf("unexpected hash for admin1")
	}
	if cfg.AdminAuth.Credentials["admin2"] != "$2a$10$hash2" {
		t.Fatalf("unexpected hash for admin2")
	}
}
