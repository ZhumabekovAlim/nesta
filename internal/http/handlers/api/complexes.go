package api

import (
	"database/sql"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"nesta/internal/auth"
	"nesta/internal/http/handlers"
	"nesta/internal/http/middleware"
	"nesta/internal/http/response"
	"nesta/internal/repositories"
	"nesta/internal/services"
)

type ComplexHandler struct {
	Complexes         *repositories.ComplexRepository
	Requests          *repositories.ComplexRequestRepository
	Users             *repositories.UserRepository
	SubscriptionTypes *repositories.SubscriptionTypeRepository
	TimeWindows       *repositories.TimeWindowRepository
	Service           *services.ComplexService
	JWTSecret         string
}

type complexGetResponse struct {
	repositories.ResidentialComplex
	HasRequested      bool                            `json:"has_requested"`
	SubscriptionTypes []repositories.SubscriptionType `json:"subscription_types"`
	TimeWindows       []repositories.TimeWindow       `json:"time_windows"`
}

type requestCreate struct {
	Phone string `json:"phone"`
}

func (h ComplexHandler) List(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	limit := parseInt(query.Get("limit"), 20)
	offset := parseInt(query.Get("offset"), 0)
	onlyActive := true
	if query.Get("status") != "" {
		onlyActive = false
	}
	if query.Get("only_active") != "" {
		onlyActive = query.Get("only_active") == "1"
	}
	cityID := query.Get("city_id")
	if cityID == "" {
		cityID = query.Get("city")
	}
	items, err := h.Complexes.List(r.Context(), query.Get("search"), query.Get("status"), cityID, onlyActive, limit, offset)
	if err != nil {
		response.ErrorJSON(w, http.StatusInternalServerError, response.Error{Code: "INTERNAL_ERROR", Message: "failed to list", RequestID: middleware.GetRequestID(r.Context())})
		return
	}

	response.JSON(w, http.StatusOK, map[string]any{"items": items, "limit": limit, "offset": offset})
}

func (h ComplexHandler) Get(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/api/v1/complexes/")

	if id == "" {
		response.ErrorJSON(w, http.StatusNotFound, response.Error{Code: "NOT_FOUND", Message: "complex not found", RequestID: middleware.GetRequestID(r.Context())})
		return
	}

	item, err := h.Complexes.Get(r.Context(), id)
	if err != nil {
		response.ErrorJSON(w, http.StatusNotFound, response.Error{Code: "NOT_FOUND", Message: "complex not found", RequestID: middleware.GetRequestID(r.Context())})
		return
	}

	hasRequested := false
	if h.Users != nil {
		authHeader := r.Header.Get("Authorization")
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) == 2 && strings.EqualFold(parts[0], "Bearer") {
			claims, parseErr := auth.ParseToken(h.JWTSecret, parts[1])
			if parseErr == nil && claims.Subject != "" {
				user, userErr := h.Users.FindByID(r.Context(), claims.Subject)
				if userErr == nil {
					exists, existsErr := h.Requests.ExistsByComplexAndPhone(r.Context(), id, user.Phone)
					if existsErr == nil {
						hasRequested = exists
					}
				} else if userErr != nil && !errors.Is(userErr, sql.ErrNoRows) {
					response.ErrorJSON(w, http.StatusInternalServerError, response.Error{Code: "INTERNAL_ERROR", Message: "failed to get complex", RequestID: middleware.GetRequestID(r.Context())})
					return
				}
			}
		}
	}

	subscriptionTypes := make([]repositories.SubscriptionType, 0)
	if h.SubscriptionTypes != nil {
		subscriptionTypes, err = h.SubscriptionTypes.ListActive(r.Context())
		if err != nil {
			response.ErrorJSON(w, http.StatusInternalServerError, response.Error{Code: "INTERNAL_ERROR", Message: "failed to get subscription types", RequestID: middleware.GetRequestID(r.Context())})
			return
		}
	}

	timeWindows := make([]repositories.TimeWindow, 0)
	if h.TimeWindows != nil {
		timeWindows, err = h.TimeWindows.ListActive(r.Context())
		if err != nil {
			response.ErrorJSON(w, http.StatusInternalServerError, response.Error{Code: "INTERNAL_ERROR", Message: "failed to get time windows", RequestID: middleware.GetRequestID(r.Context())})
			return
		}
	}

	response.JSON(w, http.StatusOK, complexGetResponse{
		ResidentialComplex: item,
		HasRequested:       hasRequested,
		SubscriptionTypes:  subscriptionTypes,
		TimeWindows:        timeWindows,
	})
}

func (h ComplexHandler) CreateRequest(w http.ResponseWriter, r *http.Request) {
	complexID := strings.TrimSuffix(strings.TrimPrefix(r.URL.Path, "/api/v1/complexes/"), "/request")
	if complexID == "" {
		response.ErrorJSON(w, http.StatusNotFound, response.Error{Code: "NOT_FOUND", Message: "complex not found", RequestID: middleware.GetRequestID(r.Context())})
		return
	}

	authHeader := r.Header.Get("Authorization")
	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		response.ErrorJSON(w, http.StatusUnauthorized, response.Error{Code: "UNAUTHORIZED", Message: "unauthorized", RequestID: middleware.GetRequestID(r.Context())})
		return
	}
	claims, err := auth.ParseToken(h.JWTSecret, parts[1])
	if err != nil || claims.Subject == "" {
		response.ErrorJSON(w, http.StatusUnauthorized, response.Error{Code: "UNAUTHORIZED", Message: "unauthorized", RequestID: middleware.GetRequestID(r.Context())})
		return
	}

	var req requestCreate
	if err := handlers.DecodeJSON(r, &req); err != nil {
		response.ErrorJSON(w, http.StatusBadRequest, response.Error{Code: "VALIDATION_ERROR", Message: "invalid payload", RequestID: middleware.GetRequestID(r.Context())})
		return
	}

	request, _, err := h.Service.CreateRequest(r.Context(), complexID, req.Phone)
	if err != nil {
		response.ErrorJSON(w, http.StatusBadRequest, response.Error{Code: "VALIDATION_ERROR", Message: err.Error(), RequestID: middleware.GetRequestID(r.Context())})
		return
	}

	response.JSON(w, http.StatusCreated, request)
}

func (h ComplexHandler) HandleItem(w http.ResponseWriter, r *http.Request) {
	if strings.HasSuffix(r.URL.Path, "/request") {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		h.CreateRequest(w, r)
		return
	}

	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	h.Get(w, r)
}

func parseInt(value string, fallback int) int {
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}
	return parsed
}
