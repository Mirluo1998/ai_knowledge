// Package middleware 提供 HTTP 中间件。
package middleware

import (
	"log/slog"
	"net/http"
	"time"
)

// Recover 捕获 handler 中的 panic，避免进程崩溃，
// 并向客户端返回 500。
func Recover(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if rec := recover(); rec != nil {
					logger.Error("http handler panicked",
						"panic", rec,
						"method", r.Method,
						"path", r.URL.Path,
					)
					http.Error(w, "internal server error", http.StatusInternalServerError)
				}
			}()
			next.ServeHTTP(w, r)
		})
	}
}

// responseRecorder 包装 ResponseWriter 以记录状态码。
type responseRecorder struct {
	http.ResponseWriter
	status int
}

func (rr *responseRecorder) WriteHeader(code int) {
	rr.status = code
	rr.ResponseWriter.WriteHeader(code)
}

// AccessLog 记录每个请求的方法、路径、状态码与耗时。
func AccessLog(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			rr := &responseRecorder{ResponseWriter: w, status: http.StatusOK}

			next.ServeHTTP(rr, r)

			logger.Info("http request",
				"method", r.Method,
				"path", r.URL.Path,
				"status", rr.status,
				"duration", time.Since(start).String(),
				"remote", r.RemoteAddr,
			)
		})
	}
}

// Chain 按顺序将中间件应用到 handler，列表中靠前的中间件位于外层。
func Chain(h http.Handler, middlewares ...func(http.Handler) http.Handler) http.Handler {
	for i := len(middlewares) - 1; i >= 0; i-- {
		h = middlewares[i](h)
	}
	return h
}
