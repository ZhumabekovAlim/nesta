package admin

import (
	"database/sql"
	"net/http"
	"strings"
	"time"

	"nesta/internal/http/handlers"
	"nesta/internal/http/middleware"
	"nesta/internal/http/response"
	"nesta/internal/repositories"
	"nesta/internal/services"
)

type PickupLogHandler struct {
	Logs *repositories.PickupLogRepository
}

type pickupRequest struct {
	SubscriptionID string `json:"subscription_id"`
	PickupDate     string `json:"pickup_date"`
	Status         string `json:"status"`
	Comment        string `json:"comment"`
	Reason         string `json:"reason"`
}

func (h PickupLogHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req pickupRequest
	if err := handlers.DecodeJSON(r, &req); err != nil {
		response.ErrorJSON(w, http.StatusBadRequest, response.Error{Code: "VALIDATION_ERROR", Message: "invalid payload", RequestID: middleware.GetRequestID(r.Context())})
		return
	}

	pickupDate, err := time.Parse("2006-01-02", req.PickupDate)
	if err != nil {
		response.ErrorJSON(w, http.StatusBadRequest, response.Error{Code: "VALIDATION_ERROR", Message: "invalid date", RequestID: middleware.GetRequestID(r.Context())})
		return
	}

	id, err := services.NewID()
	if err != nil {
		response.ErrorJSON(w, http.StatusInternalServerError, response.Error{Code: "INTERNAL_ERROR", Message: "failed to create", RequestID: middleware.GetRequestID(r.Context())})
		return
	}

	log := repositories.PickupLog{
		ID:             id,
		SubscriptionID: req.SubscriptionID,
		PickupDate:     pickupDate,
		Status:         req.Status,
	}
	if req.Comment != "" {
		log.Comment = sql.NullString{String: req.Comment, Valid: true}
	}
	if req.Reason != "" {
		log.Reason = sql.NullString{String: req.Reason, Valid: true}
	}

	if err := h.Logs.Create(r.Context(), log); err != nil {
		response.ErrorJSON(w, http.StatusBadRequest, response.Error{Code: "VALIDATION_ERROR", Message: err.Error(), RequestID: middleware.GetRequestID(r.Context())})
		return
	}

	response.JSON(w, http.StatusCreated, log)
}

func (h PickupLogHandler) HandleCollection(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		h.Create(w, r)
		return
	}
	w.WriteHeader(http.StatusMethodNotAllowed)
}

func (h PickupLogHandler) Update(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/api/v1/admin/pickup-logs/")
	if id == "" {
		response.ErrorJSON(w, http.StatusNotFound, response.Error{Code: "NOT_FOUND", Message: "pickup log not found", RequestID: middleware.GetRequestID(r.Context())})
		return
	}

	var req pickupRequest
	if err := handlers.DecodeJSON(r, &req); err != nil {
		response.ErrorJSON(w, http.StatusBadRequest, response.Error{Code: "VALIDATION_ERROR", Message: "invalid payload", RequestID: middleware.GetRequestID(r.Context())})
		return
	}

	log := repositories.PickupLog{
		ID:     id,
		Status: req.Status,
	}
	if req.Comment != "" {
		log.Comment = sql.NullString{String: req.Comment, Valid: true}
	}
	if req.Reason != "" {
		log.Reason = sql.NullString{String: req.Reason, Valid: true}
	}

	if err := h.Logs.Update(r.Context(), log); err != nil {
		response.ErrorJSON(w, http.StatusBadRequest, response.Error{Code: "VALIDATION_ERROR", Message: err.Error(), RequestID: middleware.GetRequestID(r.Context())})
		return
	}

	response.JSON(w, http.StatusOK, log)
}
