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

type fakeUserStore struct {
	findByPhoneFn func(context.Context, string) (repositories.User, error)
	createdUsers  []repositories.User
}

func (f *fakeUserStore) FindByPhone(ctx context.Context, phone string) (repositories.User, error) {
	if f.findByPhoneFn != nil {
		return f.findByPhoneFn(ctx, phone)
	}
	return repositories.User{}, sql.ErrNoRows
}

func (f *fakeUserStore) Create(_ context.Context, user repositories.User) error {
	f.createdUsers = append(f.createdUsers, user)
	return nil
}

func (f *fakeUserStore) FindByID(context.Context, string) (repositories.User, error) {
	return repositories.User{}, sql.ErrNoRows
}

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

func TestCheckPhone_NewUser(t *testing.T) {
	users := &fakeUserStore{}
	svc := &AuthService{Users: users}

	result, err := svc.CheckPhone(context.Background(), "+79990001122")
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if !result.IsNewUser || !result.RequiresProfile {
		t.Fatalf("expected new user with profile requirement")
	}
}

func TestSendOTP_NewUserRequiresName(t *testing.T) {
	otp := &fakeOTPStore{latestErr: sql.ErrNoRows}
	users := &fakeUserStore{}
	svc := &AuthService{Users: users, OTP: otp, OTPTTL: time.Minute, OTPDeliveryMode: OTPDeliveryModeDev, OTPRateLimit: time.Second}

	_, err := svc.SendOTP(context.Background(), "+79990001122", "", "")
	if !errors.Is(err, ErrNameRequired) {
		t.Fatalf("expected ErrNameRequired got %v", err)
	}
}

func TestSendOTP_DevModeDoesNotCallSMS(t *testing.T) {
	otp := &fakeOTPStore{latestErr: sql.ErrNoRows}
	sms := &fakeSMS{}
	users := &fakeUserStore{}
	svc := &AuthService{Users: users, OTP: otp, OTPTTL: time.Minute, OTPDeliveryMode: OTPDeliveryModeDev, SMS: sms, OTPMessagePrefix: "Код подтверждения:"}

	result, err := svc.SendOTP(context.Background(), "+79990001122", "Иван Иванов", "")
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if len(users.createdUsers) != 1 {
		t.Fatalf("expected user to be created")
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
	users := &fakeUserStore{findByPhoneFn: func(context.Context, string) (repositories.User, error) {
		return repositories.User{ID: "u1", Phone: "79990001122", Role: "user"}, nil
	}}
	svc := &AuthService{Users: users, OTP: otp, OTPTTL: time.Minute, OTPDeliveryMode: OTPDeliveryModeMobizon, SMS: sms, OTPMessagePrefix: "Код подтверждения:"}

	result, err := svc.SendOTP(context.Background(), "+79990001122", "", "")
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
	users := &fakeUserStore{findByPhoneFn: func(context.Context, string) (repositories.User, error) {
		return repositories.User{ID: "u1", Phone: "79990001122", Role: "user"}, nil
	}}
	svc := &AuthService{Users: users, OTP: otp, OTPTTL: time.Minute, OTPDeliveryMode: OTPDeliveryModeMobizon, SMS: sms, OTPMessagePrefix: "Код подтверждения:"}

	_, err := svc.SendOTP(context.Background(), "+79990001122", "", "")
	if !errors.Is(err, ErrSMSRateLimited) {
		t.Fatalf("expected ErrSMSRateLimited got %v", err)
	}
}
