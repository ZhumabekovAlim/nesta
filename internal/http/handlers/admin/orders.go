package admin

import (
	"net/http"
	"strings"

	"nesta/internal/http/handlers"
	"nesta/internal/http/middleware"
	"nesta/internal/http/response"
	"nesta/internal/repositories"
)

type OrderHandler struct {
	Orders *repositories.OrderRepository
}

type orderStatusRequest struct {
	Status string `json:"status"`
}

func (h OrderHandler) List(w http.ResponseWriter, r *http.Request) {
	items, err := h.Orders.ListAll(r.Context())
	if err != nil {
		response.ErrorJSON(w, http.StatusInternalServerError, response.Error{Code: "INTERNAL_ERROR", Message: "failed to list", RequestID: middleware.GetRequestID(r.Context())})
		return
	}
	response.JSON(w, http.StatusOK, map[string]any{"items": items})
}

func (h OrderHandler) HandleCollection(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		h.List(w, r)
		return
	}
	w.WriteHeader(http.StatusMethodNotAllowed)
}

func (h OrderHandler) Update(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/api/v1/admin/orders/")
	if id == "" {
		response.ErrorJSON(w, http.StatusNotFound, response.Error{Code: "NOT_FOUND", Message: "order not found", RequestID: middleware.GetRequestID(r.Context())})
		return
	}

	var req orderStatusRequest
	if err := handlers.DecodeJSON(r, &req); err != nil {
		response.ErrorJSON(w, http.StatusBadRequest, response.Error{Code: "VALIDATION_ERROR", Message: "invalid payload", RequestID: middleware.GetRequestID(r.Context())})
		return
	}

	if err := h.Orders.UpdateStatus(r.Context(), id, req.Status); err != nil {
		response.ErrorJSON(w, http.StatusBadRequest, response.Error{Code: "VALIDATION_ERROR", Message: err.Error(), RequestID: middleware.GetRequestID(r.Context())})
		return
	}

	response.JSON(w, http.StatusOK, map[string]string{"status": req.Status})
}
