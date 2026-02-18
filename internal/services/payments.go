package services

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"net/url"
	"strconv"
	"strings"
	"time"

	"nesta/internal/config"
	"nesta/internal/payments/robokassa"

	"github.com/rs/zerolog/log"
)

type PaymentService struct {
	DB     *sql.DB
	Config config.RobokassaConfig
}

type RobokassaInitRequest struct {
	Type        string
	EntityID    string
	UserID      string
	Description string
}

type RobokassaInitResponse struct {
	PaymentID  string            `json:"payment_id"`
	InvID      int64             `json:"inv_id"`
	Status     string            `json:"status"`
	PaymentURL string            `json:"payment_url"`
	Params     map[string]string `json:"params"`
}

type RobokassaCallback struct {
	OutSum         string
	InvID          string
	SignatureValue string
	Shp            map[string]string
	RawPayload     map[string]string
}

func (s *PaymentService) InitRobokassa(ctx context.Context, req RobokassaInitRequest) (RobokassaInitResponse, error) {
	if s.Config.MerchantLogin == "" || s.Config.PaymentURL == "" {
		return RobokassaInitResponse{}, errors.New("robokassa is not configured")
	}
	if req.Type != "order" && req.Type != "subscription" {
		return RobokassaInitResponse{}, errors.New("unsupported payment type")
	}

	amount, currency, err := s.resolveAmount(ctx, req.Type, req.EntityID)
	if err != nil {
		return RobokassaInitResponse{}, err
	}
	invID, err := s.nextInvID(ctx)
	if err != nil {
		return RobokassaInitResponse{}, err
	}
	paymentID, err := NewID()
	if err != nil {
		return RobokassaInitResponse{}, err
	}

	amountStr := robokassa.FormatAmount(amount, currency)
	password1 := s.Config.Password1
	if s.Config.IsTest && s.Config.TestPassword1 != "" {
		password1 = s.Config.TestPassword1
	}
	shp := map[string]string{"Shp_payment_id": paymentID}
	signature, err := robokassa.SignatureForInit(robokassa.HashAlgorithm(s.Config.HashAlgorithm), s.Config.MerchantLogin, amountStr, strconv.FormatInt(invID, 10), password1, nil, shp)
	if err != nil {
		return RobokassaInitResponse{}, err
	}

	rawInit, _ := json.Marshal(map[string]any{
		"MerchantLogin":  s.Config.MerchantLogin,
		"OutSum":         amountStr,
		"InvId":          invID,
		"Description":    req.Description,
		"Shp_payment_id": paymentID,
		"IsTest":         s.Config.IsTest,
	})

	_, err = s.DB.ExecContext(ctx, `
		INSERT INTO payments (
			id, type, entity_id, provider, status, amount_cents, amount_value, currency,
			inv_id, user_id, order_id, subscription_id, created_at, updated_at, raw_init_payload
		)
		VALUES ($1, $2, $3, 'robokassa', 'PENDING', 0, $4, $5, $6, $7,
			CASE WHEN $2='order' THEN $3 ELSE NULL END,
			CASE WHEN $2='subscription' THEN $3 ELSE NULL END,
			NOW(), NOW(), $8)
	`, paymentID, req.Type, req.EntityID, amountStr, currency, invID, req.UserID, rawInit)
	if err != nil {
		return RobokassaInitResponse{}, err
	}
	_ = s.logEvent(ctx, paymentID, "init", map[string]any{"inv_id": invID, "type": req.Type})

	params := map[string]string{
		"MerchantLogin":  s.Config.MerchantLogin,
		"OutSum":         amountStr,
		"InvId":          strconv.FormatInt(invID, 10),
		"Description":    req.Description,
		"SignatureValue": signature,
		"Shp_payment_id": paymentID,
		"ResultURL":      s.Config.ResultURL,
		"SuccessURL":     s.Config.SuccessURL,
		"FailURL":        s.Config.FailURL,
	}
	if s.Config.IsTest {
		params["IsTest"] = "1"
	}

	paymentURL := buildPaymentURL(s.Config.PaymentURL, params)
	log.Info().Str("provider", "robokassa").Int64("inv_id", invID).Msg("payment initialized")
	return RobokassaInitResponse{PaymentID: paymentID, InvID: invID, Status: "PENDING", PaymentURL: paymentURL, Params: params}, nil
}

