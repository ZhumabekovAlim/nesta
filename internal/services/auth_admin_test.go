package services

import (
	"context"
	"errors"
	"testing"
	"time"

	"golang.org/x/crypto/bcrypt"
)

func TestAdminLogin_Success(t *testing.T) {
	hash, err := bcrypt.GenerateFromPassword([]byte("secret123"), bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("failed to generate hash: %v", err)
	}

	svc := &AuthService{
		JWTSecret:        "test-secret",
		AccessTTL:        time.Minute,
		AdminCredentials: map[string]string{"admin": string(hash)},
	}

	pair, err := svc.AdminLogin(context.Background(), "admin", "secret123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if pair.AccessToken == "" {
		t.Fatalf("expected access token")
	}
	if pair.RefreshToken != "" {
		t.Fatalf("expected empty refresh token")
	}
}

func TestAdminLogin_InvalidCredentials(t *testing.T) {
	hash, err := bcrypt.GenerateFromPassword([]byte("secret123"), bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("failed to generate hash: %v", err)
	}

	svc := &AuthService{
		JWTSecret:        "test-secret",
		AccessTTL:        time.Minute,
		AdminCredentials: map[string]string{"admin": string(hash)},
	}

	_, err = svc.AdminLogin(context.Background(), "admin", "wrong")
	if !errors.Is(err, ErrInvalidAdminCredentials) {
		t.Fatalf("expected ErrInvalidAdminCredentials, got %v", err)
	}
}
