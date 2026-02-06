package addresses

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
	Service   *services.AddressService
	Addresses *repositories.AddressRepository
}

type addressRequest struct {
	Name      string         `json:"name"`
	ComplexID string         `json:"complex_id"`
	Address   map[string]any `json:"address_json"`
}

func (h Handler) ListMine(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		response.ErrorJSON(w, http.StatusUnauthorized, response.Error{Code: "UNAUTHORIZED", Message: "unauthorized", RequestID: middleware.GetRequestID(r.Context())})
		return
	}

	items, err := h.Addresses.ListByUser(r.Context(), userID)
	if err != nil {
		response.ErrorJSON(w, http.StatusInternalServerError, response.Error{Code: "INTERNAL_ERROR", Message: "failed to list", RequestID: middleware.GetRequestID(r.Context())})
		return
	}

	response.JSON(w, http.StatusOK, map[string]any{"items": jsonRawList(items)})
}

func (h Handler) HandleCollection(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.ListMine(w, r)
	case http.MethodPost:
		h.Create(w, r)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (h Handler) HandleItem(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPatch, http.MethodPut:
		h.Update(w, r)
	case http.MethodDelete:
		h.Delete(w, r)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (h Handler) Create(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		response.ErrorJSON(w, http.StatusUnauthorized, response.Error{Code: "UNAUTHORIZED", Message: "unauthorized", RequestID: middleware.GetRequestID(r.Context())})
		return
	}

	var req addressRequest
	if err := handlers.DecodeJSON(r, &req); err != nil {
		response.ErrorJSON(w, http.StatusBadRequest, response.Error{Code: "VALIDATION_ERROR", Message: "invalid payload", RequestID: middleware.GetRequestID(r.Context())})
		return
	}

	address, err := h.Service.Create(r.Context(), userID, services.AddressInput{
		Name:      strings.TrimSpace(req.Name),
		ComplexID: strings.TrimSpace(req.ComplexID),
		Address:   req.Address,
	})
	if err != nil {
		response.ErrorJSON(w, http.StatusBadRequest, response.Error{Code: "VALIDATION_ERROR", Message: err.Error(), RequestID: middleware.GetRequestID(r.Context())})
		return
	}

	response.JSON(w, http.StatusCreated, map[string]any{"address": jsonRaw(address)})
}

func (h Handler) Update(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		response.ErrorJSON(w, http.StatusUnauthorized, response.Error{Code: "UNAUTHORIZED", Message: "unauthorized", RequestID: middleware.GetRequestID(r.Context())})
		return
	}

	id := strings.TrimPrefix(r.URL.Path, "/api/v1/addresses/")
	if id == "" {
		response.ErrorJSON(w, http.StatusNotFound, response.Error{Code: "NOT_FOUND", Message: "address not found", RequestID: middleware.GetRequestID(r.Context())})
		return
	}

	var req addressRequest
	if err := handlers.DecodeJSON(r, &req); err != nil {
		response.ErrorJSON(w, http.StatusBadRequest, response.Error{Code: "VALIDATION_ERROR", Message: "invalid payload", RequestID: middleware.GetRequestID(r.Context())})
		return
	}

	if err := h.Service.Update(r.Context(), userID, id, services.AddressInput{
		Name:      strings.TrimSpace(req.Name),
		ComplexID: strings.TrimSpace(req.ComplexID),
		Address:   req.Address,
	}); err != nil {
		response.ErrorJSON(w, http.StatusBadRequest, response.Error{Code: "VALIDATION_ERROR", Message: err.Error(), RequestID: middleware.GetRequestID(r.Context())})
		return
	}

	response.JSON(w, http.StatusOK, map[string]string{"status": "updated"})
}

func (h Handler) Delete(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		response.ErrorJSON(w, http.StatusUnauthorized, response.Error{Code: "UNAUTHORIZED", Message: "unauthorized", RequestID: middleware.GetRequestID(r.Context())})
		return
	}

	id := strings.TrimPrefix(r.URL.Path, "/api/v1/addresses/")
	if id == "" {
		response.ErrorJSON(w, http.StatusNotFound, response.Error{Code: "NOT_FOUND", Message: "address not found", RequestID: middleware.GetRequestID(r.Context())})
		return
	}

	deleted, err := h.Service.Delete(r.Context(), userID, id)
	if err != nil {
		response.ErrorJSON(w, http.StatusInternalServerError, response.Error{Code: "INTERNAL_ERROR", Message: "failed to delete", RequestID: middleware.GetRequestID(r.Context())})
		return
	}
	if !deleted {
		response.ErrorJSON(w, http.StatusNotFound, response.Error{Code: "NOT_FOUND", Message: "address not found", RequestID: middleware.GetRequestID(r.Context())})
		return
	}

	response.JSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

func jsonRawList(items []repositories.Address) []map[string]any {
	out := make([]map[string]any, 0, len(items))
	for _, item := range items {
		out = append(out, jsonRaw(item))
	}
	return out
}

func jsonRaw(address repositories.Address) map[string]any {
	payload := map[string]any{
		"id":         address.ID,
		"user_id":    address.UserID,
		"name":       address.Name,
		"complex_id": address.ComplexID,
		"created_at": address.CreatedAt,
	}
	if len(address.AddressJSON) == 0 {
		payload["address_json"] = nil
		return payload
	}
	var raw any
	if err := json.Unmarshal(address.AddressJSON, &raw); err != nil {
		payload["address_json"] = nil
		return payload
	}
	payload["address_json"] = raw
	return payload
}
