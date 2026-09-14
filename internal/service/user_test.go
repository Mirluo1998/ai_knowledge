package service_test

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"strings"
	"testing"
	"time"

	"golang.org/x/crypto/bcrypt"

	"knowledge/internal/model"
	"knowledge/internal/service"
)

// fakeUserRepo 是 UserRepository 的内存实现。
type fakeUserRepo struct {
	users       map[string]*model.User // key: username
	addErr      error
	queryResult []*model.User
	queryErr    error
}

func (f *fakeUserRepo) AddUser(_ context.Context, user model.User) (bool, error) {
	if f.addErr != nil {
		return false, f.addErr
	}
	f.users[user.Username] = &user
	return true, nil
}

func (f *fakeUserRepo) QueryUser(_ context.Context, _ model.UserQuery) ([]*model.User, error) {
	return f.queryResult, f.queryErr
}

func (f *fakeUserRepo) GetByUsername(_ context.Context, username string) (*model.User, error) {
	if u, ok := f.users[username]; ok {
		return u, nil
	}
	// 与 repository 实现保持一致：用户不存在时包装 sql.ErrNoRows。
	return nil, sql.ErrNoRows
}

// fakeSessionStore 是 SessionStore 的内存实现。
type fakeSessionStore struct {
	data map[string]*model.UserResponse
	ttls map[string]time.Duration
	err  error
}

func newFakeSessionStore() *fakeSessionStore {
	return &fakeSessionStore{
		data: make(map[string]*model.UserResponse),
		ttls: make(map[string]time.Duration),
	}
}

func (f *fakeSessionStore) Set(_ context.Context, token string, user *model.UserResponse, ttl time.Duration) error {
	if f.err != nil {
		return f.err
	}
	f.data[token] = user
	f.ttls[token] = ttl
	return nil
}

func (f *fakeSessionStore) Get(_ context.Context, token string) (*model.UserResponse, error) {
	if u, ok := f.data[token]; ok {
		return u, nil
	}
	return nil, service.ErrSessionNotFound
}

func newUserService(t *testing.T, repo service.UserRepository, sessions service.SessionStore) *service.UserService {
	t.Helper()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	return service.NewUserService(repo, sessions, time.Hour, logger)
}

// seedUser 插入一个密码已做 bcrypt 哈希的用户。
func seedUser(t *testing.T, repo *fakeUserRepo, username, password string) {
	t.Helper()
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.MinCost)
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}
	repo.users[username] = &model.User{
		ID:       1,
		Username: username,
		Password: string(hash),
		Email:    username + "@example.com",
	}
}

func TestLoginUserNotFound(t *testing.T) {
	repo := &fakeUserRepo{users: make(map[string]*model.User)}
	sessions := newFakeSessionStore()
	svc := newUserService(t, repo, sessions)

	_, err := svc.Login(context.Background(), model.User{Username: "nobody", Password: "pw"})

	var verr *service.ValidationError
	if !errors.As(err, &verr) {
		t.Fatalf("expected *ValidationError, got %T (%v)", err, err)
	}
	if len(sessions.data) != 0 {
		t.Errorf("expected no session created, got %d", len(sessions.data))
	}
}

func TestLoginWrongPassword(t *testing.T) {
	repo := &fakeUserRepo{users: make(map[string]*model.User)}
	seedUser(t, repo, "alice", "correct-pw")
	sessions := newFakeSessionStore()
	svc := newUserService(t, repo, sessions)

	_, err := svc.Login(context.Background(), model.User{Username: "alice", Password: "wrong-pw"})

	var verr *service.ValidationError
	if !errors.As(err, &verr) {
		t.Fatalf("expected *ValidationError, got %T (%v)", err, err)
	}
	if verr.Message != "invalid username or password" {
		t.Errorf("unexpected message: %q", verr.Message)
	}
	if len(sessions.data) != 0 {
		t.Errorf("expected no session created, got %d", len(sessions.data))
	}
}

func TestLoginSuccessAndVerifyToken(t *testing.T) {
	repo := &fakeUserRepo{users: make(map[string]*model.User)}
	seedUser(t, repo, "alice", "pw123456")
	sessions := newFakeSessionStore()
	svc := newUserService(t, repo, sessions)

	authed, err := svc.Login(context.Background(), model.User{Username: "alice", Password: "pw123456"})
	if err != nil {
		t.Fatalf("Login returned error: %v", err)
	}
	if authed.Token == "" {
		t.Fatal("expected non-empty token")
	}
	// 会话载荷与登录响应都不应包含密码。
	if authed.UserResponse == nil {
		t.Fatal("expected non-nil user payload")
	}
	if payload, err := json.Marshal(authed); err != nil || strings.Contains(string(payload), "password") {
		t.Errorf("login response must not contain password, got %s", payload)
	}
	if authed.Username != "alice" || authed.Email != "alice@example.com" {
		t.Errorf("unexpected user payload: %+v", authed.UserResponse)
	}
	if sessions.ttls[authed.Token] != time.Hour {
		t.Errorf("expected ttl 1h, got %s", sessions.ttls[authed.Token])
	}

	// 用 token 能取回会话用户。
	got, err := svc.VerifyToken(context.Background(), authed.Token)
	if err != nil {
		t.Fatalf("VerifyToken returned error: %v", err)
	}
	if got.Username != "alice" {
		t.Errorf("unexpected user from session: %+v", got)
	}

	// 伪造/过期 token 返回校验错误。
	_, err = svc.VerifyToken(context.Background(), "not-a-token")
	var verr *service.ValidationError
	if !errors.As(err, &verr) {
		t.Fatalf("expected *ValidationError for bad token, got %T (%v)", err, err)
	}
}

