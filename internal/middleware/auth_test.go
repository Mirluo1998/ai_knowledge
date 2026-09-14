package middleware_test

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"knowledge/internal/middleware"
	"knowledge/internal/model"
	"knowledge/internal/service"
)

// fakeVerifier 是 middleware.TokenVerifier 的内存实现。
type fakeVerifier struct {
	user     *model.UserResponse
	err      error
	gotToken string
}

func (f *fakeVerifier) VerifyToken(_ context.Context, token string) (*model.UserResponse, error) {
	f.gotToken = token
	return f.user, f.err
}

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

// serve 用 Auth 包裹 next 并发起请求，返回响应记录器和 next 是否被调用。
func serve(t *testing.T, verifier middleware.TokenVerifier, req *http.Request) (*httptest.ResponseRecorder, bool) {
	t.Helper()
	called := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		user, ok := middleware.UserFromContext(r.Context())
		if !ok || user.Username != "alice" {
			t.Errorf("expected user alice in context, got %+v (ok=%v)", user, ok)
		}
		w.WriteHeader(http.StatusNoContent)
	})

	rec := httptest.NewRecorder()
	middleware.Auth(verifier, testLogger())(next).ServeHTTP(rec, req)
	return rec, called
}

func TestAuthSuccess(t *testing.T) {
	verifier := &fakeVerifier{user: &model.UserResponse{ID: 1, Username: "alice", Email: "a@x.com"}}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/knowledge", nil)
	req.Header.Set("Authorization", "Bearer tok-123")

	rec, called := serve(t, verifier, req)
	if !called {
		t.Fatal("expected next handler to be called")
	}
	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", rec.Code)
	}
	if verifier.gotToken != "tok-123" {
		t.Errorf("token not passed to verifier, got %q", verifier.gotToken)
	}
}

func TestAuthMissingHeader(t *testing.T) {
	verifier := &fakeVerifier{user: &model.UserResponse{Username: "alice"}}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/knowledge", nil)

	rec, called := serve(t, verifier, req)
	if called {
		t.Error("next handler must not be called")
	}
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rec.Code)
	}
	if rec.Header().Get("WWW-Authenticate") != "Bearer" {
		t.Errorf("expected WWW-Authenticate: Bearer, got %q", rec.Header().Get("WWW-Authenticate"))
	}
}

func TestAuthMalformedHeader(t *testing.T) {
	verifier := &fakeVerifier{}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/knowledge", nil)
	req.Header.Set("Authorization", "Basic abc")

	rec, called := serve(t, verifier, req)
	if called {
		t.Error("next handler must not be called")
	}
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rec.Code)
	}
}

func TestAuthInvalidToken(t *testing.T) {
	verifier := &fakeVerifier{err: &service.ValidationError{Message: "invalid or expired token"}}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/knowledge", nil)
	req.Header.Set("Authorization", "Bearer expired")

	rec, called := serve(t, verifier, req)
	if called {
		t.Error("next handler must not be called")
	}
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rec.Code)
	}
}

func TestAuthSessionNotFoundSentinel(t *testing.T) {
	verifier := &fakeVerifier{err: service.ErrSessionNotFound}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/knowledge", nil)
	req.Header.Set("Authorization", "Bearer gone")

	rec, called := serve(t, verifier, req)
	if called {
		t.Error("next handler must not be called")
	}
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rec.Code)
	}
}

func TestAuthVerifierFailure(t *testing.T) {
	verifier := &fakeVerifier{err: errors.New("redis down")}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/knowledge", nil)
	req.Header.Set("Authorization", "Bearer tok")

	rec, called := serve(t, verifier, req)
	if called {
		t.Error("next handler must not be called")
	}
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", rec.Code)
	}
}
