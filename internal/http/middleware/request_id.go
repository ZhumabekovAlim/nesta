package middleware

import (
	"context"
	"net/http"

	"github.com/rs/xid"
	"github.com/rs/zerolog/hlog"
)

func RequestID(next http.Handler) http.Handler {
	return hlog.RequestIDHandler("request_id", "X-Request-Id")(next)
}

func GetRequestID(ctx context.Context) string {
	id, ok := hlog.IDFromCtx(ctx)
	if !ok || id == xid.NilID() {
		return ""
	}
	return id.String()
}
