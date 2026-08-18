package handler

import (
	"encoding/json"
	"net/http"
)

// ErrorResponse 是统一的错误响应结构。
// 对外只暴露可读信息，内部错误细节仅写入日志。
type ErrorResponse struct {
	Error string `json:"error"`
}

// writeJSON 将 v 序列化为 JSON 并以指定状态码写出。
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		// 响应已开始写出，无法再更改状态码；此处只能放弃。
		// 调用方中间件会记录该错误。
		return
	}
}

// writeError 写出统一格式的错误响应。
func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, ErrorResponse{Error: msg})
}
