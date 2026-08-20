APP_NAME := knowledge-server
BIN_DIR  := bin

.PHONY: all build run test cover vet fmt tidy clean

all: fmt vet test build

## build: 编译服务端二进制到 bin/
build:
	go build -o $(BIN_DIR)/$(APP_NAME) ./cmd/server

## run: 本地启动服务（默认 development 环境）
run:
	go run ./cmd/server

## run-prod: 以 production 环境配置启动服务
run-prod:
	APP_ENV=production go run ./cmd/server

## run-staging: 以 staging 环境配置启动服务
run-staging:
	APP_ENV=staging go run ./cmd/server

## test: 运行全部单元测试（带竞态检测）
test:
	go test -race ./...

## cover: 生成测试覆盖率报告
cover:
	go test -race -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

## vet: 静态检查
vet:
	go vet ./...

## fmt: 格式化代码
fmt:
	gofmt -s -w .

## tidy: 整理模块依赖
tidy:
	go mod tidy

## clean: 清理构建产物
clean:
	rm -rf $(BIN_DIR) coverage.out coverage.html
