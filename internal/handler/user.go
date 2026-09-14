package handler

import (
	"context"
	"errors"
	"knowledge/internal/model"
	"knowledge/internal/service"
	"log/slog"
	"net/http"
	"strconv"
)

// UserService 定义用户处理器所需的业务能力，接口由消费方定义，
// 使 handler 可以脱离具体 service 实现进行测试（与 AuthService 模式一致）。
type UserService interface {
	RegisterUser(ctx context.Context, user model.User) (bool, error)
	GetUser(ctx context.Context, query model.UserQuery) ([]*model.UserResponse, error)
}

type UserHandler struct {
	logger *slog.Logger
	svc    UserService
}

func NewUserHandler(logger *slog.Logger, svc UserService) *UserHandler {
	return &UserHandler{logger: logger, svc: svc}
}

// GetUser 处理 GET /api/v1/user/get：按 id/username/email 精确检索用户。
// 至少提供一个查询条件；响应只包含非敏感字段。
func (h *UserHandler) GetUser(w http.ResponseWriter, r *http.Request) {
	var query model.UserQuery
	if rawID := r.URL.Query().Get("id"); rawID != "" {
		id, err := strconv.ParseInt(rawID, 10, 64)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, Fail("invalid id"))
			return
		}
		query.ID = id
	}
	query.Username = r.URL.Query().Get("username")
	query.Email = r.URL.Query().Get("email")

	resp, err := h.svc.GetUser(r.Context(), query)
	if err != nil {
		var verr *service.ValidationError
		if errors.As(err, &verr) {
			writeJSON(w, http.StatusBadRequest, Fail(verr.Message))
			return
		}
		h.logger.Error("get user failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, Fail("internal server error"))
		return
	}

	writeJSON(w, http.StatusOK, Success(resp))
}

// Register 处理 POST /api/v1/user/register。
// 输入不合法或用户名/邮箱冲突返回 4xx；数据库等内部错误只记日志，不泄漏细节。
func (h *UserHandler) Register(w http.ResponseWriter, r *http.Request) {
	var registerUser model.User
	if err := decodeJSONBody(w, r, &registerUser); err != nil {
		writeBodyDecodeError(w, err)
		return
	}

	if _, err := h.svc.RegisterUser(r.Context(), registerUser); err != nil {
		var verr *service.ValidationError
		if errors.As(err, &verr) {
			writeJSON(w, http.StatusBadRequest, Fail(verr.Message))
			return
		}
		h.logger.Error("register user failed", "username", registerUser.Username, "error", err)
		writeJSON(w, http.StatusInternalServerError, Fail("internal server error"))
		return
	}

	writeJSON(w, http.StatusCreated, Success("user registered successfully"))
}
