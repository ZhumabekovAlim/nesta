package store

import (
	"net/http"
	"strconv"
	"strings"

	"nesta/internal/http/middleware"
	"nesta/internal/http/response"
	"nesta/internal/repositories"
)

type ProductHandler struct {
	Products *repositories.ProductRepository
}

func (h ProductHandler) List(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	limit := parseInt(query.Get("limit"), 20)
	offset := parseInt(query.Get("offset"), 0)
	items, err := h.Products.List(r.Context(), query.Get("category"), query.Get("search"), query.Get("in_stock") == "1", limit, offset)
	if err != nil {
		response.ErrorJSON(w, http.StatusInternalServerError, response.Error{Code: "INTERNAL_ERROR", Message: "failed to list", RequestID: middleware.GetRequestID(r.Context())})
		return
	}

	response.JSON(w, http.StatusOK, map[string]any{"items": items, "limit": limit, "offset": offset})
}

func (h ProductHandler) Get(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/api/v1/products/")
	if id == "" {
		response.ErrorJSON(w, http.StatusNotFound, response.Error{Code: "NOT_FOUND", Message: "product not found", RequestID: middleware.GetRequestID(r.Context())})
		return
	}

	item, err := h.Products.Get(r.Context(), id)
	if err != nil {
		response.ErrorJSON(w, http.StatusNotFound, response.Error{Code: "NOT_FOUND", Message: "product not found", RequestID: middleware.GetRequestID(r.Context())})
		return
	}
	response.JSON(w, http.StatusOK, item)
}

func parseInt(value string, fallback int) int {
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}
	return parsed
}
