package handler_test

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"knowledge/internal/handler"
	"knowledge/internal/model"
	"knowledge/internal/service"
)

// fakeAuthService 是 handler.AuthService 的内存实现。
type fakeAuthService struct {
	authed   *model.AuthedUser
	loginErr error
}

func (f *fakeAuthService) Login(_ context.Context, _ model.User) (*model.AuthedUser, error) {
	return f.authed, f.loginErr
}

func newAuthHandler(svc handler.AuthService) *handler.AuthHandler {
	return handler.NewAuthHandler(svc, slog.New(slog.NewTextHandler(io.Discard, nil)))
}

func TestLoginSuccess(t *testing.T) {
	svc := &fakeAuthService{authed: &model.AuthedUser{
		UserResponse: &model.UserResponse{ID: 1, Username: "alice", Email: "a@x.com"},
		Token:        "tok-123",
	}}
	h := newAuthHandler(svc)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/user/login",
		strings.NewReader(`{"username":"alice","password":"pw"}`))
	rec := httptest.NewRecorder()
	h.Login(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	if body := rec.Body.String(); !strings.Contains(body, "tok-123") || strings.Contains(body, "password") {
		t.Errorf("unexpected body: %s", body)
	}
}

func TestLoginInvalidBody(t *testing.T) {
	h := newAuthHandler(&fakeAuthService{})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/user/login", strings.NewReader("{bad json"))
	rec := httptest.NewRecorder()

	h.Login(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestLoginInvalidCredentials(t *testing.T) {
	svc := &fakeAuthService{loginErr: &service.ValidationError{Message: "invalid username or password"}}
	h := newAuthHandler(svc)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/user/login",
		strings.NewReader(`{"username":"alice","password":"wrong"}`))
	rec := httptest.NewRecorder()

	h.Login(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rec.Code)
	}
}

func TestLoginInternalError(t *testing.T) {
	svc := &fakeAuthService{loginErr: errors.New("redis down")}
	h := newAuthHandler(svc)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/user/login",
		strings.NewReader(`{"username":"alice","password":"pw"}`))
	rec := httptest.NewRecorder()

	h.Login(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", rec.Code)
	}
	if strings.Contains(rec.Body.String(), "redis down") {
		t.Error("internal error detail must not leak to client")
	}
}
