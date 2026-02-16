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
