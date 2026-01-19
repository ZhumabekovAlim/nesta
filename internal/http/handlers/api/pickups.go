package api

import (
	"net/http"
	"strings"

	"nesta/internal/http/middleware"
	"nesta/internal/http/response"
	"nesta/internal/repositories"
)

type PickupHandler struct {
	Logs *repositories.PickupLogRepository
}

func (h PickupHandler) ListBySubscription(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/api/v1/pickups/")
	if id == "" {
		response.ErrorJSON(w, http.StatusNotFound, response.Error{Code: "NOT_FOUND", Message: "subscription not found", RequestID: middleware.GetRequestID(r.Context())})
		return
	}

	logs, err := h.Logs.ListBySubscription(r.Context(), id)
	if err != nil {
		response.ErrorJSON(w, http.StatusInternalServerError, response.Error{Code: "INTERNAL_ERROR", Message: "failed to list", RequestID: middleware.GetRequestID(r.Context())})
		return
	}

	response.JSON(w, http.StatusOK, map[string]any{"items": logs})
}
