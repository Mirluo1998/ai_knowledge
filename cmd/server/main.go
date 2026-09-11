// Command server 是知识服务的入口。
// 负责加载配置、初始化依赖、组装各层（依赖注入）并启动 HTTP 服务。
package main

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	_ "github.com/go-sql-driver/mysql" // 注册 MySQL 驱动

	"knowledge/internal/config"
	"knowledge/internal/handler"
	"knowledge/internal/middleware"
	"knowledge/internal/repository"
	"knowledge/internal/service"
)

func main() {
	os.Exit(run())
}

// run 执行完整的服务生命周期，返回进程退出码。
// 拆出 run 而不是把所有逻辑塞进 main，便于测试与 defer 管理。
func run() int {
	cfg := config.Load()

	// 根据配置初始化日志：开发环境用 text 方便阅读，生产环境用 json 方便采集。
	logger := newLogger(cfg)
	slog.SetDefault(logger)

	db, err := initDB(cfg)
	if err != nil {
		logger.Error("init database failed", "error", err)
		return 1
	}
	defer db.Close()

	mux := newRouter(cfg, db, logger)

	srv := &http.Server{
		Addr:              cfg.ServerAddr,
		Handler:           mux,
		ReadHeaderTimeout: cfg.ReadHeaderTimeout,
		ReadTimeout:       cfg.ReadTimeout,
		WriteTimeout:      cfg.WriteTimeout,
		IdleTimeout:       cfg.IdleTimeout,
	}

	// 监听系统信号，收到后触发优雅关闭。
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	errCh := make(chan error, 1)
	go func() {
		logger.Info("server starting", "addr", cfg.ServerAddr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
	}()

	select {
	case err := <-errCh:
		logger.Error("server error", "error", err)
		return 1
	case <-ctx.Done():
		logger.Info("shutdown signal received")
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		logger.Error("graceful shutdown failed", "error", err)
		return 1
	}

	logger.Info("server stopped")
	return 0
}

// initDB 打开数据库连接并配置连接池参数。
func initDB(cfg config.Config) (*sql.DB, error) {
	db, err := sql.Open(cfg.DBDriver, cfg.DBDSN)
	if err != nil {
		return nil, err
	}

	db.SetMaxOpenConns(cfg.DBMaxOpenConns)
	db.SetMaxIdleConns(cfg.DBMaxIdleConns)
	db.SetConnMaxLifetime(cfg.DBConnMaxLifetime)

	// 启动时立即验证连接可用，快速失败。
	pingCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := db.PingContext(pingCtx); err != nil {
		return nil, err
	}
	return db, nil
}

// newRouter 组装依赖并构建路由。
func newRouter(cfg config.Config, db *sql.DB, logger *slog.Logger) http.Handler {
	// 依赖注入：repository -> service -> handler，方向单一、便于替换与测试。
	knowledgeRepo := repository.NewKnowledgeRepository(db)
	knowledgeSvc := service.NewKnowledgeService(knowledgeRepo)
	userRepo := repository.NewUserRepository(db)
	userSvc := service.NewUserService(userRepo, logger)

	knowledgeHandler := handler.NewKnowledgeHandler(knowledgeSvc, logger)
	userHandler := handler.NewUserHandler(logger, userSvc)

	mux := http.NewServeMux()

	// 健康检查：用于负载均衡 / 编排系统的存活与就绪探测。
	mux.HandleFunc("GET /healthz", healthz(cfg, db))

	// 业务路由：Go 1.22+ 的 ServeMux 支持方法匹配。
	mux.HandleFunc("GET /api/v1/knowledge", knowledgeHandler.ListKnowledge)
	mux.HandleFunc("PUT /api/v1/knowledge", knowledgeHandler.CreateKnowledge)
	mux.HandleFunc("POST /api/v1/user/register", userHandler.Register)
	mux.HandleFunc("GET /api/v1/user/get", userHandler.GetUser)
	mux.HandleFunc("POST /api/v1/user/login", userHandler.Login)

	return middleware.Chain(mux,
		middleware.Recover(logger),
		middleware.AccessLog(logger),
	)
}

// newLogger 根据配置创建日志器。
func newLogger(cfg config.Config) *slog.Logger {
	var level slog.Level
	switch cfg.LogLevel {
	case "debug":
		level = slog.LevelDebug
	case "warn", "warning":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	default:
		level = slog.LevelInfo
	}

	opts := &slog.HandlerOptions{Level: level}

	var handler slog.Handler
	if cfg.LogFormat == "text" {
		handler = slog.NewTextHandler(os.Stdout, opts)
	} else {
		handler = slog.NewJSONHandler(os.Stdout, opts)
	}

	return slog.New(handler)
}

// healthz 返回服务与数据库的健康状态。
func healthz(cfg config.Config, db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), cfg.HealthCheckTimeout)
		defer cancel()

		status := http.StatusOK
		state := "ok"
		if err := db.PingContext(ctx); err != nil {
			status = http.StatusServiceUnavailable
			state = "database unavailable"
		}

		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(status)
		_, _ = w.Write([]byte(`{"status":"` + state + `"}` + "\n"))
	}
}
