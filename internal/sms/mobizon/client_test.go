package mobizon

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestBuildFormEncodesBody(t *testing.T) {
	form := buildForm("79990001122", "Код подтверждения: 123456", "NESTA", 10)
	encoded := form.Encode()
	if !strings.Contains(encoded, "recipient=79990001122") {
		t.Fatalf("recipient missing: %s", encoded)
	}
	if !strings.Contains(encoded, "from=NESTA") {
		t.Fatalf("sender missing: %s", encoded)
	}
	if !strings.Contains(encoded, "params%5Bvalidity%5D=10") {
		t.Fatalf("validity missing: %s", encoded)
	}
	if !strings.Contains(encoded, "text=%D0%9A%D0%BE%D0%B4+") {
		t.Fatalf("text must be url-encoded: %s", encoded)
	}
}

func TestParseSendSMSResponse(t *testing.T) {
	_, err := parseSendSMSResponse([]byte(`{"code":1,"message":"validation"}`))
	if err == nil {
		t.Fatal("expected error")
	}

	result, err := parseSendSMSResponse([]byte(`{"code":0,"data":{"messageId":"123","campaignId":456,"status":"accepted"}}`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.MessageID != "123" || result.CampaignID != "456" || result.Status != "accepted" {
		t.Fatalf("unexpected result: %+v", result)
	}
}

func TestSendSMS_IntegrationMock(t *testing.T) {
	tests := []struct {
		name       string
		response   string
		wantErr    bool
		wantAPIErr int
	}{
		{name: "ok", response: `{"code":0,"data":{"messageId":"1","campaignId":"2","status":"sent"}}`},
		{name: "rate limit", response: `{"code":30,"message":"too many requests"}`, wantErr: true, wantAPIErr: 30},
		{name: "validation", response: `{"code":1,"message":"invalid recipient"}`, wantErr: true, wantAPIErr: 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodPost {
					t.Fatalf("expected POST got %s", r.Method)
				}
				if r.URL.Path != "/service/Message/SendSmsMessage" {
					t.Fatalf("unexpected path %s", r.URL.Path)
				}
				body, _ := io.ReadAll(r.Body)
				if !strings.Contains(string(body), "recipient=79990001122") {
					t.Fatalf("body missing recipient: %s", string(body))
				}
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(tt.response))
			}))
			defer srv.Close()

			client, err := NewClient(srv.Client(), srv.URL+"/service/", "secret")
			if err != nil {
				t.Fatalf("new client error: %v", err)
			}
			_, err = client.SendSMS(context.Background(), "79990001122", "Код подтверждения: 111111", "", 0)
			if tt.wantErr && err == nil {
				t.Fatalf("expected error")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if tt.wantAPIErr != 0 {
				apiErr, ok := err.(*APIError)
				if !ok || apiErr.Code != tt.wantAPIErr {
					t.Fatalf("expected APIError code %d got %v", tt.wantAPIErr, err)
				}
			}
		})
	}
}
