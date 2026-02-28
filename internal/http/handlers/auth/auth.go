package auth

import (
	"errors"
	"net/http"

	"nesta/internal/http/handlers"
	"nesta/internal/http/middleware"
	"nesta/internal/http/response"
	"nesta/internal/services"
)

type Handler struct {
	Auth *services.AuthService
}

type checkPhoneRequest struct {
	Phone string `json:"phone"`
}

type sendOTPRequest struct {
	Phone string `json:"phone"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

type verifyOTPRequest struct {
	Phone string `json:"phone"`
	Code  string `json:"code"`
}

type refreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}

func (h Handler) CheckPhone(w http.ResponseWriter, r *http.Request) {
	var req checkPhoneRequest
	if err := handlers.DecodeJSON(r, &req); err != nil {
		response.ErrorJSON(w, http.StatusBadRequest, response.Error{Code: "VALIDATION_ERROR", Message: "invalid payload", RequestID: middleware.GetRequestID(r.Context())})
		return
	}

	result, err := h.Auth.CheckPhone(r.Context(), req.Phone)
	if err != nil {
		response.ErrorJSON(w, http.StatusBadRequest, response.Error{Code: "VALIDATION_ERROR", Message: err.Error(), RequestID: middleware.GetRequestID(r.Context())})
		return
	}

	response.JSON(w, http.StatusOK, map[string]any{
		"phone":            result.Phone,
		"is_new_user":      result.IsNewUser,
		"requires_profile": result.RequiresProfile,
	})
}

func (h Handler) SendOTP(w http.ResponseWriter, r *http.Request) {
	var req sendOTPRequest
	if err := handlers.DecodeJSON(r, &req); err != nil {
		response.ErrorJSON(w, http.StatusBadRequest, response.Error{Code: "VALIDATION_ERROR", Message: "invalid payload", RequestID: middleware.GetRequestID(r.Context())})
		return
	}

	result, err := h.Auth.SendOTP(r.Context(), req.Phone, req.Name, req.Email)
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
		response.ErrorJSON(w, status, response.Error{Code: code, Message: err.Error(), RequestID: middleware.GetRequestID(r.Context())})
		return
	}

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
