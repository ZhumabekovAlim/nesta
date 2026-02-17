package api

import (
	"net/http"

	"nesta/internal/http/middleware"
	"nesta/internal/http/response"
	"nesta/internal/repositories"
)

type CityHandler struct {
	Cities *repositories.CityRepository
}

func (h CityHandler) List(w http.ResponseWriter, r *http.Request) {
	items, err := h.Cities.ListActive(r.Context())
	if err != nil {
		response.ErrorJSON(w, http.StatusInternalServerError, response.Error{Code: "INTERNAL_ERROR", Message: "failed to list", RequestID: middleware.GetRequestID(r.Context())})
		return
	}

	response.JSON(w, http.StatusOK, map[string]any{"items": items})
}
