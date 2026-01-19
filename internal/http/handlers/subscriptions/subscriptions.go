package subscriptions

import (
	"encoding/json"
	"net/http"
	"strings"

	"nesta/internal/http/handlers"
	"nesta/internal/http/middleware"
	"nesta/internal/http/response"
	"nesta/internal/repositories"
	"nesta/internal/services"
)

type Handler struct {
	Service       *services.SubscriptionService
	Subscriptions *repositories.SubscriptionRepository
}

type createRequest struct {
	PlanID       string         `json:"plan_id"`
	ComplexID    string         `json:"complex_id"`
	Address      map[string]any `json:"address_json"`
	TimeWindow   string         `json:"time_window"`
	Instructions string         `json:"instructions"`
}

type actionRequest struct {
	Action string `json:"action"`
}

func (h Handler) Create(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		response.ErrorJSON(w, http.StatusUnauthorized, response.Error{Code: "UNAUTHORIZED", Message: "unauthorized", RequestID: middleware.GetRequestID(r.Context())})
		return
	}

	var req createRequest
	if err := handlers.DecodeJSON(r, &req); err != nil {
		response.ErrorJSON(w, http.StatusBadRequest, response.Error{Code: "VALIDATION_ERROR", Message: "invalid payload", RequestID: middleware.GetRequestID(r.Context())})
		return
	}

	addressRaw, err := json.Marshal(req.Address)
	if err != nil {
		response.ErrorJSON(w, http.StatusBadRequest, response.Error{Code: "VALIDATION_ERROR", Message: "invalid address", RequestID: middleware.GetRequestID(r.Context())})
		return
	}

	result, err := h.Service.Create(r.Context(), userID, req.ComplexID, req.PlanID, addressRaw, req.TimeWindow, req.Instructions)
	if err != nil {
		response.ErrorJSON(w, http.StatusBadRequest, response.Error{Code: "VALIDATION_ERROR", Message: err.Error(), RequestID: middleware.GetRequestID(r.Context())})
		return
	}

	response.JSON(w, http.StatusCreated, map[string]any{
		"subscription":     result.Subscription,
		"payment_required": result.RequiresPayment,
	})
}

func (h Handler) ListMine(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		response.ErrorJSON(w, http.StatusUnauthorized, response.Error{Code: "UNAUTHORIZED", Message: "unauthorized", RequestID: middleware.GetRequestID(r.Context())})
		return
	}

	subs, err := h.Subscriptions.ListByUser(r.Context(), userID)
	if err != nil {
		response.ErrorJSON(w, http.StatusInternalServerError, response.Error{Code: "INTERNAL_ERROR", Message: "failed to list", RequestID: middleware.GetRequestID(r.Context())})
		return
	}

	response.JSON(w, http.StatusOK, map[string]any{"items": subs})
}

func (h Handler) Update(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/api/v1/subscriptions/")
	if id == "" {
		response.ErrorJSON(w, http.StatusNotFound, response.Error{Code: "NOT_FOUND", Message: "subscription not found", RequestID: middleware.GetRequestID(r.Context())})
		return
	}

	var req actionRequest
	if err := handlers.DecodeJSON(r, &req); err != nil {
		response.ErrorJSON(w, http.StatusBadRequest, response.Error{Code: "VALIDATION_ERROR", Message: "invalid payload", RequestID: middleware.GetRequestID(r.Context())})
		return
	}

	status := ""
	switch req.Action {
	case "cancel":
		status = "CANCELED"
	case "pause":
		status = "PAUSED"
	case "resume":
		status = "ACTIVE"
	default:
		response.ErrorJSON(w, http.StatusBadRequest, response.Error{Code: "VALIDATION_ERROR", Message: "invalid action", RequestID: middleware.GetRequestID(r.Context())})
		return
	}

	if err := h.Service.UpdateStatus(r.Context(), id, status); err != nil {
		response.ErrorJSON(w, http.StatusBadRequest, response.Error{Code: "VALIDATION_ERROR", Message: err.Error(), RequestID: middleware.GetRequestID(r.Context())})
		return
	}

	response.JSON(w, http.StatusOK, map[string]string{"status": status})
}
