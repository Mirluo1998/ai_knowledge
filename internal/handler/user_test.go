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

// fakeUserService 是 handler.UserService 的内存实现。
type fakeUserService struct {
	getResult   []*model.UserResponse
	getErr      error
	registerErr error
	addOK       bool

	gotQuery model.UserQuery
	gotUser  model.User
}

func (f *fakeUserService) GetUser(_ context.Context, query model.UserQuery) ([]*model.UserResponse, error) {
	f.gotQuery = query
	return f.getResult, f.getErr
}

func (f *fakeUserService) RegisterUser(_ context.Context, user model.User) (bool, error) {
	f.gotUser = user
	return f.addOK, f.registerErr
}

func newUserHandler(svc handler.UserService) *handler.UserHandler {
	return handler.NewUserHandler(slog.New(slog.NewTextHandler(io.Discard, nil)), svc)
}

func TestGetUserInvalidID(t *testing.T) {
	h := newUserHandler(&fakeUserService{})
	req := httptest.NewRequest(http.MethodGet, "/api/v1/user/get?id=abc", nil)
	rec := httptest.NewRecorder()

	h.GetUser(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for non-numeric id, got %d", rec.Code)
	}
}

func TestGetUserValidationError(t *testing.T) {
	svc := &fakeUserService{getErr: &service.ValidationError{Message: "at least one filter is required"}}
	h := newUserHandler(svc)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/user/get", nil)
	rec := httptest.NewRecorder()

	h.GetUser(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "at least one filter") {
		t.Errorf("unexpected body: %s", rec.Body.String())
	}
}

func TestGetUserSuccessDoesNotLeakPassword(t *testing.T) {
	svc := &fakeUserService{getResult: []*model.UserResponse{
		{ID: 1, Username: "alice", Email: "alice@example.com"},
	}}
	h := newUserHandler(svc)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/user/get?username=alice", nil)
	rec := httptest.NewRecorder()

	h.GetUser(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	if ct := rec.Header().Get("Content-Type"); !strings.Contains(ct, "application/json") {
		t.Errorf("expected JSON content type, got %q", ct)
	}
	if body := strings.ToLower(rec.Body.String()); strings.Contains(body, "password") {
		t.Errorf("response must not contain password fields, got %s", rec.Body.String())
	}
	if svc.gotQuery.Username != "alice" {
		t.Errorf("unexpected query passed to service: %+v", svc.gotQuery)
	}
}

func TestGetUserInternalErrorDoesNotLeak(t *testing.T) {
	svc := &fakeUserService{getErr: errors.New("db connection refused")}
	h := newUserHandler(svc)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/user/get?username=alice", nil)
	rec := httptest.NewRecorder()

	h.GetUser(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", rec.Code)
	}
	if strings.Contains(rec.Body.String(), "db connection refused") {
		t.Error("internal error detail must not leak to client")
	}
}

func TestRegisterSuccess(t *testing.T) {
	svc := &fakeUserService{addOK: true}
	h := newUserHandler(svc)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/user/register",
		strings.NewReader(`{"username":"alice","password":"plaintext-pw","email":"a@example.com"}`))
	rec := httptest.NewRecorder()

	h.Register(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rec.Code, rec.Body.String())
	}
	if svc.gotUser.Username != "alice" {
		t.Errorf("unexpected user passed to service: %+v", svc.gotUser)
	}
}

func TestRegisterValidationError(t *testing.T) {
	svc := &fakeUserService{registerErr: &service.ValidationError{Message: "password must be 8-72 characters"}}
	h := newUserHandler(svc)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/user/register",
		strings.NewReader(`{"username":"alice","password":"short","email":"a@example.com"}`))
	rec := httptest.NewRecorder()

	h.Register(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "8-72") {
		t.Errorf("unexpected body: %s", rec.Body.String())
	}
}

func TestRegisterDuplicate(t *testing.T) {
	svc := &fakeUserService{registerErr: &service.ValidationError{Message: "username or email already registered"}}
	h := newUserHandler(svc)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/user/register",
		strings.NewReader(`{"username":"alice","password":"plaintext-pw","email":"a@example.com"}`))
	rec := httptest.NewRecorder()

	h.Register(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for duplicate user, got %d", rec.Code)
	}
}

func TestRegisterInternalErrorDoesNotLeak(t *testing.T) {
	svc := &fakeUserService{registerErr: errors.New("Error 1062 Duplicate entry raw leak")}
	h := newUserHandler(svc)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/user/register",
		strings.NewReader(`{"username":"alice","password":"plaintext-pw","email":"a@example.com"}`))
	rec := httptest.NewRecorder()

	h.Register(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", rec.Code)
	}
	if strings.Contains(rec.Body.String(), "Duplicate entry") {
		t.Error("internal error detail must not leak to client")
	}
}

func TestRegisterBodyTooLarge(t *testing.T) {
	h := newUserHandler(&fakeUserService{})
	// 1 MiB 限制：构造超出限制的 JSON。
	large := strings.NewReader(`{"username":"alice","password":"` + strings.Repeat("a", 2<<20) + `"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/user/register", large)
	rec := httptest.NewRecorder()

	h.Register(rec, req)

	if rec.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("expected 413, got %d", rec.Code)
	}
}
