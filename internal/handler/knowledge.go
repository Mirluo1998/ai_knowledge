// Package handler 实现知识服务的 HTTP 接入层。
package handler

import (
	"context"
	"errors"
	"log/slog"
	"net/http"

	"knowledge/internal/model"
	"knowledge/internal/service"
)

// KnowledgeService 定义 handler 层所需的业务能力。
type KnowledgeService interface {
	ListKnowledge(ctx context.Context, query model.KnowledgeQuery) ([]model.Knowledge, error)
}

// KnowledgeHandler 处理知识条目相关的 HTTP 请求。
type KnowledgeHandler struct {
	svc    KnowledgeService
	logger *slog.Logger
}

// NewKnowledgeHandler 创建知识条目处理器。
func NewKnowledgeHandler(svc KnowledgeService, logger *slog.Logger) *KnowledgeHandler {
	return &KnowledgeHandler{svc: svc, logger: logger}
}

// ListKnowledge 处理 GET /api/v1/knowledge，支持 ?type=&title= 过滤。
func (h *KnowledgeHandler) ListKnowledge(w http.ResponseWriter, r *http.Request) {
	query := model.KnowledgeQuery{
		Type:  r.URL.Query().Get("type"),
		Title: r.URL.Query().Get("title"),
	}

	items, err := h.svc.ListKnowledge(r.Context(), query)
	if err != nil {
		var verr *service.ValidationError
		if errors.As(err, &verr) {
			writeError(w, http.StatusBadRequest, verr.Message)
			return
		}
		// 内部错误只记录日志，不向客户端泄漏细节。
		h.logger.Error("list knowledge failed", "error", err)
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	writeJSON(w, http.StatusOK, items)
}
