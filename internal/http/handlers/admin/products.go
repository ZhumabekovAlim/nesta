package admin

import (
	"database/sql"
	"net/http"
	"strings"

	"nesta/internal/http/handlers"
	"nesta/internal/http/middleware"
	"nesta/internal/http/response"
	"nesta/internal/repositories"
	"nesta/internal/services"
)

type ProductHandler struct {
	Products *repositories.ProductRepository
}

type productRequest struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	PriceCents  int    `json:"price_cents"`
	Stock       int    `json:"stock"`
	CategoryID  string `json:"category_id"`
	IsActive    bool   `json:"is_active"`
}

func (h ProductHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req productRequest
	if err := handlers.DecodeJSON(r, &req); err != nil {
		response.ErrorJSON(w, http.StatusBadRequest, response.Error{Code: "VALIDATION_ERROR", Message: "invalid payload", RequestID: middleware.GetRequestID(r.Context())})
		return
	}

	id, err := services.NewID()
	if err != nil {
		response.ErrorJSON(w, http.StatusInternalServerError, response.Error{Code: "INTERNAL_ERROR", Message: "failed to create", RequestID: middleware.GetRequestID(r.Context())})
		return
	}

	product := repositories.Product{
		ID:         id,
		Title:      req.Title,
		PriceCents: req.PriceCents,
		Stock:      req.Stock,
		IsActive:   req.IsActive,
	}
	if req.Description != "" {
		product.Description = sql.NullString{String: req.Description, Valid: true}
	}
	if req.CategoryID != "" {
		product.CategoryID = sql.NullString{String: req.CategoryID, Valid: true}
	}

	if err := h.Products.Create(r.Context(), product); err != nil {
		response.ErrorJSON(w, http.StatusBadRequest, response.Error{Code: "VALIDATION_ERROR", Message: err.Error(), RequestID: middleware.GetRequestID(r.Context())})
		return
	}

	response.JSON(w, http.StatusCreated, product)
}

func (h ProductHandler) List(w http.ResponseWriter, r *http.Request) {
	items, err := h.Products.ListAll(r.Context(), 100, 0)
	if err != nil {
		response.ErrorJSON(w, http.StatusInternalServerError, response.Error{Code: "INTERNAL_ERROR", Message: "failed to list", RequestID: middleware.GetRequestID(r.Context())})
		return
	}
	response.JSON(w, http.StatusOK, map[string]any{"items": items})
}

func (h ProductHandler) Update(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/api/v1/admin/products/")
	if id == "" {
		response.ErrorJSON(w, http.StatusNotFound, response.Error{Code: "NOT_FOUND", Message: "product not found", RequestID: middleware.GetRequestID(r.Context())})
		return
	}

	var req productRequest
	if err := handlers.DecodeJSON(r, &req); err != nil {
		response.ErrorJSON(w, http.StatusBadRequest, response.Error{Code: "VALIDATION_ERROR", Message: "invalid payload", RequestID: middleware.GetRequestID(r.Context())})
		return
	}

	product := repositories.Product{
		ID:         id,
		Title:      req.Title,
		PriceCents: req.PriceCents,
		Stock:      req.Stock,
		IsActive:   req.IsActive,
	}
	if req.Description != "" {
		product.Description = sql.NullString{String: req.Description, Valid: true}
	}
	if req.CategoryID != "" {
		product.CategoryID = sql.NullString{String: req.CategoryID, Valid: true}
	}

	if err := h.Products.Update(r.Context(), product); err != nil {
		response.ErrorJSON(w, http.StatusBadRequest, response.Error{Code: "VALIDATION_ERROR", Message: err.Error(), RequestID: middleware.GetRequestID(r.Context())})
		return
	}

	response.JSON(w, http.StatusOK, product)
}

func (h ProductHandler) HandleCollection(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.List(w, r)
	case http.MethodPost:
		h.Create(w, r)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}
