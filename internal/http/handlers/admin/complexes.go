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

type ComplexHandler struct {
	Complexes *repositories.ComplexRepository
	Cities    *repositories.CityRepository
	Service   *services.ComplexService
}

type complexCreateRequest struct {
	Name      string `json:"name"`
	Address   string `json:"address"`
	CityID    string `json:"city_id"`
	Status    string `json:"status"`
	Threshold int    `json:"threshold_n"`
}

type statusUpdateRequest struct {
	Status string `json:"status"`
}

func (h ComplexHandler) List(w http.ResponseWriter, r *http.Request) {
	items, err := h.Complexes.List(r.Context(), "", "", "", false, 100, 0)
	if err != nil {
		response.ErrorJSON(w, http.StatusInternalServerError, response.Error{Code: "INTERNAL_ERROR", Message: "failed to list", RequestID: middleware.GetRequestID(r.Context())})
		return
	}
	response.JSON(w, http.StatusOK, map[string]any{"items": items})
}

func (h ComplexHandler) HandleCollection(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.List(w, r)
	case http.MethodPost:
		h.Create(w, r)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (h ComplexHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req complexCreateRequest
	if err := handlers.DecodeJSON(r, &req); err != nil {
		response.ErrorJSON(w, http.StatusBadRequest, response.Error{Code: "VALIDATION_ERROR", Message: "invalid payload", RequestID: middleware.GetRequestID(r.Context())})
		return
	}

	id, err := services.NewID()
	if err != nil {
		response.ErrorJSON(w, http.StatusInternalServerError, response.Error{Code: "INTERNAL_ERROR", Message: "failed to create", RequestID: middleware.GetRequestID(r.Context())})
		return
	}

	complex := repositories.ResidentialComplex{
		ID:              id,
		Name:            req.Name,
		Address:         req.Address,
		CityID:          req.CityID,
		Status:          req.Status,
		Threshold:       defaultThreshold(req.Threshold),
		CurrentRequests: 0,
	}

	city, err := h.Cities.Get(r.Context(), req.CityID)
	if err != nil {
		response.ErrorJSON(w, http.StatusBadRequest, response.Error{Code: "VALIDATION_ERROR", Message: "invalid city", RequestID: middleware.GetRequestID(r.Context())})
		return
	}
	complex.City = city.Name

	if err := h.Complexes.Create(r.Context(), complex); err != nil {
		response.ErrorJSON(w, http.StatusBadRequest, response.Error{Code: "VALIDATION_ERROR", Message: err.Error(), RequestID: middleware.GetRequestID(r.Context())})
		return
	}

	response.JSON(w, http.StatusCreated, complex)
}

func defaultThreshold(value int) int {
	if value == 0 {
		return 30
	}
	return value
}

func (h ComplexHandler) UpdateStatus(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimSuffix(strings.TrimPrefix(r.URL.Path, "/api/v1/admin/complexes/"), "/status")
	if id == "" {
		response.ErrorJSON(w, http.StatusNotFound, response.Error{Code: "NOT_FOUND", Message: "complex not found", RequestID: middleware.GetRequestID(r.Context())})
		return
	}

	var req statusUpdateRequest
	if err := handlers.DecodeJSON(r, &req); err != nil {
		response.ErrorJSON(w, http.StatusBadRequest, response.Error{Code: "VALIDATION_ERROR", Message: "invalid payload", RequestID: middleware.GetRequestID(r.Context())})
		return
	}

	complex, err := h.Complexes.Get(r.Context(), id)
	if err != nil {
		response.ErrorJSON(w, http.StatusNotFound, response.Error{Code: "NOT_FOUND", Message: "complex not found", RequestID: middleware.GetRequestID(r.Context())})
		return
	}

	if err := h.Complexes.UpdateStatusAndRequests(r.Context(), id, req.Status, complex.CurrentRequests); err != nil {
		response.ErrorJSON(w, http.StatusInternalServerError, response.Error{Code: "INTERNAL_ERROR", Message: "failed to update", RequestID: middleware.GetRequestID(r.Context())})
		return
	}

	response.JSON(w, http.StatusOK, map[string]string{"status": req.Status})
}
