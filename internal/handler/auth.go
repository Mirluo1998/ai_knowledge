package handler

import (
	"context"
	"errors"
	"knowledge/internal/model"
	"knowledge/internal/service"
	"log/slog"
	"net/http"
)

// AuthService 定义鉴权处理器所需的业务能力，接口由消费方定义，
// 使 handler 可以脱离具体 service 实现进行测试。
type AuthService interface {
	Login(ctx context.Context, user model.User) (*model.AuthedUser, error)
}

// AuthHandler 处理登录与会话校验相关的 HTTP 请求。
type AuthHandler struct {
	svc    AuthService
	logger *slog.Logger
}

// NewAuthHandler 创建鉴权处理器。
func NewAuthHandler(svc AuthService, logger *slog.Logger) *AuthHandler {
	return &AuthHandler{svc: svc, logger: logger}
}

// Login 处理 POST /api/v1/user/login：校验用户名密码，
// 成功后返回 token 及用户信息（会话已写入 Redis）。
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var loginUser model.User
	if err := decodeJSONBody(w, r, &loginUser); err != nil {
		writeBodyDecodeError(w, err)
		return
	}

	authedUser, err := h.svc.Login(r.Context(), loginUser)
	if err != nil {
		var verr *service.ValidationError
		if errors.As(err, &verr) {
			// 凭证错误属于客户端问题，返回 401。
			writeJSON(w, http.StatusUnauthorized, Fail(verr.Message))
			return
		}
		h.logger.Error("login failed", "username", loginUser.Username, "error", err)
		writeJSON(w, http.StatusInternalServerError, Fail("internal server error"))
		return
	}

	writeJSON(w, http.StatusOK, Success(authedUser))
}
