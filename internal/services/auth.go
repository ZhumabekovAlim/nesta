package services

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"math/rand"
	"regexp"
	"strings"
	"time"

	"nesta/internal/auth"
	"nesta/internal/repositories"
	"nesta/internal/sms/mobizon"

	"github.com/rs/zerolog/log"
)

const (
	OTPDeliveryModeDev         = "dev"
	OTPDeliveryModeMobizon     = "mobizon"
	OTPDeliveryModeMobizonEcho = "mobizon+echo"
)

var (
	ErrPhoneRequired    = errors.New("phone required")
	ErrPhoneInvalid     = errors.New("invalid phone")
	ErrRateLimited      = errors.New("rate limited")
	ErrBlocked          = errors.New("blocked")
	ErrOTPNotFound      = errors.New("otp not found")
	ErrOTPExpired       = errors.New("otp expired")
	ErrInvalidCode      = errors.New("invalid code")
	ErrSMSRateLimited   = errors.New("sms provider rate limited")
	ErrSMSDeliveryError = errors.New("sms delivery failed")
)

var nonDigit = regexp.MustCompile(`\D`)

type userStore interface {
	FindByPhone(ctx context.Context, phone string) (repositories.User, error)
	Create(ctx context.Context, user repositories.User) error
	FindByID(ctx context.Context, id string) (repositories.User, error)
}

type otpStore interface {
	LatestByPhone(ctx context.Context, phone string) (repositories.OTPCode, error)
	Create(ctx context.Context, code repositories.OTPCode) error
	IncrementAttempts(ctx context.Context, id string, attempts int) error
	Block(ctx context.Context, id string, until time.Time) error
}

type refreshTokenStore interface {
	FindByToken(ctx context.Context, token string) (repositories.RefreshToken, error)
	Create(ctx context.Context, token repositories.RefreshToken) error
	Revoke(ctx context.Context, token string, revokedAt time.Time) error
}

type smsSender interface {
	SendSMS(ctx context.Context, recipient, text, sender string, validity int) (mobizon.SendSMSResult, error)
}

type AuthService struct {
	Users            userStore
	OTP              otpStore
	RefreshTokens    refreshTokenStore
	JWTSecret        string
	AccessTTL        time.Duration
	RefreshTTL       time.Duration
	OTPTTL           time.Duration
	OTPRateLimit     time.Duration
	OTPMaxAttempts   int
	OTPDeliveryMode  string
	OTPSender        string
	OTPValidityMin   int
	OTPMessagePrefix string
	SMS              smsSender
}

type OTPResult struct {
	Code      *string
	ExpiresAt time.Time
}

type TokenPair struct {
	AccessToken  string
	RefreshToken string
	ExpiresAt    time.Time
}

func (s *AuthService) SendOTP(ctx context.Context, phone string) (OTPResult, error) {
	log.Info().Str("phone", MaskPhone(phone)).Msg("otp send started")

	normalizedPhone, err := NormalizePhone(phone)
	if err != nil {
		log.Warn().Err(err).Str("phone", MaskPhone(phone)).Msg("otp send failed: phone normalization error")
		return OTPResult{}, err
	}

	log.Info().Str("phone", MaskPhone(normalizedPhone)).Msg("otp send phone normalized")

	latest, err := s.OTP.LatestByPhone(ctx, normalizedPhone)
	if err == nil {
		log.Info().Str("phone", MaskPhone(normalizedPhone)).Str("otp_id", latest.ID).Time("latest_created_at", latest.CreatedAt).Msg("otp send latest code loaded")
		if time.Since(latest.CreatedAt) < s.OTPRateLimit {
			log.Warn().Str("phone", MaskPhone(normalizedPhone)).Str("otp_id", latest.ID).Msg("otp send rejected: rate limited")
			return OTPResult{}, ErrRateLimited
		}
		if latest.BlockedUntil.Valid && latest.BlockedUntil.Time.After(time.Now()) {
			log.Warn().Str("phone", MaskPhone(normalizedPhone)).Str("otp_id", latest.ID).Time("blocked_until", latest.BlockedUntil.Time).Msg("otp send rejected: phone blocked")
			return OTPResult{}, ErrBlocked
		}
	} else if !errors.Is(err, sql.ErrNoRows) {
		log.Error().Err(err).Str("phone", MaskPhone(normalizedPhone)).Msg("otp send failed: latest code lookup error")
		return OTPResult{}, err
	} else {
		log.Info().Str("phone", MaskPhone(normalizedPhone)).Msg("otp send latest code not found")
	}

	code := fmt.Sprintf("%06d", rand.Intn(1000000))
	codeHash := hashOTP(code)
	log.Info().Str("phone", MaskPhone(normalizedPhone)).Msg("otp send code generated")

	id, err := NewID()
	if err != nil {
		log.Error().Err(err).Str("phone", MaskPhone(normalizedPhone)).Msg("otp send failed: otp id generation error")
		return OTPResult{}, err
	}

	log.Info().Str("phone", MaskPhone(normalizedPhone)).Str("otp_id", id).Msg("otp send otp id generated")

	expiresAt := time.Now().Add(s.OTPTTL)
	log.Info().Str("phone", MaskPhone(normalizedPhone)).Str("otp_id", id).Time("expires_at", expiresAt).Msg("otp send persisting otp code")
	if err := s.OTP.Create(ctx, repositories.OTPCode{
		ID:        id,
		Phone:     normalizedPhone,
		CodeHash:  codeHash,
		ExpiresAt: expiresAt,
		Attempts:  0,
	}); err != nil {
		log.Error().Err(err).Str("phone", MaskPhone(normalizedPhone)).Str("otp_id", id).Msg("otp send failed: otp create error")
		return OTPResult{}, err
	}

	log.Info().Str("phone", MaskPhone(normalizedPhone)).Str("otp_id", id).Msg("otp send otp code persisted")

	if s.isSMSDeliveryEnabled() {
		log.Info().Str("phone", MaskPhone(normalizedPhone)).Str("otp_id", id).Str("delivery_mode", s.OTPDeliveryMode).Msg("otp send sms delivery enabled")
		if s.SMS == nil {
			log.Error().Str("phone", MaskPhone(normalizedPhone)).Str("otp_id", id).Msg("otp send failed: sms client is not configured")
			return OTPResult{}, ErrSMSDeliveryError
		}

		prefix := strings.TrimSpace(s.OTPMessagePrefix)
		text := strings.TrimSpace(fmt.Sprintf("%s %s", prefix, code))
		log.Info().
			Str("phone", MaskPhone(normalizedPhone)).
			Str("otp_id", id).
			Str("sender", s.OTPSender).
			Int("validity_min", s.OTPValidityMin).
			Str("message_prefix", prefix).
			Int("message_len", len(text)).
			Msg("otp send sending sms")

		sendResult, sendErr := s.SMS.SendSMS(ctx, normalizedPhone, text, s.OTPSender, s.OTPValidityMin)
		if sendErr != nil {
			var apiErr *mobizon.APIError
			if errors.As(sendErr, &apiErr) {
				entry := log.Error().Err(sendErr).Str("phone", MaskPhone(normalizedPhone)).Str("otp_id", id).Int("provider_error_code", apiErr.Code).Str("provider_error_message", apiErr.Message)
				if apiErr.Code == 30 {
					entry.Msg("otp send failed: sms provider rate limited")
					return OTPResult{}, ErrSMSRateLimited
				}
				entry.Msg("otp send failed: sms provider validation or delivery error")
				return OTPResult{}, ErrSMSDeliveryError
			}

			log.Error().Err(sendErr).Str("phone", MaskPhone(normalizedPhone)).Str("otp_id", id).Msg("otp send failed: sms delivery error")
			return OTPResult{}, ErrSMSDeliveryError
		}

		log.Info().Str("phone", MaskPhone(normalizedPhone)).Str("otp_id", id).Str("message_id", sendResult.MessageID).Str("campaign_id", sendResult.CampaignID).Msg("otp send sms delivered")
	} else {
		log.Info().Str("phone", MaskPhone(normalizedPhone)).Str("otp_id", id).Str("delivery_mode", s.OTPDeliveryMode).Msg("otp send sms delivery skipped")
	}

	result := OTPResult{ExpiresAt: expiresAt}
	if s.shouldEchoCode() {
		result.Code = &code
		log.Info().Str("phone", MaskPhone(normalizedPhone)).Str("otp_id", id).Msg("otp send dev code echo enabled")
	}

	log.Info().Str("phone", MaskPhone(normalizedPhone)).Str("otp_id", id).Time("expires_at", result.ExpiresAt).Msg("otp send completed")

	return result, nil
}

