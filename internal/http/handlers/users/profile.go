package users

import (
	"database/sql"
	"encoding/json"
	"net/http"

	"nesta/internal/http/handlers"
	"nesta/internal/http/middleware"
	"nesta/internal/http/response"
	"nesta/internal/repositories"
)

type Handler struct {
	Users *repositories.UserRepository
}

type updateProfileRequest struct {
	Name           *string        `json:"name"`
	Email          *string        `json:"email"`
	DefaultAddress map[string]any `json:"default_address_json"`
}

func (h Handler) Me(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		response.ErrorJSON(w, http.StatusUnauthorized, response.Error{Code: "UNAUTHORIZED", Message: "unauthorized", RequestID: middleware.GetRequestID(r.Context())})
		return
	}

	user, err := h.Users.FindByID(r.Context(), userID)
	if err != nil {
		response.ErrorJSON(w, http.StatusNotFound, response.Error{Code: "NOT_FOUND", Message: "user not found", RequestID: middleware.GetRequestID(r.Context())})
		return
	}

	response.JSON(w, http.StatusOK, map[string]any{
		"id":                   user.ID,
		"phone":                user.Phone,
		"name":                 user.Name.String,
		"email":                user.Email.String,
		"role":                 user.Role,
		"default_address_json": jsonRaw(user.DefaultAddressRaw),
	})
}

func (h Handler) UpdateMe(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		response.ErrorJSON(w, http.StatusUnauthorized, response.Error{Code: "UNAUTHORIZED", Message: "unauthorized", RequestID: middleware.GetRequestID(r.Context())})
		return
	}

	var req updateProfileRequest
	if err := handlers.DecodeJSON(r, &req); err != nil {
		response.ErrorJSON(w, http.StatusBadRequest, response.Error{Code: "VALIDATION_ERROR", Message: "invalid payload", RequestID: middleware.GetRequestID(r.Context())})
		return
	}

	name := sql.NullString{Valid: false}
	if req.Name != nil {
		name = sql.NullString{String: *req.Name, Valid: true}
	}

	email := sql.NullString{Valid: false}
	if req.Email != nil {
		email = sql.NullString{String: *req.Email, Valid: true}
	}

	addressRaw, err := marshalJSON(req.DefaultAddress)
	if err != nil {
		response.ErrorJSON(w, http.StatusBadRequest, response.Error{Code: "VALIDATION_ERROR", Message: "invalid address", RequestID: middleware.GetRequestID(r.Context())})
		return
	}

	if err := h.Users.UpdateProfile(r.Context(), userID, name, email, addressRaw); err != nil {
		response.ErrorJSON(w, http.StatusInternalServerError, response.Error{Code: "INTERNAL_ERROR", Message: "failed to update", RequestID: middleware.GetRequestID(r.Context())})
		return
	}

	response.JSON(w, http.StatusOK, map[string]string{"status": "updated"})
}

func marshalJSON(input any) ([]byte, error) {
	if input == nil {
		return nil, nil
	}
	return json.Marshal(input)
}

func jsonRaw(raw []byte) any {
	if len(raw) == 0 {
		return nil
	}
	var out any
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil
	}
	return out
}
