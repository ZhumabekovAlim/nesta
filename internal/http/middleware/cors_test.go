package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCORSAllowedOrigin(t *testing.T) {
	h := CORS([]string{"https://v0-admin-panel-design-one-snowy.vercel.app"})(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/health", nil)
	req.Header.Set("Origin", "https://v0-admin-panel-design-one-snowy.vercel.app")
	rr := httptest.NewRecorder()

	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rr.Code)
	}
	if got := rr.Header().Get("Access-Control-Allow-Origin"); got != "https://v0-admin-panel-design-one-snowy.vercel.app" {
		t.Fatalf("expected allow origin header to be set, got %q", got)
	}
}

func TestCORSPreflightAllowedOrigin(t *testing.T) {
	h := CORS([]string{"https://v0-admin-panel-design-one-snowy.vercel.app"})(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodOptions, "/api/v1/health", nil)
	req.Header.Set("Origin", "https://v0-admin-panel-design-one-snowy.vercel.app")
	rr := httptest.NewRecorder()

	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusNoContent {
		t.Fatalf("expected status %d, got %d", http.StatusNoContent, rr.Code)
	}
	if got := rr.Header().Get("Access-Control-Allow-Origin"); got != "https://v0-admin-panel-design-one-snowy.vercel.app" {
		t.Fatalf("expected allow origin header to be set, got %q", got)
	}
}
