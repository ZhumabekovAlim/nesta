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
	Type        string `json:"type"`
	EntityID    string `json:"entity_id"`
	Description string `json:"description"`
}

func (h Handler) InitRobokassa(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		response.ErrorJSON(w, http.StatusMethodNotAllowed, response.Error{Code: "METHOD_NOT_ALLOWED", Message: "method not allowed", RequestID: middleware.GetRequestID(r.Context())})
		return
	}
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		response.ErrorJSON(w, http.StatusUnauthorized, response.Error{Code: "UNAUTHORIZED", Message: "unauthorized", RequestID: middleware.GetRequestID(r.Context())})
		return
	}
	var req initRequest
	if err := handlers.DecodeJSON(r, &req); err != nil {
		response.ErrorJSON(w, http.StatusBadRequest, response.Error{Code: "VALIDATION_ERROR", Message: "invalid payload", RequestID: middleware.GetRequestID(r.Context())})
		return
	}
	result, err := h.Payments.InitRobokassa(r.Context(), services.RobokassaInitRequest{
		Type: req.Type, EntityID: req.EntityID, UserID: userID, Description: req.Description,
	})
	if err != nil {
		response.ErrorJSON(w, http.StatusBadRequest, response.Error{Code: "PAYMENT_INIT_FAILED", Message: err.Error(), RequestID: middleware.GetRequestID(r.Context())})
		return
	}
	response.JSON(w, http.StatusCreated, result)
}

func (h Handler) ResultRobokassa(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	shp := map[string]string{}
	raw := map[string]string{}
	for key, values := range r.Form {
		if len(values) == 0 {
			continue
		}
		raw[key] = values[0]
		if strings.HasPrefix(key, "Shp_") {
			shp[key] = values[0]
		}
	}
	okBody, err := h.Payments.HandleRobokassaResult(r.Context(), services.RobokassaCallback{
		OutSum: r.FormValue("OutSum"), InvID: r.FormValue("InvId"), SignatureValue: r.FormValue("SignatureValue"), Shp: shp, RawPayload: raw,
	})
	if err != nil {
		http.Error(w, "bad sign", http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", "text/plain")
	_, _ = w.Write([]byte(okBody))
}

func (h Handler) SuccessRobokassa(w http.ResponseWriter, r *http.Request) {
	_ = r.ParseForm()
	shp := extractShp(r)
	err := h.Payments.VerifySuccessSignature(r.FormValue("OutSum"), r.FormValue("InvId"), r.FormValue("SignatureValue"), shp)
	if err != nil {
		response.ErrorJSON(w, http.StatusBadRequest, response.Error{Code: "INVALID_SIGNATURE", Message: "invalid signature", RequestID: middleware.GetRequestID(r.Context())})
		return
	}
	invID := r.FormValue("InvId")
	response.JSON(w, http.StatusOK, map[string]string{
		"message": "Оплата принята, проверяем статус.",
		"inv_id":  invID,
	})
}

func (h Handler) FailRobokassa(w http.ResponseWriter, r *http.Request) {
	_ = r.ParseForm()
	invID := r.FormValue("InvId")
	response.JSON(w, http.StatusOK, map[string]string{
		"message": "Платеж отменен или завершился ошибкой.",
		"inv_id":  invID,
	})
}

func extractShp(r *http.Request) map[string]string {
	shp := map[string]string{}
	for key, values := range r.Form {
		if strings.HasPrefix(key, "Shp_") && len(values) > 0 {
			shp[key] = values[0]
		}
	}
	return shp
}