func (s *PaymentService) HandleRobokassaResult(ctx context.Context, cb RobokassaCallback) (string, error) {
	password2 := s.Config.Password2
	if s.Config.IsTest && s.Config.TestPassword2 != "" {
		password2 = s.Config.TestPassword2
	}
	expectedSig, err := robokassa.SignatureForResult(robokassa.HashAlgorithm(s.Config.HashAlgorithm), cb.OutSum, cb.InvID, password2, cb.Shp)
	if err != nil {
		return "", err
	}
	if !robokassa.ConstantTimeEqualSignature(expectedSig, cb.SignatureValue) {
		log.Warn().Str("inv_id", cb.InvID).Msg("robokassa signature verification failed")
		return "", errors.New("invalid signature")
	}
	invID, err := strconv.ParseInt(cb.InvID, 10, 64)
	if err != nil {
		return "", errors.New("invalid inv_id")
	}
	actualAmount, err := robokassa.ParseAmount(cb.OutSum)
	if err != nil {
		return "", err
	}
	payload, _ := json.Marshal(cb.RawPayload)

	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return "", err
	}
	defer func() { _ = tx.Rollback() }()

	var paymentID, status, paymentType, entityID, amountRaw, currency string
	err = tx.QueryRowContext(ctx, `
		SELECT id, status, type, entity_id, amount_value::text, currency
		FROM payments
		WHERE provider='robokassa' AND inv_id=$1
		FOR UPDATE
	`, invID).Scan(&paymentID, &status, &paymentType, &entityID, &amountRaw, &currency)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", errors.New("payment not found")
		}
		return "", err
	}
	if status == "SUCCEEDED" {
		if err := tx.Commit(); err != nil {
			return "", err
		}
		return fmt.Sprintf("OK%d", invID), nil
	}

	expectedAmount, err := robokassa.ParseAmount(amountRaw)
	if err != nil {
		return "", err
	}
	if !robokassa.EqualAmountByCurrency(expectedAmount, actualAmount, currency) {
		return "", errors.New("invalid amount")
	}

	if _, err = tx.ExecContext(ctx, `
		UPDATE payments
		SET status='SUCCEEDED', paid_at=NOW(), updated_at=NOW(), raw_callback_payload=$2
		WHERE id=$1
	`, paymentID, payload); err != nil {
		return "", err
	}

	if err := s.applyBusinessEffectTx(ctx, tx, paymentType, entityID); err != nil {
		return "", err
	}

	if _, err = tx.ExecContext(ctx, `INSERT INTO payment_events (payment_id, event_type, payload_json) VALUES ($1, 'result_success', $2)`, paymentID, payload); err != nil {
		return "", err
	}
	if err := tx.Commit(); err != nil {
		return "", err
	}
	log.Info().Int64("inv_id", invID).Str("payment_id", paymentID).Msg("robokassa result processed")
	return fmt.Sprintf("OK%d", invID), nil
}

func (s *PaymentService) VerifySuccessSignature(outSum, invID, signature string, shp map[string]string) error {
	password1 := s.Config.Password1
	if s.Config.IsTest && s.Config.TestPassword1 != "" {
		password1 = s.Config.TestPassword1
	}
	expected, err := robokassa.SignatureForSuccess(robokassa.HashAlgorithm(s.Config.HashAlgorithm), outSum, invID, password1, shp)
	if err != nil {
		return err
	}
	if !robokassa.ConstantTimeEqualSignature(expected, signature) {
		return errors.New("invalid signature")
	}
	return nil
}

func (s *PaymentService) resolveAmount(ctx context.Context, paymentType, entityID string) (*big.Rat, string, error) {
	var cents int
	var q string
	switch paymentType {
	case "order":
		if err := s.DB.QueryRowContext(ctx, `SELECT total_cents FROM orders WHERE id=$1`, entityID).Scan(&cents); err != nil {
			return nil, "", err
		}
	case "subscription":
		if err := s.DB.QueryRowContext(ctx, `SELECT p.price_cents::text FROM subscriptions s JOIN plans p ON p.id=s.plan_id WHERE s.id=$1`, entityID).Scan(&q); err != nil {
			return nil, "", err
		}
		parsed, _ := strconv.Atoi(q)
		cents = parsed
	}
	amount, err := robokassa.ParseAmount(fmt.Sprintf("%d.%02d", cents/100, cents%100))
	if err != nil {
		return nil, "", err
	}
	return amount, s.Config.DefaultCurrency, nil
}

func (s *PaymentService) applyBusinessEffectTx(ctx context.Context, tx *sql.Tx, paymentType, entityID string) error {
	now := time.Now()
	switch paymentType {
	case "order":
		if _, err := tx.ExecContext(ctx, `UPDATE orders SET status='PAID' WHERE id=$1 AND status <> 'PAID'`, entityID); err != nil {
			return err
		}
		rows, err := tx.QueryContext(ctx, `SELECT product_id, quantity FROM order_items WHERE order_id=$1`, entityID)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			var productID string
			var quantity int
			if err := rows.Scan(&productID, &quantity); err != nil {
				return err
			}
			var stock int
			if err := tx.QueryRowContext(ctx, `SELECT stock FROM products WHERE id=$1 FOR UPDATE`, productID).Scan(&stock); err != nil {
				return err
			}
			if stock < quantity {
				return errors.New("insufficient stock")
			}
			if _, err := tx.ExecContext(ctx, `UPDATE products SET stock=$2 WHERE id=$1`, productID, stock-quantity); err != nil {
				return err
			}
		}
		return rows.Err()
	case "subscription":
		_, err := tx.ExecContext(ctx, `UPDATE subscriptions SET status='ACTIVE', current_period_start=$2, current_period_end=$3 WHERE id=$1`, entityID, now, now.Add(30*24*time.Hour))
		return err
	default:
		return errors.New("unsupported payment type")
	}
}

func (s *PaymentService) nextInvID(ctx context.Context) (int64, error) {
	var invID int64
	err := s.DB.QueryRowContext(ctx, `SELECT nextval('public.robokassa_inv_id_seq'::regclass)`).Scan(&invID)
	return invID, err
}

func (s *PaymentService) logEvent(ctx context.Context, paymentID, eventType string, payload any) error {
	buf, _ := json.Marshal(payload)
	_, err := s.DB.ExecContext(ctx, `INSERT INTO payment_events (payment_id, event_type, payload_json) VALUES ($1, $2, $3)`, paymentID, eventType, buf)
	return err
}

func buildPaymentURL(base string, params map[string]string) string {
	values := url.Values{}
	for k, v := range params {
		values.Set(k, v)
	}
	if strings.Contains(base, "?") {
		return base + "&" + values.Encode()
	}
	return base + "?" + values.Encode()
}
