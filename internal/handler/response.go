package handler

import (
	"encoding/json"
	"errors"
	"net/http"
)

// ErrorResponse 是统一的错误响应结构。
// 对外只暴露可读信息，内部错误细节仅写入日志。
type ErrorResponse struct {
	Error string `json:"error"`
}

// maxRequestBodyBytes 限制请求体大小为 1 MiB，防止超大 body 耗尽内存。
const maxRequestBodyBytes = 1 << 20

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

// decodeJSONBody 限制请求体大小并把 JSON 解码到 dst 中。
func decodeJSONBody(w http.ResponseWriter, r *http.Request, dst any) error {
	r.Body = http.MaxBytesReader(w, r.Body, maxRequestBodyBytes)
	defer r.Body.Close()
	return json.NewDecoder(r.Body).Decode(dst)
}

// writeBodyDecodeError 把请求体解码失败统一映射为 413（过大）或 400（格式错误）。
func writeBodyDecodeError(w http.ResponseWriter, err error) {
	var maxBytesErr *http.MaxBytesError
	if errors.As(err, &maxBytesErr) {
		writeJSON(w, http.StatusRequestEntityTooLarge, Fail("request body too large"))
		return
	}
	writeJSON(w, http.StatusBadRequest, Fail("invalid request body"))
}

type Result struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data"`
	Message string      `json:"message"`
}

func Success(data interface{}) Result {
	return Result{
		Success: true,
		Data:    data,
		Message: "success",
	}
}

func Fail(message string) Result {
	return Result{
		Success: false,
		Data:    nil,
		Message: message,
	}
}
