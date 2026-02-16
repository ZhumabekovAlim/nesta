package mobizon

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/rs/zerolog/log"
)

const (
	DefaultBaseURL = "https://api.mobizon.kz/service/"
)

type Client struct {
	httpClient *http.Client
	baseURL    string
	apiKey     string
}

type SendSMSResult struct {
	MessageID  string
	CampaignID string
	Status     string
}

type APIError struct {
	Code    int
	Message string
}

func (e *APIError) Error() string {
	if e.Message == "" {
		return fmt.Sprintf("mobizon api error: code=%d", e.Code)
	}
	return fmt.Sprintf("mobizon api error: code=%d message=%s", e.Code, e.Message)
}

func NewClient(httpClient *http.Client, baseURL, apiKey string) (*Client, error) {
	if strings.TrimSpace(apiKey) == "" {
		return nil, errors.New("mobizon api key required")
	}
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 10 * time.Second}
	}
	if strings.TrimSpace(baseURL) == "" {
		baseURL = DefaultBaseURL
	}
	if !strings.HasSuffix(baseURL, "/") {
		baseURL += "/"
	}

	return &Client{httpClient: httpClient, baseURL: baseURL, apiKey: apiKey}, nil
}

func (c *Client) SendSMS(ctx context.Context, recipient, text, sender string, validity int) (SendSMSResult, error) {
	endpoint, err := url.Parse(c.baseURL + "Message/SendSmsMessage")
	if err != nil {
		return SendSMSResult{}, err
	}
	q := endpoint.Query()
	q.Set("output", "json")
	q.Set("api", "v1")
	q.Set("apiKey", c.apiKey)
	endpoint.RawQuery = q.Encode()

	form := buildForm(recipient, text, sender, validity)
	encodedForm := form.Encode()

	log.Info().
		Str("provider", "mobizon").
		Str("method", http.MethodPost).
		Str("endpoint", endpoint.Redacted()).
		Str("content_type", "application/x-www-form-urlencoded").
		Str("transport", "https").
		Str("recipient", recipient).
		Str("sender", sender).
		Int("validity_min", validity).
		Str("text", text).
		Str("encoded_body", encodedForm).
		Str("api_key_masked", maskAPIKey(c.apiKey)).
		Msg("mobizon request prepared")

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint.String(), strings.NewReader(encodedForm))
	if err != nil {
		return SendSMSResult{}, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	log.Info().
		Str("provider", "mobizon").
		Str("endpoint", endpoint.Redacted()).
		Msg("mobizon request sent")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return SendSMSResult{}, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return SendSMSResult{}, err
	}

	log.Info().
		Str("provider", "mobizon").
		Int("http_status", resp.StatusCode).
		Str("response_body", string(body)).
		Msg("mobizon response received")

	return parseSendSMSResponse(body)
}

func buildForm(recipient, text, sender string, validity int) url.Values {
	form := url.Values{}
	form.Set("recipient", recipient)
	form.Set("text", text)
	if strings.TrimSpace(sender) != "" {
		form.Set("from", sender)
	}
	if validity > 0 {
		form.Set("params[validity]", strconv.Itoa(validity))
	}
	return form
}

type apiEnvelope struct {
	Code    int             `json:"code"`
	Data    json.RawMessage `json:"data"`
	Message string          `json:"message"`
}

type sendSMSData struct {
	MessageID  any `json:"messageId"`
	CampaignID any `json:"campaignId"`
	Status     any `json:"status"`
}

func parseSendSMSResponse(body []byte) (SendSMSResult, error) {
	var envelope apiEnvelope
	if err := json.Unmarshal(body, &envelope); err != nil {
		return SendSMSResult{}, err
	}
	if envelope.Code != 0 {
		return SendSMSResult{}, &APIError{Code: envelope.Code, Message: envelope.Message}
	}
	var data sendSMSData
	if len(envelope.Data) > 0 && string(envelope.Data) != "null" {
		if err := json.Unmarshal(envelope.Data, &data); err != nil {
			return SendSMSResult{}, err
		}
	}
	return SendSMSResult{
		MessageID:  anyToString(data.MessageID),
		CampaignID: anyToString(data.CampaignID),
		Status:     anyToString(data.Status),
	}, nil
}

func anyToString(v any) string {
	switch value := v.(type) {
	case nil:
		return ""
	case string:
		return value
	case float64:
		return strconv.FormatInt(int64(value), 10)
	default:
		return fmt.Sprint(value)
	}
}

func maskAPIKey(key string) string {
	trimmed := strings.TrimSpace(key)
	if trimmed == "" {
		return ""
	}
	if len(trimmed) <= 4 {
		return "****"
	}
	return trimmed[:2] + strings.Repeat("*", len(trimmed)-4) + trimmed[len(trimmed)-2:]
}
