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

type PlanHandler struct {
	Plans *repositories.PlanRepository
}

type planRequest struct {
	Name        string `json:"name"`
	PriceCents  int    `json:"price_cents"`
	Frequency   string `json:"frequency"`
	BagsPerDay  int    `json:"bags_per_day"`
	Description string `json:"description"`
	IsActive    bool   `json:"is_active"`
}

func (h PlanHandler) List(w http.ResponseWriter, r *http.Request) {
	plans, err := h.Plans.ListActive(r.Context())
	if err != nil {
		response.ErrorJSON(w, http.StatusInternalServerError, response.Error{Code: "INTERNAL_ERROR", Message: "failed to list", RequestID: middleware.GetRequestID(r.Context())})
		return
	}
	response.JSON(w, http.StatusOK, map[string]any{"items": plans})
}

func (h PlanHandler) HandleCollection(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.List(w, r)
	case http.MethodPost:
		h.Create(w, r)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (h PlanHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req planRequest
	if err := handlers.DecodeJSON(r, &req); err != nil {
		response.ErrorJSON(w, http.StatusBadRequest, response.Error{Code: "VALIDATION_ERROR", Message: "invalid payload", RequestID: middleware.GetRequestID(r.Context())})
		return
	}

	id, err := services.NewID()
	if err != nil {
		response.ErrorJSON(w, http.StatusInternalServerError, response.Error{Code: "INTERNAL_ERROR", Message: "failed to create", RequestID: middleware.GetRequestID(r.Context())})
		return
	}

	plan := repositories.Plan{
		ID:         id,
		Name:       req.Name,
		PriceCents: req.PriceCents,
		Frequency:  req.Frequency,
		BagsPerDay: req.BagsPerDay,
		IsActive:   req.IsActive,
	}
	if req.Description != "" {
		plan.Description = sql.NullString{String: req.Description, Valid: true}
	}

	if err := h.Plans.Create(r.Context(), plan); err != nil {
		response.ErrorJSON(w, http.StatusBadRequest, response.Error{Code: "VALIDATION_ERROR", Message: err.Error(), RequestID: middleware.GetRequestID(r.Context())})
		return
	}

	response.JSON(w, http.StatusCreated, plan)
}

func (h PlanHandler) Update(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/api/v1/admin/plans/")
	if id == "" {
		response.ErrorJSON(w, http.StatusNotFound, response.Error{Code: "NOT_FOUND", Message: "plan not found", RequestID: middleware.GetRequestID(r.Context())})
		return
	}

	var req planRequest
	if err := handlers.DecodeJSON(r, &req); err != nil {
		response.ErrorJSON(w, http.StatusBadRequest, response.Error{Code: "VALIDATION_ERROR", Message: "invalid payload", RequestID: middleware.GetRequestID(r.Context())})
		return
	}

	plan := repositories.Plan{
		ID:         id,
		Name:       req.Name,
		PriceCents: req.PriceCents,
		Frequency:  req.Frequency,
		BagsPerDay: req.BagsPerDay,
		IsActive:   req.IsActive,
	}
	if req.Description != "" {
		plan.Description = sql.NullString{String: req.Description, Valid: true}
	}

	if err := h.Plans.Update(r.Context(), plan); err != nil {
		response.ErrorJSON(w, http.StatusBadRequest, response.Error{Code: "VALIDATION_ERROR", Message: err.Error(), RequestID: middleware.GetRequestID(r.Context())})
		return
	}

	response.JSON(w, http.StatusOK, plan)
}
