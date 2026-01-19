package services

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"math/rand"
	"time"

	"nesta/internal/auth"
	"nesta/internal/repositories"
)

type AuthService struct {
	Users          *repositories.UserRepository
	OTP            *repositories.OTPRepository
	RefreshTokens  *repositories.RefreshTokenRepository
	JWTSecret      string
	AccessTTL      time.Duration
	RefreshTTL     time.Duration
	OTPTTL         time.Duration
	OTPRateLimit   time.Duration
	OTPMaxAttempts int
}

type OTPResult struct {
	Code      string
	ExpiresAt time.Time
}

type TokenPair struct {
	AccessToken  string
	RefreshToken string
	ExpiresAt    time.Time
}

func (s *AuthService) SendOTP(ctx context.Context, phone string) (OTPResult, error) {
	if phone == "" {
		return OTPResult{}, errors.New("phone required")
	}

	latest, err := s.OTP.LatestByPhone(ctx, phone)
	if err == nil {
		if time.Since(latest.CreatedAt) < s.OTPRateLimit {
			return OTPResult{}, errors.New("rate limited")
		}
		if latest.BlockedUntil.Valid && latest.BlockedUntil.Time.After(time.Now()) {
			return OTPResult{}, errors.New("blocked")
		}
	}

	code := fmt.Sprintf("%06d", rand.Intn(1000000))
	codeHash := hashOTP(code)
	id, err := NewID()
	if err != nil {
		return OTPResult{}, err
	}

	expiresAt := time.Now().Add(s.OTPTTL)
	if err := s.OTP.Create(ctx, repositories.OTPCode{
		ID:        id,
		Phone:     phone,
		CodeHash:  codeHash,
		ExpiresAt: expiresAt,
		Attempts:  0,
	}); err != nil {
		return OTPResult{}, err
	}

	return OTPResult{Code: code, ExpiresAt: expiresAt}, nil
}

func (s *AuthService) VerifyOTP(ctx context.Context, phone, code string) (TokenPair, error) {
	latest, err := s.OTP.LatestByPhone(ctx, phone)
	if err != nil {
		return TokenPair{}, errors.New("otp not found")
	}
	if latest.BlockedUntil.Valid && latest.BlockedUntil.Time.After(time.Now()) {
		return TokenPair{}, errors.New("blocked")
	}
	if latest.ExpiresAt.Before(time.Now()) {
		return TokenPair{}, errors.New("otp expired")
	}

	if hashOTP(code) != latest.CodeHash {
		attempts := latest.Attempts + 1
		_ = s.OTP.IncrementAttempts(ctx, latest.ID, attempts)
		if attempts >= s.OTPMaxAttempts {
			_ = s.OTP.Block(ctx, latest.ID, time.Now().Add(s.OTPTTL))
		}
		return TokenPair{}, errors.New("invalid code")
	}

	user, err := s.Users.FindByPhone(ctx, phone)
	if err != nil {
		if !errors.Is(err, sql.ErrNoRows) {
			return TokenPair{}, err
		}
		id, err := NewID()
		if err != nil {
			return TokenPair{}, err
		}
		user = repositories.User{ID: id, Phone: phone, Role: "user"}
		if err := s.Users.Create(ctx, user); err != nil {
			return TokenPair{}, err
		}
	}

	return s.issueTokens(ctx, user.ID, user.Role)
}

func (s *AuthService) Refresh(ctx context.Context, refreshToken string) (TokenPair, error) {
	stored, err := s.RefreshTokens.FindByToken(ctx, refreshToken)
	if err != nil {
		return TokenPair{}, errors.New("invalid refresh")
	}
	if stored.RevokedAt.Valid {
		return TokenPair{}, errors.New("refresh revoked")
	}
	if stored.ExpiresAt.Before(time.Now()) {
		return TokenPair{}, errors.New("refresh expired")
	}

	user, err := s.Users.FindByID(ctx, stored.UserID)
	if err != nil {
		return TokenPair{}, err
	}

	return s.issueTokens(ctx, user.ID, user.Role)
}

func (s *AuthService) Logout(ctx context.Context, refreshToken string) error {
	return s.RefreshTokens.Revoke(ctx, refreshToken, time.Now())
}

func (s *AuthService) issueTokens(ctx context.Context, userID, role string) (TokenPair, error) {
	accessID, err := NewID()
	if err != nil {
		return TokenPair{}, err
	}
	refreshID, err := NewID()
	if err != nil {
		return TokenPair{}, err
	}

	issuedAt := time.Now()
	expiresAt := issuedAt.Add(s.AccessTTL)
	accessToken, err := auth.NewToken(s.JWTSecret, auth.Claims{
		Subject: userID,
		Role:    role,
		Issued:  issuedAt.Unix(),
		Expires: expiresAt.Unix(),
		ID:      accessID,
	})
	if err != nil {
		return TokenPair{}, err
	}

	refreshTokenValue, err := NewID()
	if err != nil {
		return TokenPair{}, err
	}

	if err := s.RefreshTokens.Create(ctx, repositories.RefreshToken{
		ID:        refreshID,
		UserID:    userID,
		Token:     refreshTokenValue,
		ExpiresAt: issuedAt.Add(s.RefreshTTL),
	}); err != nil {
		return TokenPair{}, err
	}

	return TokenPair{AccessToken: accessToken, RefreshToken: refreshTokenValue, ExpiresAt: expiresAt}, nil
}

func hashOTP(code string) string {
	sum := sha256.Sum256([]byte(code))
	return hex.EncodeToString(sum[:])
}
