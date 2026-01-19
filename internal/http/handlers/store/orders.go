package store

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

type OrderHandler struct {
	Service *services.OrderService
	Orders  *repositories.OrderRepository
}

type orderItemInput struct {
	ProductID string `json:"product_id"`
	Quantity  int    `json:"quantity"`
}

type createOrderRequest struct {
	Items   []orderItemInput `json:"items"`
	Address map[string]any   `json:"address_json"`
	Comment string           `json:"comment"`
}

func (h OrderHandler) Create(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		response.ErrorJSON(w, http.StatusUnauthorized, response.Error{Code: "UNAUTHORIZED", Message: "unauthorized", RequestID: middleware.GetRequestID(r.Context())})
		return
	}

	var req createOrderRequest
	if err := handlers.DecodeJSON(r, &req); err != nil {
		response.ErrorJSON(w, http.StatusBadRequest, response.Error{Code: "VALIDATION_ERROR", Message: "invalid payload", RequestID: middleware.GetRequestID(r.Context())})
		return
	}

	addressRaw, err := json.Marshal(req.Address)
	if err != nil {
		response.ErrorJSON(w, http.StatusBadRequest, response.Error{Code: "VALIDATION_ERROR", Message: "invalid address", RequestID: middleware.GetRequestID(r.Context())})
		return
	}

	items := make([]services.OrderItemInput, 0, len(req.Items))
	for _, item := range req.Items {
		items = append(items, services.OrderItemInput{ProductID: item.ProductID, Quantity: item.Quantity})
	}

	order, orderItems, err := h.Service.Create(r.Context(), userID, addressRaw, req.Comment, items)
	if err != nil {
		response.ErrorJSON(w, http.StatusBadRequest, response.Error{Code: "VALIDATION_ERROR", Message: err.Error(), RequestID: middleware.GetRequestID(r.Context())})
		return
	}

	response.JSON(w, http.StatusCreated, map[string]any{"order": order, "items": orderItems})
}

func (h OrderHandler) ListMine(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		response.ErrorJSON(w, http.StatusUnauthorized, response.Error{Code: "UNAUTHORIZED", Message: "unauthorized", RequestID: middleware.GetRequestID(r.Context())})
		return
	}

	orders, err := h.Orders.ListByUser(r.Context(), userID)
	if err != nil {
		response.ErrorJSON(w, http.StatusInternalServerError, response.Error{Code: "INTERNAL_ERROR", Message: "failed to list", RequestID: middleware.GetRequestID(r.Context())})
		return
	}

	response.JSON(w, http.StatusOK, map[string]any{"items": orders})
}

func (h OrderHandler) Get(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/api/v1/orders/")
	if id == "" {
		response.ErrorJSON(w, http.StatusNotFound, response.Error{Code: "NOT_FOUND", Message: "order not found", RequestID: middleware.GetRequestID(r.Context())})
		return
	}

	order, err := h.Orders.Get(r.Context(), id)
	if err != nil {
		response.ErrorJSON(w, http.StatusNotFound, response.Error{Code: "NOT_FOUND", Message: "order not found", RequestID: middleware.GetRequestID(r.Context())})
		return
	}

	response.JSON(w, http.StatusOK, order)
}