func (s *AuthService) VerifyOTP(ctx context.Context, phone, code string) (TokenPair, error) {
	normalizedPhone, err := NormalizePhone(phone)
	if err != nil {
		return TokenPair{}, err
	}

	latest, err := s.OTP.LatestByPhone(ctx, normalizedPhone)
	fmt.Println(latest)
	if err != nil {
		return TokenPair{}, ErrOTPNotFound
	}
	if latest.BlockedUntil.Valid && latest.BlockedUntil.Time.After(time.Now()) {
		return TokenPair{}, ErrBlocked
	}
	if latest.ExpiresAt.Before(time.Now().Add(5 * time.Hour)) {
		return TokenPair{}, errors.New("otp expired")
	}

	if hashOTP(code) != latest.CodeHash {
		attempts := latest.Attempts + 1
		_ = s.OTP.IncrementAttempts(ctx, latest.ID, attempts)
		if attempts >= s.OTPMaxAttempts {
			_ = s.OTP.Block(ctx, latest.ID, time.Now().Add(s.OTPTTL))
		}
		return TokenPair{}, ErrInvalidCode
	}

	user, err := s.Users.FindByPhone(ctx, normalizedPhone)
	if err != nil {
		if !errors.Is(err, sql.ErrNoRows) {
			return TokenPair{}, err
		}
		id, err := NewID()
		if err != nil {
			return TokenPair{}, err
		}
		user = repositories.User{ID: id, Phone: normalizedPhone, Role: "user"}
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

func NormalizePhone(phone string) (string, error) {
	trimmed := strings.TrimSpace(phone)
	if trimmed == "" {
		return "", ErrPhoneRequired
	}
	normalized := nonDigit.ReplaceAllString(trimmed, "")
	if len(normalized) < 10 || len(normalized) > 15 {
		return "", ErrPhoneInvalid
	}
	return normalized, nil
}

func MaskPhone(phone string) string {
	if len(phone) <= 4 {
		return "****"
	}
	return strings.Repeat("*", len(phone)-4) + phone[len(phone)-4:]
}

func (s *AuthService) isSMSDeliveryEnabled() bool {
	mode := strings.ToLower(strings.TrimSpace(s.OTPDeliveryMode))
	return mode == OTPDeliveryModeMobizon || mode == OTPDeliveryModeMobizonEcho
}

func (s *AuthService) shouldEchoCode() bool {
	mode := strings.ToLower(strings.TrimSpace(s.OTPDeliveryMode))
	return mode == "" || mode == OTPDeliveryModeDev || mode == OTPDeliveryModeMobizonEcho
}

func hashOTP(code string) string {
	sum := sha256.Sum256([]byte(code))
	return hex.EncodeToString(sum[:])
}
