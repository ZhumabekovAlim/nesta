package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

type Claims struct {
	Subject string `json:"sub"`
	Role    string `json:"role"`
	Issued  int64  `json:"iat"`
	Expires int64  `json:"exp"`
	ID      string `json:"jti"`
}

func NewToken(secret string, claims Claims) (string, error) {
	header := map[string]string{"alg": "HS256", "typ": "JWT"}
	headerJSON, err := json.Marshal(header)
	if err != nil {
		return "", err
	}
	claimsJSON, err := json.Marshal(claims)
	if err != nil {
		return "", err
	}

	encodedHeader := base64.RawURLEncoding.EncodeToString(headerJSON)
	encodedClaims := base64.RawURLEncoding.EncodeToString(claimsJSON)
	signingInput := fmt.Sprintf("%s.%s", encodedHeader, encodedClaims)

	signature := sign(secret, signingInput)

	return fmt.Sprintf("%s.%s", signingInput, signature), nil
}

func ParseToken(secret, token string) (Claims, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return Claims{}, errors.New("invalid token")
	}

	signingInput := fmt.Sprintf("%s.%s", parts[0], parts[1])
	expected := sign(secret, signingInput)
	if !hmac.Equal([]byte(parts[2]), []byte(expected)) {
		return Claims{}, errors.New("invalid signature")
	}

	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return Claims{}, errors.New("invalid payload")
	}

	var claims Claims
	if err := json.Unmarshal(payload, &claims); err != nil {
		return Claims{}, errors.New("invalid claims")
	}

	if claims.Expires > 0 && time.Now().Unix() > claims.Expires {
		return Claims{}, errors.New("token expired")
	}

	return claims, nil
}

func sign(secret, input string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write([]byte(input))
	signature := mac.Sum(nil)
	return base64.RawURLEncoding.EncodeToString(signature)
}
