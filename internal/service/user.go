package service

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"knowledge/internal/model"
	"log/slog"
	"regexp"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

// ErrSessionNotFound 表示 token 对应的会话不存在或已过期。
// 由消费方定义，会话存储实现（cache 包）返回该哨兵错误，
// service 据此转换为面向客户端的 ValidationError。
var ErrSessionNotFound = errors.New("session not found or expired")

// ErrUserAlreadyExists 表示用户名或邮箱已被注册（底层唯一索引冲突）。
// repository 实现返回该哨兵错误，service 转换为面向客户端的 ValidationError。
var ErrUserAlreadyExists = errors.New("user already exists")

type UserRepository interface {
	AddUser(ctx context.Context, user model.User) (bool, error)
	QueryUser(ctx context.Context, query model.UserQuery) ([]*model.User, error)
	GetByUsername(ctx context.Context, username string) (*model.User, error)
}

// SessionStore 是登录会话存储，由消费方定义，cache 包提供 Redis 实现。
type SessionStore interface {
	Set(ctx context.Context, token string, user *model.UserResponse, ttl time.Duration) error
	Get(ctx context.Context, token string) (*model.UserResponse, error)
}

type UserService struct {
	repo       UserRepository
	sessions   SessionStore
	sessionTTL time.Duration
	logger     *slog.Logger
}

func NewUserService(repo UserRepository, sessions SessionStore, sessionTTL time.Duration, logger *slog.Logger) *UserService {
	return &UserService{repo: repo, sessions: sessions, sessionTTL: sessionTTL, logger: logger}
}

// 注册入参的基本格式约束。
var (
	usernameRegexp = regexp.MustCompile(`^[a-zA-Z0-9_]{3,32}$`)
	emailRegexp    = regexp.MustCompile(`^[^@\s]+@[^@\s]+\.[^@\s]+$`)
)

const (
	minPasswordBytes = 8
	// bcrypt 只使用前 72 字节，超长密码没有实际强度意义，直接拒绝。
	maxPasswordBytes = 72
	maxEmailLength   = 254
)

// validateRegistration 校验注册入参，不合法时返回 *ValidationError。
func validateRegistration(user model.User) error {
	if !usernameRegexp.MatchString(user.Username) {
		return &ValidationError{Message: "username must be 3-32 characters and contain only letters, digits or underscore"}
	}
	if len(user.Password) < minPasswordBytes || len(user.Password) > maxPasswordBytes {
		return &ValidationError{Message: "password must be 8-72 characters"}
	}
	if len(user.Email) > maxEmailLength || !emailRegexp.MatchString(user.Email) {
		return &ValidationError{Message: "invalid email address"}
	}
	return nil
}

func (s *UserService) RegisterUser(ctx context.Context, user model.User) (bool, error) {
	if err := validateRegistration(user); err != nil {
		return false, err
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.DefaultCost)
	if err != nil {
		return false, fmt.Errorf("hash password: %w", err)
	}
	user.Password = string(hash)

	ok, err := s.repo.AddUser(ctx, user)
	if err != nil {
		if errors.Is(err, ErrUserAlreadyExists) {
			// 不区分是用户名还是邮箱冲突，避免被用来枚举注册情况。
			return false, &ValidationError{Message: "username or email already registered"}
		}
		return false, fmt.Errorf("add user: %w", err)
	}
	return ok, nil
}

// GetUser 按条件检索用户。至少提供一个过滤条件，防止无条件拉取全表；
// 返回的 UserResponse 不含密码哈希等敏感字段。
func (s *UserService) GetUser(ctx context.Context, query model.UserQuery) ([]*model.UserResponse, error) {
	if query.ID == 0 && query.Username == "" && query.Email == "" {
		return nil, &ValidationError{Message: "at least one filter (id, username or email) is required"}
	}

	users, err := s.repo.QueryUser(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("query users: %w", err)
	}

	resp := make([]*model.UserResponse, 0, len(users))
	for _, u := range users {
		// 显式逐字段映射，即使 repository 误查出密码也不会泄漏到响应中。
		resp = append(resp, &model.UserResponse{
			ID:       u.ID,
			Username: u.Username,
			Email:    u.Email,
		})
	}
	return resp, nil
}

// invalidCredentials 是用户名或密码错误时统一返回的校验错误，
// 不区分“用户不存在”和“密码错误”，避免泄露用户是否已注册。
var invalidCredentials = &ValidationError{Message: "invalid username or password"}

// dummyPasswordHash 是一个一次性生成的 bcrypt 哈希。
// 用户不存在时仍拿它与提交的密码做一次比较，让“用户不存在”和“密码错误”
// 两条路径耗时接近，消除可用于枚举用户名的时序侧信道。
var dummyPasswordHash = mustGenerateDummyHash()

func mustGenerateDummyHash() []byte {
	h, err := bcrypt.GenerateFromPassword([]byte("dummy-password-for-timing"), bcrypt.DefaultCost)
	if err != nil {
		panic(err)
	}
	return h
}

func (s *UserService) Login(ctx context.Context, loginUser model.User) (*model.AuthedUser, error) {
	user, err := s.repo.GetByUsername(ctx, loginUser.Username)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			// 即使没有对应用户也执行一次 bcrypt 比较，保持时序一致。
			_ = bcrypt.CompareHashAndPassword(dummyPasswordHash, []byte(loginUser.Password))
			return nil, invalidCredentials
		}
		return nil, fmt.Errorf("get user by username: %w", err)
	}

	// 数据库中存的是 bcrypt 哈希，与客户端提交的明文密码做常量时间比较。
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(loginUser.Password)); err != nil {
		return nil, invalidCredentials
	}

	// 会话中只放非敏感信息，绝不写入密码哈希。
	resp := &model.UserResponse{
		ID:       user.ID,
		Username: user.Username,
		Email:    user.Email,
	}
	token := uuid.NewString()
	if err := s.sessions.Set(ctx, token, resp, s.sessionTTL); err != nil {
		return nil, fmt.Errorf("create session: %w", err)
	}

	return &model.AuthedUser{UserResponse: resp, Token: token}, nil
}

// VerifyToken 校验登录 token 的有效性，返回对应的登录用户信息。
// 供后续鉴权中间件使用。
func (s *UserService) VerifyToken(ctx context.Context, token string) (*model.UserResponse, error) {
	user, err := s.sessions.Get(ctx, token)
	if err != nil {
		if errors.Is(err, ErrSessionNotFound) {
			return nil, &ValidationError{Message: "invalid or expired token"}
		}
		return nil, fmt.Errorf("get session: %w", err)
	}
	return user, nil
}
