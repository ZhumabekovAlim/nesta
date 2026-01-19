package api

import (
	"net/http"

	"nesta/internal/http/middleware"
	"nesta/internal/http/response"
	"nesta/internal/repositories"
)

type PlanHandler struct {
	Plans *repositories.PlanRepository
}

func (h PlanHandler) List(w http.ResponseWriter, r *http.Request) {
	plans, err := h.Plans.ListActive(r.Context())
	if err != nil {
		response.ErrorJSON(w, http.StatusInternalServerError, response.Error{Code: "INTERNAL_ERROR", Message: "failed to list", RequestID: middleware.GetRequestID(r.Context())})
		return
	}

	response.JSON(w, http.StatusOK, map[string]any{"items": plans})
}
