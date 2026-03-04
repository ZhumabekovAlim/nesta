package users

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"nesta/internal/auth"
	"nesta/internal/http/middleware"
)

func TestMe_AdminProfileFromToken(t *testing.T) {
	secret := "test-secret"
	token, err := auth.NewToken(secret, auth.Claims{
		Subject: "admin:root",
		Role:    "admin",
		Issued:  time.Now().Unix(),
		Expires: time.Now().Add(time.Minute).Unix(),
		ID:      "token-id",
	})
	if err != nil {
		t.Fatalf("new token: %v", err)
	}

	handler := middleware.Auth(secret)(http.HandlerFunc(Handler{}.HandleProfile))
	req := httptest.NewRequest(http.MethodGet, "/api/v1/me", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d; body=%s", http.StatusOK, rr.Code, rr.Body.String())
	}

	body := rr.Body.String()
	for _, want := range []string{`"id":"admin:root"`, `"name":"root"`, `"role":"admin"`} {
		if !strings.Contains(body, want) {
			t.Fatalf("response missing %q: %s", want, body)
		}
	}
}
