package admin

import (
	"net/http"
	"strings"

	"nesta/internal/http/handlers"
	"nesta/internal/http/middleware"
	"nesta/internal/http/response"
	"nesta/internal/repositories"
	"nesta/internal/services"
)

type SubscriptionHandler struct {
	Subscriptions *repositories.SubscriptionRepository
	Service       *services.SubscriptionService
}

type subscriptionActionRequest struct {
	Action string `json:"action"`
}

func (h SubscriptionHandler) List(w http.ResponseWriter, r *http.Request) {
	items, err := h.Subscriptions.ListAll(r.Context())
	if err != nil {
		response.ErrorJSON(w, http.StatusInternalServerError, response.Error{Code: "INTERNAL_ERROR", Message: "failed to list", RequestID: middleware.GetRequestID(r.Context())})
		return
	}
	response.JSON(w, http.StatusOK, map[string]any{"items": items})
}

func (h SubscriptionHandler) HandleCollection(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		h.List(w, r)
		return
	}
	w.WriteHeader(http.StatusMethodNotAllowed)
}

func (h SubscriptionHandler) Update(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/api/v1/admin/subscriptions/")
	if id == "" {
		response.ErrorJSON(w, http.StatusNotFound, response.Error{Code: "NOT_FOUND", Message: "subscription not found", RequestID: middleware.GetRequestID(r.Context())})
		return
	}

	var req subscriptionActionRequest
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
