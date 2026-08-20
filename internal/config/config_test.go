package config_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"knowledge/internal/config"
)

// testdataDir 返回测试用配置文件所在目录。
func testdataDir(t *testing.T) string {
	t.Helper()
	// 测试用的 YAML 和项目根目录的 configs/config.yaml 一致，
	// 这里直接用项目根目录下的真实文件做集成测试。
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	// internal/config -> 项目根目录
	root := filepath.Join(wd, "..", "..")
	return filepath.Join(root, "configs")
}

func TestLoad_DefaultEnvUsesDevelopment(t *testing.T) {
	// 清理环境变量，确保不影响默认值
	os.Unsetenv("APP_ENV")
	os.Unsetenv("SERVER_ADDR")
	os.Unsetenv("DB_DSN")
	os.Unsetenv("LOG_LEVEL")

	confPath := filepath.Join(testdataDir(t), "config.yaml")
	os.Setenv("CONFIG_FILE", confPath)
	defer os.Unsetenv("CONFIG_FILE")

	cfg := config.Load()

	// development 段的默认端口应该是 8080
	if cfg.ServerAddr != ":8080" {
		t.Errorf("ServerAddr = %q, want \":8080\"", cfg.ServerAddr)
	}

	// development 段的日志级别应该是 debug
	if cfg.LogLevel != "debug" {
		t.Errorf("LogLevel = %q, want \"debug\"", cfg.LogLevel)
	}

	// development 段的日志格式应该是 text
	if cfg.LogFormat != "text" {
		t.Errorf("LogFormat = %q, want \"text\"", cfg.LogFormat)
	}
}

func TestLoad_ProductionEnv(t *testing.T) {
	os.Setenv("APP_ENV", "production")
	defer os.Unsetenv("APP_ENV")

	confPath := filepath.Join(testdataDir(t), "config.yaml")
	os.Setenv("CONFIG_FILE", confPath)
	defer os.Unsetenv("CONFIG_FILE")

	// 确保环境变量不覆盖
	os.Unsetenv("SERVER_ADDR")
	os.Unsetenv("DB_MAX_OPEN_CONNS")
	os.Unsetenv("LOG_LEVEL")
	os.Unsetenv("LOG_FORMAT")

	cfg := config.Load()

	if cfg.ServerAddr != ":8080" {
		t.Errorf("ServerAddr = %q, want \":8080\"", cfg.ServerAddr)
	}

	// production 的 max_open_conns 是 100
	if cfg.DBMaxOpenConns != 100 {
		t.Errorf("DBMaxOpenConns = %d, want 100", cfg.DBMaxOpenConns)
	}

	if cfg.LogLevel != "info" {
		t.Errorf("LogLevel = %q, want \"info\"", cfg.LogLevel)
	}

	if cfg.LogFormat != "json" {
		t.Errorf("LogFormat = %q, want \"json\"", cfg.LogFormat)
	}

	// production 的 shutdown_timeout 是 30s
	if cfg.ShutdownTimeout != 30*time.Second {
		t.Errorf("ShutdownTimeout = %v, want 30s", cfg.ShutdownTimeout)
	}
}

func TestLoad_StagingEnv(t *testing.T) {
	os.Setenv("APP_ENV", "staging")
	defer os.Unsetenv("APP_ENV")

	confPath := filepath.Join(testdataDir(t), "config.yaml")
	os.Setenv("CONFIG_FILE", confPath)
	defer os.Unsetenv("CONFIG_FILE")

	os.Unsetenv("DB_MAX_OPEN_CONNS")

	cfg := config.Load()

	if cfg.DBMaxOpenConns != 50 {
		t.Errorf("DBMaxOpenConns = %d, want 50", cfg.DBMaxOpenConns)
	}
}

func TestLoad_EnvVarOverridesYAML(t *testing.T) {
	os.Setenv("APP_ENV", "production")
	defer os.Unsetenv("APP_ENV")

	confPath := filepath.Join(testdataDir(t), "config.yaml")
	os.Setenv("CONFIG_FILE", confPath)
	defer os.Unsetenv("CONFIG_FILE")

	// 用环境变量覆盖 YAML 中的值
	os.Setenv("SERVER_ADDR", ":9090")
	defer os.Unsetenv("SERVER_ADDR")

	os.Setenv("DB_MAX_OPEN_CONNS", "200")
	defer os.Unsetenv("DB_MAX_OPEN_CONNS")

	os.Setenv("LOG_LEVEL", "debug")
	defer os.Unsetenv("LOG_LEVEL")

	cfg := config.Load()

	if cfg.ServerAddr != ":9090" {
		t.Errorf("ServerAddr = %q, want \":9090\" (env var should override yaml)", cfg.ServerAddr)
	}

	if cfg.DBMaxOpenConns != 200 {
		t.Errorf("DBMaxOpenConns = %d, want 200 (env var should override yaml)", cfg.DBMaxOpenConns)
	}

	if cfg.LogLevel != "debug" {
		t.Errorf("LogLevel = %q, want \"debug\" (env var should override yaml)", cfg.LogLevel)
	}
}

func TestLoad_MissingConfigFile_FallsBackToDefaults(t *testing.T) {
	os.Setenv("CONFIG_FILE", "/nonexistent/path/config.yaml")
	defer os.Unsetenv("CONFIG_FILE")
	os.Unsetenv("APP_ENV")
	os.Unsetenv("SERVER_ADDR")
	os.Unsetenv("DB_DSN")

	cfg := config.Load()

	// 应该退回默认值
	if cfg.ServerAddr != ":8080" {
		t.Errorf("ServerAddr = %q, want default \":8080\"", cfg.ServerAddr)
	}

	if cfg.DBMaxOpenConns != 25 {
		t.Errorf("DBMaxOpenConns = %d, want default 25", cfg.DBMaxOpenConns)
	}
}
