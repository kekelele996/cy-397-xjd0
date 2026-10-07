package service_test

import (
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/contractapi/contractapi/internal/service"
	"github.com/contractapi/contractapi/pkg/jwtutil"
)

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func TestAuthServiceRegisterAndLogin(t *testing.T) {
	tests := []struct {
		name     string
		username string
		password string
		wantErr  bool
	}{
		{name: "success", username: "alice", password: "secret123", wantErr: false},
		{name: "duplicate", username: "alice", password: "secret456", wantErr: true},
		{name: "another user", username: "bob", password: "secret789", wantErr: false},
	}

	repo := newMockUserRepo()
	jwt := jwtutil.NewManager("test-secret", time.Hour)
	svc := service.NewAuthService(repo, jwt, testLogger())

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			user, err := svc.Register(tt.username, tt.password)
			if (err != nil) != tt.wantErr {
				t.Fatalf("Register() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err != nil {
				return
			}
			if user.ID == 0 {
				t.Fatal("Register() user id should not be zero")
			}
			token, _, gotUser, err := svc.Login(tt.username, tt.password)
			if err != nil {
				t.Fatalf("Login() unexpected error: %v", err)
			}
			if token == "" || gotUser.ID != user.ID {
				t.Fatalf("Login() returned unexpected token/user: %q, %+v", token, gotUser)
			}
		})
	}
}

func TestAuthServiceLoginWrongPassword(t *testing.T) {
	repo := newMockUserRepo()
	jwt := jwtutil.NewManager("test-secret", time.Hour)
	svc := service.NewAuthService(repo, jwt, testLogger())

	if _, err := svc.Register("carol", "secret123"); err != nil {
		t.Fatalf("Register() unexpected error: %v", err)
	}
	if _, _, _, err := svc.Login("carol", "wrong-password"); err == nil {
		t.Fatal("Login() expected error for wrong password, got nil")
	}
	if _, _, _, err := svc.Login("missing", "secret123"); err == nil {
		t.Fatal("Login() expected error for missing user, got nil")
	}
}
