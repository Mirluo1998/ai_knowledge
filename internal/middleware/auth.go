package middleware

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"knowledge/internal/model"
	"knowledge/internal/service"
)

// TokenVerifier 定义鉴权中间件所需的能力，接口由消费方定义，
// *service.UserService 天然满足，测试时可替换为内存 fake。
type TokenVerifier interface {
	VerifyToken(ctx context.Context, token string) (*model.UserResponse, error)
}

// ctxKey 是中间件私有 context 键类型，避免与其他包的键冲突。
type ctxKey int

const userCtxKey ctxKey = iota

// authError 与 handler.Result 的失败响应保持同一结构。
type authError struct {
	Success bool   `json:"success"`
	Data    any    `json:"data"`
	Message string `json:"message"`
}

// Auth 校验 Authorization 头中的 Bearer token：
// 缺失/格式错误、会话无效或过期返回 401；校验服务本身故障返回 500；
// 校验通过则把用户信息写入 request context 并放行。
func Auth(verifier TokenVerifier, logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token, err := bearerToken(r)
			if err != nil {
				writeAuthError(w, http.StatusUnauthorized, err.Error())
				return
			}

			user, err := verifier.VerifyToken(r.Context(), token)
			if err != nil {
				var verr *service.ValidationError
				if errors.As(err, &verr) || errors.Is(err, service.ErrSessionNotFound) {
					writeAuthError(w, http.StatusUnauthorized, "invalid or expired token")
					return
				}
				logger.Error("verify token failed", "error", err)
				writeAuthError(w, http.StatusInternalServerError, "internal server error")
				return
			}

			ctx := context.WithValue(r.Context(), userCtxKey, user)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// UserFromContext 取出鉴权中间件注入的当前登录用户。
func UserFromContext(ctx context.Context) (*model.UserResponse, bool) {
	user, ok := ctx.Value(userCtxKey).(*model.UserResponse)
	return user, ok
}

// bearerToken 从 Authorization 头提取 Bearer token。
func bearerToken(r *http.Request) (string, error) {
	auth := r.Header.Get("Authorization")
	if auth == "" {
		return "", errors.New("missing Authorization header")
	}
	scheme, token, ok := strings.Cut(auth, " ")
	if !ok || !strings.EqualFold(scheme, "Bearer") || token == "" {
		return "", errors.New("invalid Authorization header, expected: Bearer <token>")
	}
	return token, nil
}

// writeAuthError 写出统一结构的 JSON 错误响应；401 时附带 WWW-Authenticate 质询。
func writeAuthError(w http.ResponseWriter, status int, message string) {
	if status == http.StatusUnauthorized {
		w.Header().Set("WWW-Authenticate", "Bearer")
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(authError{Success: false, Data: nil, Message: message})
}
