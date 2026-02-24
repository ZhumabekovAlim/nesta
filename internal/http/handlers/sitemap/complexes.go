package sitemap

import (
	"crypto/subtle"
	"net/http"

	"nesta/internal/http/middleware"
	"nesta/internal/http/response"
	"nesta/internal/repositories"
)

type ComplexHandler struct {
	Complexes     *repositories.ComplexRepository
	SitemapSecret string
}

func (h ComplexHandler) Sitemap(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	if !h.isSecretValid(r.Header.Get("X-Sitemap-Secret")) {
		response.ErrorJSON(w, http.StatusForbidden, response.Error{Code: "FORBIDDEN", Message: "forbidden", RequestID: middleware.GetRequestID(r.Context())})
		return
	}

	items, err := h.Complexes.ListSitemap(r.Context())
	if err != nil {
		response.ErrorJSON(w, http.StatusInternalServerError, response.Error{Code: "INTERNAL_ERROR", Message: "failed to list", RequestID: middleware.GetRequestID(r.Context())})
		return
	}

	response.JSON(w, http.StatusOK, map[string]any{"items": items})
}

func (h ComplexHandler) isSecretValid(secret string) bool {
	if h.SitemapSecret == "" || secret == "" {
		return false
	}

	return subtle.ConstantTimeCompare([]byte(secret), []byte(h.SitemapSecret)) == 1
}
