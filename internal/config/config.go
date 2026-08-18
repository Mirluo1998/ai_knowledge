// Package config 负责从环境变量加载应用配置。
// 遵循 12-Factor 原则：环境差异全部通过环境变量注入，
// 代码中只保留开发与测试用的默认值。
package config

import (
	"os"
	"strconv"
	"time"
)

// Config 聚合应用运行所需的全部配置。
type Config struct {
	// Server
	ServerAddr         string        // HTTP 监听地址，如 ":8080"
	ReadHeaderTimeout  time.Duration // 读取请求头超时
	ReadTimeout        time.Duration // 读取整个请求超时
	WriteTimeout       time.Duration // 写响应超时
	IdleTimeout        time.Duration // keep-alive 空闲超时
	ShutdownTimeout    time.Duration // 优雅关闭等待时间
	HealthCheckTimeout time.Duration // 健康检查探测超时

	// Database
	DBDriver          string        // database/sql 驱动名
	DBDSN             string        // 数据源连接串
	DBMaxOpenConns    int           // 最大打开连接数
	DBMaxIdleConns    int           // 最大空闲连接数
	DBConnMaxLifetime time.Duration // 连接最长存活时间
}

// Load 从环境变量读取配置，未设置时使用默认值。
func Load() Config {
	return Config{
		ServerAddr:         getEnv("SERVER_ADDR", ":8080"),
		ReadHeaderTimeout:  getEnvDuration("READ_HEADER_TIMEOUT", 5*time.Second),
		ReadTimeout:        getEnvDuration("READ_TIMEOUT", 10*time.Second),
		WriteTimeout:       getEnvDuration("WRITE_TIMEOUT", 15*time.Second),
		IdleTimeout:        getEnvDuration("IDLE_TIMEOUT", 60*time.Second),
		ShutdownTimeout:    getEnvDuration("SHUTDOWN_TIMEOUT", 10*time.Second),
		HealthCheckTimeout: getEnvDuration("HEALTH_CHECK_TIMEOUT", 2*time.Second),

		DBDriver:          getEnv("DB_DRIVER", "mysql"),
		DBDSN:             getEnv("DB_DSN", "root:root@tcp(127.0.0.1:3306)/knowledge?parseTime=true&loc=Local&charset=utf8mb4"),
		DBMaxOpenConns:    getEnvInt("DB_MAX_OPEN_CONNS", 25),
		DBMaxIdleConns:    getEnvInt("DB_MAX_IDLE_CONNS", 10),
		DBConnMaxLifetime: getEnvDuration("DB_CONN_MAX_LIFETIME", 5*time.Minute),
	}
}

func getEnv(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return fallback
}

func getEnvDuration(key string, fallback time.Duration) time.Duration {
	v, ok := os.LookupEnv(key)
	if !ok || v == "" {
		return fallback
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		return fallback
	}
	return d
}

func getEnvInt(key string, fallback int) int {
	v, ok := os.LookupEnv(key)
	if !ok || v == "" {
		return fallback
	}
	n, err := strconv.Atoi(v)
	if err != nil || n <= 0 {
		return fallback
	}
	return n
}
