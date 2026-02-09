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

type SubscriptionTypeHandler struct {
	Types *repositories.SubscriptionTypeRepository
}

type subscriptionTypeRequest struct {
	Title      string   `json:"title"`
	Subtitle   string   `json:"subtitle"`
	PriceCents int      `json:"price_cents"`
	Features   []string `json:"features"`
	IsActive   bool     `json:"is_active"`
}

func (h SubscriptionTypeHandler) List(w http.ResponseWriter, r *http.Request) {
	items, err := h.Types.ListActive(r.Context())
	if err != nil {
		response.ErrorJSON(w, http.StatusInternalServerError, response.Error{Code: "INTERNAL_ERROR", Message: "failed to list", RequestID: middleware.GetRequestID(r.Context())})
		return
	}
	response.JSON(w, http.StatusOK, map[string]any{"items": items})
}

func (h SubscriptionTypeHandler) HandleCollection(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.List(w, r)
	case http.MethodPost:
		h.Create(w, r)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (h SubscriptionTypeHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req subscriptionTypeRequest
	if err := handlers.DecodeJSON(r, &req); err != nil {
		response.ErrorJSON(w, http.StatusBadRequest, response.Error{Code: "VALIDATION_ERROR", Message: "invalid payload", RequestID: middleware.GetRequestID(r.Context())})
		return
	}

	id, err := services.NewID()
	if err != nil {
		response.ErrorJSON(w, http.StatusInternalServerError, response.Error{Code: "INTERNAL_ERROR", Message: "failed to create", RequestID: middleware.GetRequestID(r.Context())})
		return
	}

	item := repositories.SubscriptionType{
		ID:         id,
		Title:      req.Title,
		PriceCents: req.PriceCents,
		Features:   req.Features,
		IsActive:   req.IsActive,
	}
	if req.Subtitle != "" {
		item.Subtitle = sql.NullString{String: req.Subtitle, Valid: true}
	}

	if err := h.Types.Create(r.Context(), item); err != nil {
		response.ErrorJSON(w, http.StatusBadRequest, response.Error{Code: "VALIDATION_ERROR", Message: err.Error(), RequestID: middleware.GetRequestID(r.Context())})
		return
	}

	response.JSON(w, http.StatusCreated, item)
}

func (h SubscriptionTypeHandler) Update(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/api/v1/admin/subscription-types/")
	if id == "" {
		response.ErrorJSON(w, http.StatusNotFound, response.Error{Code: "NOT_FOUND", Message: "subscription type not found", RequestID: middleware.GetRequestID(r.Context())})
		return
	}

	var req subscriptionTypeRequest
	if err := handlers.DecodeJSON(r, &req); err != nil {
		response.ErrorJSON(w, http.StatusBadRequest, response.Error{Code: "VALIDATION_ERROR", Message: "invalid payload", RequestID: middleware.GetRequestID(r.Context())})
		return
	}

	item := repositories.SubscriptionType{
		ID:         id,
		Title:      req.Title,
		PriceCents: req.PriceCents,
		Features:   req.Features,
		IsActive:   req.IsActive,
	}
	if req.Subtitle != "" {
		item.Subtitle = sql.NullString{String: req.Subtitle, Valid: true}
	}

	if err := h.Types.Update(r.Context(), item); err != nil {
		response.ErrorJSON(w, http.StatusBadRequest, response.Error{Code: "VALIDATION_ERROR", Message: err.Error(), RequestID: middleware.GetRequestID(r.Context())})
		return
	}

	response.JSON(w, http.StatusOK, item)
}

func (h SubscriptionTypeHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/api/v1/admin/subscription-types/")
	if id == "" {
		response.ErrorJSON(w, http.StatusNotFound, response.Error{Code: "NOT_FOUND", Message: "subscription type not found", RequestID: middleware.GetRequestID(r.Context())})
		return
	}

	if err := h.Types.Delete(r.Context(), id); err != nil {
		response.ErrorJSON(w, http.StatusBadRequest, response.Error{Code: "VALIDATION_ERROR", Message: err.Error(), RequestID: middleware.GetRequestID(r.Context())})
		return
	}

	response.JSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}
