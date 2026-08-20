// Package config 负责加载应用配置。
//
// 加载优先级（从低到高）：
//  1. 代码内默认值
//  2. config.yaml 中对应环境段的配置（通过 APP_ENV 选择环境）
//  3. 环境变量覆盖
//
// 配置文件路径可通过 CONFIG_FILE 指定，默认 configs/config.yaml。
// 环境通过 APP_ENV 指定，默认 development。
//
// 新增配置项只需要三步：
//  1. 在 Config 结构体加字段，标好 yaml（扁平路径，如 "server.addr"）/ env tag
//  2. 在 defaults() 里设置默认值
//  3. 在 config.yaml 各环境段加上对应 key
//
// 无需修改合并逻辑。
package config

import (
	"fmt"
	"log/slog"
	"os"
	"reflect"
	"strconv"
	"time"

	"gopkg.in/yaml.v3"
)

// Config 聚合应用运行所需的全部配置。
//
// 字段通过 yaml tag 的点式路径（如 "server.addr"）与 YAML 嵌套结构对应，
// 通过 env tag 与环境变量对应。
// 新增字段只需加 tag，合并逻辑自动处理。
type Config struct {
	// Server
	ServerAddr         string        `yaml:"server.addr" env:"SERVER_ADDR"`
	ReadHeaderTimeout  time.Duration `yaml:"server.read_header_timeout" env:"READ_HEADER_TIMEOUT"`
	ReadTimeout        time.Duration `yaml:"server.read_timeout" env:"READ_TIMEOUT"`
	WriteTimeout       time.Duration `yaml:"server.write_timeout" env:"WRITE_TIMEOUT"`
	IdleTimeout        time.Duration `yaml:"server.idle_timeout" env:"IDLE_TIMEOUT"`
	ShutdownTimeout    time.Duration `yaml:"server.shutdown_timeout" env:"SHUTDOWN_TIMEOUT"`
	HealthCheckTimeout time.Duration `yaml:"server.health_check_timeout" env:"HEALTH_CHECK_TIMEOUT"`

	// Database
	DBDriver          string        `yaml:"database.driver" env:"DB_DRIVER"`
	DBDSN             string        `yaml:"database.dsn" env:"DB_DSN"`
	DBMaxOpenConns    int           `yaml:"database.max_open_conns" env:"DB_MAX_OPEN_CONNS"`
	DBMaxIdleConns    int           `yaml:"database.max_idle_conns" env:"DB_MAX_IDLE_CONNS"`
	DBConnMaxLifetime time.Duration `yaml:"database.conn_max_lifetime" env:"DB_CONN_MAX_LIFETIME"`

	// Log
	LogLevel  string `yaml:"log.level" env:"LOG_LEVEL"`
	LogFormat string `yaml:"log.format" env:"LOG_FORMAT"`
}

// rawFile 是 YAML 文件的顶层结构：按环境分段。
// 每个环境段内部是一个 map[string]interface{}，
// 我们通过 yaml tag 的点式路径手工取值，避免再定义一套嵌套结构体。
type rawFile struct {
	Development map[string]interface{} `yaml:"development"`
	Staging     map[string]interface{} `yaml:"staging"`
	Production  map[string]interface{} `yaml:"production"`
}

const (
	defaultEnv      = "development"
	defaultConfPath = "configs/config.yaml"
)

// Load 加载配置。
// 优先级：环境变量 > YAML 对应环境段 > 代码默认值。
func Load() Config {
	// 先拿环境名和配置文件路径（这两个只能从环境变量来）
	env := getEnv("APP_ENV", defaultEnv)
	confPath := getEnv("CONFIG_FILE", defaultConfPath)

	// 1. 从代码默认值开始
	cfg := defaults()

	// 2. 加载 YAML 并合并对应环境段
	if err := mergeYAML(&cfg, confPath, env); err != nil {
		slog.Warn("load config file failed, falling back to env vars and defaults",
			"path", confPath, "env", env, "error", err)
	}

	// 3. 环境变量覆盖
	mergeEnv(&cfg)

	return cfg
}

// defaults 返回代码内置的默认值。
func defaults() Config {
	return Config{
		ServerAddr:         ":8080",
		ReadHeaderTimeout:  5 * time.Second,
		ReadTimeout:        10 * time.Second,
		WriteTimeout:       15 * time.Second,
		IdleTimeout:        60 * time.Second,
		ShutdownTimeout:    10 * time.Second,
		HealthCheckTimeout: 2 * time.Second,

		DBDriver:          "mysql",
		DBDSN:             "root:root@tcp(127.0.0.1:3306)/knowledge?parseTime=true&loc=Local&charset=utf8mb4",
		DBMaxOpenConns:    25,
		DBMaxIdleConns:    10,
		DBConnMaxLifetime: 5 * time.Minute,

		LogLevel:  "info",
		LogFormat: "json",
	}
}

