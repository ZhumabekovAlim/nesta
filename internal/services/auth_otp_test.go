package services

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"nesta/internal/repositories"
	"nesta/internal/sms/mobizon"
)

type fakeOTPStore struct {
	latest       repositories.OTPCode
	latestErr    error
	createdCodes []repositories.OTPCode
}

func (f *fakeOTPStore) LatestByPhone(context.Context, string) (repositories.OTPCode, error) {
	return f.latest, f.latestErr
}
func (f *fakeOTPStore) Create(_ context.Context, code repositories.OTPCode) error {
	f.createdCodes = append(f.createdCodes, code)
	return nil
}
func (f *fakeOTPStore) IncrementAttempts(context.Context, string, int) error { return nil }
func (f *fakeOTPStore) Block(context.Context, string, time.Time) error       { return nil }

type fakeSMS struct {
	calls int
	err   error
}

func (f *fakeSMS) SendSMS(context.Context, string, string, string, int) (mobizon.SendSMSResult, error) {
	f.calls++
	if f.err != nil {
		return mobizon.SendSMSResult{}, f.err
	}
	return mobizon.SendSMSResult{MessageID: "1"}, nil
}

func TestNormalizePhone(t *testing.T) {
	tests := []struct {
		name    string
		phone   string
		want    string
		wantErr error
	}{
		{name: "plus format", phone: "+7 (999) 000-11-22", want: "79990001122"},
		{name: "empty", phone: "", wantErr: ErrPhoneRequired},
		{name: "invalid short", phone: "1234", wantErr: ErrPhoneInvalid},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NormalizePhone(tt.phone)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("expected err %v got %v", tt.wantErr, err)
			}
			if got != tt.want {
				t.Fatalf("expected %s got %s", tt.want, got)
			}
		})
	}
}

func TestSendOTP_DevModeDoesNotCallSMS(t *testing.T) {
	otp := &fakeOTPStore{latestErr: sql.ErrNoRows}
	sms := &fakeSMS{}
	svc := &AuthService{OTP: otp, OTPTTL: time.Minute, OTPDeliveryMode: OTPDeliveryModeDev, SMS: sms, OTPMessagePrefix: "Код подтверждения:"}

	result, err := svc.SendOTP(context.Background(), "+79990001122")
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if sms.calls != 0 {
		t.Fatalf("expected sms not called")
	}
	if result.Code == nil {
		t.Fatalf("expected dev code in dev mode")
	}
}

func TestSendOTP_MobizonModeHidesDevCode(t *testing.T) {
	otp := &fakeOTPStore{latestErr: sql.ErrNoRows}
	sms := &fakeSMS{}
	svc := &AuthService{OTP: otp, OTPTTL: time.Minute, OTPDeliveryMode: OTPDeliveryModeMobizon, SMS: sms, OTPMessagePrefix: "Код подтверждения:"}

	result, err := svc.SendOTP(context.Background(), "+79990001122")
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if sms.calls != 1 {
		t.Fatalf("expected sms called once")
	}
	if result.Code != nil {
		t.Fatalf("expected dev_code to be hidden")
	}
}

func TestSendOTP_MobizonRateLimit(t *testing.T) {
	otp := &fakeOTPStore{latestErr: sql.ErrNoRows}
	sms := &fakeSMS{err: &mobizon.APIError{Code: 30, Message: "limit"}}
	svc := &AuthService{OTP: otp, OTPTTL: time.Minute, OTPDeliveryMode: OTPDeliveryModeMobizon, SMS: sms, OTPMessagePrefix: "Код подтверждения:"}

	_, err := svc.SendOTP(context.Background(), "+79990001122")
	if !errors.Is(err, ErrSMSRateLimited) {
		t.Fatalf("expected ErrSMSRateLimited got %v", err)
	}
}
