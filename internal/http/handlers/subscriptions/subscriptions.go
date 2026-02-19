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
	Addresses     *repositories.AddressRepository
}

type createRequest struct {
	PlanID       string `json:"plan_id"`
	AddressID    string `json:"address_id"`
	TimeWindow   string `json:"time_window"`
	Instructions string `json:"instructions"`
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

	if strings.TrimSpace(req.AddressID) == "" {
		response.ErrorJSON(w, http.StatusBadRequest, response.Error{Code: "VALIDATION_ERROR", Message: "address required", RequestID: middleware.GetRequestID(r.Context())})
		return
	}

	result, err := h.Service.Create(r.Context(), userID, strings.TrimSpace(req.AddressID), req.PlanID, req.TimeWindow, req.Instructions)
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

	addresses, err := h.Addresses.ListByUser(r.Context(), userID)
	if err != nil {
		response.ErrorJSON(w, http.StatusInternalServerError, response.Error{Code: "INTERNAL_ERROR", Message: "failed to list", RequestID: middleware.GetRequestID(r.Context())})
		return
	}

	subs, err := h.Subscriptions.ListByUser(r.Context(), userID)
	if err != nil {
		response.ErrorJSON(w, http.StatusInternalServerError, response.Error{Code: "INTERNAL_ERROR", Message: "failed to list", RequestID: middleware.GetRequestID(r.Context())})
		return
	}

	subByAddress := make(map[string]repositories.Subscription, len(subs))
	for _, sub := range subs {
		subByAddress[sub.AddressID] = sub
	}

	items := make([]map[string]any, 0, len(addresses))
	for _, address := range addresses {
		var addressPayload any
		if len(address.AddressJSON) > 0 {
			if err := json.Unmarshal(address.AddressJSON, &addressPayload); err != nil {
				addressPayload = nil
			}
		}
		entry := map[string]any{
			"address": map[string]any{
				"id":             address.ID,
				"user_id":        address.UserID,
				"name":           address.Name,
				"complex_id":     address.ComplexID,
				"city_id":        address.CityID,
				"time_window_id": nil,
				"time_window":    address.TimeWindow,
				"address_json":   addressPayload,
				"created_at":     address.CreatedAt,
			},
			"subscription": nil,
		}
		if address.TimeWindowID.Valid {
			entry["address"].(map[string]any)["time_window_id"] = address.TimeWindowID.String
		}
		if sub, ok := subByAddress[address.ID]; ok {
			entry["subscription"] = sub
		}
		items = append(items, entry)
	}

	response.JSON(w, http.StatusOK, map[string]any{"items": items})
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