// mergeYAML 读取 YAML 文件，按 yaml tag 的点式路径把对应环境段的非零值合并到 cfg。
func mergeYAML(cfg *Config, path, env string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read config file: %w", err)
	}

	var file rawFile
	if err := yaml.Unmarshal(data, &file); err != nil {
		return fmt.Errorf("parse config yaml: %w", err)
	}

	var raw map[string]interface{}
	switch env {
	case "development":
		raw = file.Development
	case "staging":
		raw = file.Staging
	case "production":
		raw = file.Production
	default:
		return fmt.Errorf("unknown environment %q (expected development/staging/production)", env)
	}

	if raw == nil {
		return fmt.Errorf("environment %q not found in config file", env)
	}

	applyYAMLMap(cfg, raw)
	return nil
}

// applyYAMLMap 遍历 cfg 的字段，按 yaml tag 的点式路径从 raw 取值并设置。
// yaml tag 为空的字段跳过；raw 中不存在或值为零的不覆盖。
func applyYAMLMap(cfg *Config, raw map[string]interface{}) {
	v := reflect.ValueOf(cfg).Elem()
	t := v.Type()

	for i := 0; i < v.NumField(); i++ {
		field := v.Field(i)
		if !field.CanSet() {
			continue
		}

		yamlPath := t.Field(i).Tag.Get("yaml")
		if yamlPath == "" {
			continue
		}

		val, ok := lookupYAMLPath(raw, yamlPath)
		if !ok {
			continue
		}

		// 零值不覆盖
		if isZeroValue(val) {
			continue
		}

		setFieldFromYAML(field, val)
	}
}

// lookupYAMLPath 按点式路径（如 "server.addr"）在嵌套 map 中查找值。
func lookupYAMLPath(m map[string]interface{}, path string) (interface{}, bool) {
	parts := splitPath(path)
	current := interface{}(m)

	for i, part := range parts {
		cm, ok := current.(map[string]interface{})
		if !ok {
			return nil, false
		}
		val, exists := cm[part]
		if !exists {
			return nil, false
		}
		if i == len(parts)-1 {
			return val, true
		}
		current = val
	}
	return nil, false
}

// splitPath 按点分割路径。
func splitPath(p string) []string {
	// 简单实现，够用即可
	var parts []string
	start := 0
	for i := 0; i < len(p); i++ {
		if p[i] == '.' {
			parts = append(parts, p[start:i])
			start = i + 1
		}
	}
	parts = append(parts, p[start:])
	return parts
}

// isZeroValue 判断 YAML 解析出的值是不是"零值"（用来决定是否覆盖默认值）。
func isZeroValue(v interface{}) bool {
	if v == nil {
		return true
	}
	switch val := v.(type) {
	case string:
		return val == ""
	case int:
		return val == 0
	case int64:
		return val == 0
	case float64:
		return val == 0
	case bool:
		return !val
	case []interface{}:
		return len(val) == 0
	case map[string]interface{}:
		return len(val) == 0
	}
	return false
}

// setFieldFromYAML 把 YAML 中拿到的值设置到字段上，自动做类型转换。
func setFieldFromYAML(field reflect.Value, val interface{}) {
	switch field.Kind() {
	case reflect.String:
		if s, ok := val.(string); ok {
			field.SetString(s)
		}
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		// time.Duration 是 int64 别名
		if field.Type() == reflect.TypeOf(time.Duration(0)) {
			// YAML 中时长是字符串，如 "5s"
			if s, ok := val.(string); ok {
				if d, err := time.ParseDuration(s); err == nil {
					field.SetInt(int64(d))
				}
			}
			return
		}
		switch n := val.(type) {
		case int:
			field.SetInt(int64(n))
		case int64:
			field.SetInt(n)
		case float64:
			field.SetInt(int64(n))
		}
	case reflect.Bool:
		if b, ok := val.(bool); ok {
			field.SetBool(b)
		}
	}
}

// mergeEnv 遍历 cfg 结构体，对每个带 env tag 的字段，
// 如果对应的环境变量已设置且非空，则用环境变量的值覆盖。
func mergeEnv(cfg *Config) {
	v := reflect.ValueOf(cfg).Elem()
	t := v.Type()

	for i := 0; i < v.NumField(); i++ {
		field := v.Field(i)
		if !field.CanSet() {
			continue
		}

		envKey := t.Field(i).Tag.Get("env")
		if envKey == "" {
			continue
		}

		envVal, ok := os.LookupEnv(envKey)
		if !ok || envVal == "" {
			continue
		}

		setFieldFromString(field, envVal)
	}
}

// setFieldFromString 把字符串解析成对应类型并设置到字段上。
// 支持的类型：string, int*, bool, time.Duration。
// 解析失败则跳过（保留原值）。
func setFieldFromString(field reflect.Value, s string) {
	switch field.Kind() {
	case reflect.String:
		field.SetString(s)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		if field.Type() == reflect.TypeOf(time.Duration(0)) {
			d, err := time.ParseDuration(s)
			if err != nil {
				return
			}
			field.SetInt(int64(d))
			return
		}
		n, err := strconv.ParseInt(s, 10, 64)
		if err != nil {
			return
		}
		field.SetInt(n)
	case reflect.Bool:
		b, err := strconv.ParseBool(s)
		if err != nil {
			return
		}
		field.SetBool(b)
	}
}

// getEnv 读取环境变量，空值或未设置时返回 fallback。
func getEnv(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return fallback
}
