// Package handler 实现知识服务的 HTTP 接入层。
package handler

import (
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"strconv"

	"knowledge/internal/model"
	"knowledge/internal/service"
)

// KnowledgeHandler 处理知识条目相关的 HTTP 请求。
type KnowledgeHandler struct {
	svc    *service.KnowledgeService
	logger *slog.Logger
}

// NewKnowledgeHandler 创建知识条目处理器。
func NewKnowledgeHandler(svc *service.KnowledgeService, logger *slog.Logger) *KnowledgeHandler {
	return &KnowledgeHandler{svc: svc, logger: logger}
}

// ListKnowledge 处理 GET /api/v1/knowledge，支持 ?type=&title= 过滤。
func (h *KnowledgeHandler) ListKnowledge(w http.ResponseWriter, r *http.Request) {
	typeStr := r.URL.Query().Get("type")
	var typeValue int32
	if typeStr != "" {
		value, err := strconv.ParseInt(typeStr, 10, 32)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid type")
			return
		}
		typeValue = int32(value)
	}

	query := model.KnowledgeQuery{
		Type:  typeValue,
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

func (h *KnowledgeHandler) CreateKnowledge(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return
	}
	var knowledge model.Knowledge

	if err := json.Unmarshal(body, &knowledge); err != nil {
		h.logger.Error("read request body failed", "error", err)
		writeError(w, http.StatusBadRequest, "read request body failed")
		return
	}

	_, err = h.svc.CreateKnowledge(r.Context(), knowledge)
	if err != nil {
		h.logger.Error("create knowledge failed", "error", err)
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}
	writeJSON(w, http.StatusOK, knowledge)
}