func TestRegisterUserValidation(t *testing.T) {
	cases := []struct {
		name string
		user model.User
	}{
		{"empty username", model.User{Username: "", Password: "longenough", Email: "a@example.com"}},
		{"username too short", model.User{Username: "ab", Password: "longenough", Email: "a@example.com"}},
		{"username bad chars", model.User{Username: "bad name!", Password: "longenough", Email: "a@example.com"}},
		{"password too short", model.User{Username: "alice", Password: "short", Email: "a@example.com"}},
		{"password too long", model.User{Username: "alice", Password: strings.Repeat("a", 73), Email: "a@example.com"}},
		{"empty email", model.User{Username: "alice", Password: "longenough", Email: ""}},
		{"malformed email", model.User{Username: "alice", Password: "longenough", Email: "not-an-email"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			repo := &fakeUserRepo{users: make(map[string]*model.User)}
			svc := newUserService(t, repo, newFakeSessionStore())

			ok, err := svc.RegisterUser(context.Background(), tc.user)
			var verr *service.ValidationError
			if !errors.As(err, &verr) {
				t.Fatalf("expected *ValidationError, got %T (%v)", err, err)
			}
			if ok {
				t.Error("expected ok=false on validation failure")
			}
			if len(repo.users) != 0 {
				t.Error("invalid input must not be written to the repository")
			}
		})
	}
}

func TestRegisterUserHashesPasswordAndDuplicateMapping(t *testing.T) {
	repo := &fakeUserRepo{users: make(map[string]*model.User)}
	svc := newUserService(t, repo, newFakeSessionStore())

	ok, err := svc.RegisterUser(context.Background(), model.User{
		Username: "alice", Password: "plaintext-pw", Email: "alice@example.com",
	})
	if err != nil || !ok {
		t.Fatalf("register failed: %v", err)
	}
	stored := repo.users["alice"]
	if stored.Password == "plaintext-pw" {
		t.Fatal("password must not be stored in plaintext")
	}
	if bcrypt.CompareHashAndPassword([]byte(stored.Password), []byte("plaintext-pw")) != nil {
		t.Error("stored password must be a valid bcrypt hash of the original")
	}

	// 唯一索引冲突应转换为 ValidationError，且不区分用户名还是邮箱冲突。
	repo.addErr = fmt.Errorf("add user: %w", service.ErrUserAlreadyExists)
	_, err = svc.RegisterUser(context.Background(), model.User{
		Username: "bob", Password: "plaintext-pw", Email: "bob@example.com",
	})
	var verr *service.ValidationError
	if !errors.As(err, &verr) {
		t.Fatalf("expected *ValidationError for duplicate, got %T (%v)", err, err)
	}
	if verr.Message != "username or email already registered" {
		t.Errorf("unexpected message: %q", verr.Message)
	}
}

func TestGetUserRequiresFilter(t *testing.T) {
	svc := newUserService(t, &fakeUserRepo{users: make(map[string]*model.User)}, newFakeSessionStore())

	_, err := svc.GetUser(context.Background(), model.UserQuery{})
	var verr *service.ValidationError
	if !errors.As(err, &verr) {
		t.Fatalf("expected *ValidationError for empty query, got %T (%v)", err, err)
	}
}

func TestGetUserStripsPassword(t *testing.T) {
	repo := &fakeUserRepo{
		users: map[string]*model.User{},
		queryResult: []*model.User{
			{ID: 1, Username: "alice", Password: "secret-bcrypt-hash", Email: "alice@example.com"},
		},
	}
	svc := newUserService(t, repo, newFakeSessionStore())

	got, err := svc.GetUser(context.Background(), model.UserQuery{Username: "alice"})
	if err != nil {
		t.Fatalf("GetUser failed: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("expected 1 user, got %d", len(got))
	}
	if payload, err := json.Marshal(got); err != nil || strings.Contains(strings.ToLower(string(payload)), "password") {
		t.Errorf("response must not leak password hash, got %s", payload)
	}
}
