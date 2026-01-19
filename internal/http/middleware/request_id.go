package middleware

import (
	"context"
	"net/http"

	"github.com/rs/zerolog/hlog"
)

func RequestID(next http.Handler) http.Handler {
	return hlog.RequestIDHandler("request_id", "X-Request-Id")(next)
}

func GetRequestID(ctx context.Context) string {
	id := hlog.IDFromCtx(ctx)
	if id == nil {
		return ""
	}
	return id.String()
}
