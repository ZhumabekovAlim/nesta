package auth

import (
	"errors"
	"net/http"

	"github.com/rs/zerolog/log"

	"nesta/internal/http/handlers"
	"nesta/internal/http/middleware"
	"nesta/internal/http/response"
	"nesta/internal/services"
)

type Handler struct {
	Auth *services.AuthService
}

type sendOTPRequest struct {
	Phone string `json:"phone"`
}

type verifyOTPRequest struct {
	Phone string `json:"phone"`
	Code  string `json:"code"`
}

type refreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}

func (h Handler) SendOTP(w http.ResponseWriter, r *http.Request) {
	requestID := middleware.GetRequestID(r.Context())
	var req sendOTPRequest
	if err := handlers.DecodeJSON(r, &req); err != nil {
		log.Warn().Err(err).Str("request_id", requestID).Msg("otp send invalid payload")
		response.ErrorJSON(w, http.StatusBadRequest, response.Error{Code: "VALIDATION_ERROR", Message: "invalid payload", RequestID: requestID})
		return
	}

	result, err := h.Auth.SendOTP(r.Context(), req.Phone)
	if err != nil {
		code := "VALIDATION_ERROR"
		status := http.StatusBadRequest
		if errors.Is(err, services.ErrRateLimited) || errors.Is(err, services.ErrSMSRateLimited) {
			code = "RATE_LIMITED"
			status = http.StatusTooManyRequests
		} else if errors.Is(err, services.ErrSMSDeliveryError) {
			code = "UPSTREAM_ERROR"
			status = http.StatusBadGateway
		}
		log.Warn().Err(err).Str("request_id", requestID).Str("response_code", code).Int("http_status", status).Msg("otp send request failed")
		response.ErrorJSON(w, status, response.Error{Code: code, Message: err.Error(), RequestID: requestID})
		return
	}

	log.Info().Str("request_id", requestID).Bool("dev_code_in_response", result.Code != nil).Msg("otp send request succeeded")
	response.JSON(w, http.StatusOK, map[string]any{
		"status":     "sent",
		"expires_at": result.ExpiresAt,
		"dev_code":   result.Code,
	})
}

func (h Handler) VerifyOTP(w http.ResponseWriter, r *http.Request) {
	var req verifyOTPRequest
	if err := handlers.DecodeJSON(r, &req); err != nil {
		response.ErrorJSON(w, http.StatusBadRequest, response.Error{Code: "VALIDATION_ERROR", Message: "invalid payload", RequestID: middleware.GetRequestID(r.Context())})
		return
	}

	pair, err := h.Auth.VerifyOTP(r.Context(), req.Phone, req.Code)
	if err != nil {
		response.ErrorJSON(w, http.StatusBadRequest, response.Error{Code: "VALIDATION_ERROR", Message: err.Error(), RequestID: middleware.GetRequestID(r.Context())})
		return
	}

	response.JSON(w, http.StatusOK, map[string]any{
		"access_token":  pair.AccessToken,
		"refresh_token": pair.RefreshToken,
		"expires_at":    pair.ExpiresAt,
	})
}

func (h Handler) Refresh(w http.ResponseWriter, r *http.Request) {
	var req refreshRequest
	if err := handlers.DecodeJSON(r, &req); err != nil {
		response.ErrorJSON(w, http.StatusBadRequest, response.Error{Code: "VALIDATION_ERROR", Message: "invalid payload", RequestID: middleware.GetRequestID(r.Context())})
		return
	}

	pair, err := h.Auth.Refresh(r.Context(), req.RefreshToken)
	if err != nil {
		response.ErrorJSON(w, http.StatusUnauthorized, response.Error{Code: "UNAUTHORIZED", Message: err.Error(), RequestID: middleware.GetRequestID(r.Context())})
		return
	}

	response.JSON(w, http.StatusOK, map[string]any{
		"access_token":  pair.AccessToken,
		"refresh_token": pair.RefreshToken,
		"expires_at":    pair.ExpiresAt,
	})
}

func (h Handler) Logout(w http.ResponseWriter, r *http.Request) {
	var req refreshRequest
	if err := handlers.DecodeJSON(r, &req); err != nil {
		response.ErrorJSON(w, http.StatusBadRequest, response.Error{Code: "VALIDATION_ERROR", Message: "invalid payload", RequestID: middleware.GetRequestID(r.Context())})
		return
	}

	if err := h.Auth.Logout(r.Context(), req.RefreshToken); err != nil {
		response.ErrorJSON(w, http.StatusBadRequest, response.Error{Code: "VALIDATION_ERROR", Message: err.Error(), RequestID: middleware.GetRequestID(r.Context())})
		return
	}

	response.JSON(w, http.StatusOK, map[string]string{"status": "logged_out"})
}
