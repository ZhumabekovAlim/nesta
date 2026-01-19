package payments

import (
	"net/http"
	"strings"

	"nesta/internal/http/handlers"
	"nesta/internal/http/middleware"
	"nesta/internal/http/response"
	"nesta/internal/services"
)

type Handler struct {
	Payments *services.PaymentService
}

type initRequest struct {
	Type              string `json:"type"`
	EntityID          string `json:"entity_id"`
	Provider          string `json:"provider"`
	ProviderPaymentID string `json:"provider_payment_id"`
	AmountCents       int    `json:"amount_cents"`
}

type webhookRequest struct {
	ProviderPaymentID string `json:"provider_payment_id"`
	Status            string `json:"status"`
	Payload           any    `json:"payload"`
}

func (h Handler) Init(w http.ResponseWriter, r *http.Request) {
	var req initRequest
	if err := handlers.DecodeJSON(r, &req); err != nil {
		response.ErrorJSON(w, http.StatusBadRequest, response.Error{Code: "VALIDATION_ERROR", Message: "invalid payload", RequestID: middleware.GetRequestID(r.Context())})
		return
	}

	payment, err := h.Payments.Init(r.Context(), services.PaymentInitRequest{
		Type:              req.Type,
		EntityID:          req.EntityID,
		Provider:          req.Provider,
		ProviderPaymentID: req.ProviderPaymentID,
		AmountCents:       req.AmountCents,
	})
	if err != nil {
		response.ErrorJSON(w, http.StatusBadRequest, response.Error{Code: "VALIDATION_ERROR", Message: err.Error(), RequestID: middleware.GetRequestID(r.Context())})
		return
	}

	response.JSON(w, http.StatusCreated, payment)
}

func (h Handler) Webhook(w http.ResponseWriter, r *http.Request) {
	provider := strings.TrimPrefix(r.URL.Path, "/api/v1/payments/webhook/")
	if provider == "" {
		response.ErrorJSON(w, http.StatusNotFound, response.Error{Code: "NOT_FOUND", Message: "provider not found", RequestID: middleware.GetRequestID(r.Context())})
		return
	}

	var req webhookRequest
	if err := handlers.DecodeJSON(r, &req); err != nil {
		response.ErrorJSON(w, http.StatusBadRequest, response.Error{Code: "VALIDATION_ERROR", Message: "invalid payload", RequestID: middleware.GetRequestID(r.Context())})
		return
	}

	if err := h.Payments.HandleWebhook(r.Context(), services.PaymentWebhook{
		Provider:          provider,
		ProviderPaymentID: req.ProviderPaymentID,
		Status:            req.Status,
		Payload:           req.Payload,
	}); err != nil {
		response.ErrorJSON(w, http.StatusBadRequest, response.Error{Code: "PAYMENT_WEBHOOK_INVALID", Message: err.Error(), RequestID: middleware.GetRequestID(r.Context())})
		return
	}

	response.JSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
