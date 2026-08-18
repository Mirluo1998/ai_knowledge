# Knowledge Service

知识条目管理服务的后端 API，基于 Go 标准库 `net/http` + `database/sql`（MySQL）。

## 项目结构

采用 Go 社区标准的 `cmd/` + `internal/` 布局，分层清晰、依赖方向单一（handler → service → repository）：

```
.
├── cmd/
│   └── server/              # 程序入口：配置加载、依赖注入、优雅关闭
├── internal/                # 私有包，禁止外部模块引用
│   ├── config/              # 环境变量配置
│   ├── handler/             # HTTP 接入层（请求解析 / 响应编码）
│   ├── middleware/          # 中间件（panic 恢复、访问日志）
│   ├── model/               # 领域模型
│   ├── repository/          # 数据访问层（MySQL 实现）
│   └── service/             # 业务逻辑层
├── Makefile
├── go.mod
└── go.sum
```

### 设计约定

- **依赖注入**：`main` 负责组装，各层通过构造函数注入依赖，不在包内使用全局变量。
- **面向接口**：接口定义在消费方（service 定义所需 repository 接口，handler 定义所需 service 接口），便于单元测试与实现替换。
- **context 贯穿**：所有数据库与业务调用传递 `context.Context`，支持超时与取消。
- **错误处理**：底层错误用 `%w` 包装后向上传递；handler 统一转换为 HTTP 状态码，内部细节只进日志、不出响应体。
- **结构化日志**：使用标准库 `log/slog`（JSON 输出），可直接对接日志采集系统。

## 快速开始

### 前置条件

- Go 1.26+
- MySQL（数据库名：`knowledge`）

建表参考：

```sql
CREATE TABLE knowledge (
    id         BIGINT       NOT NULL AUTO_INCREMENT PRIMARY KEY,
    type       VARCHAR(64)  NOT NULL,
    title      VARCHAR(255) NOT NULL,
    content    TEXT         NOT NULL,
    created_at DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
);
```

### 运行

```bash
# 本地直接运行（使用默认配置，按需设置 DB_DSN）
make run

# 或编译后运行
make build
./bin/knowledge-server
```

### 常用命令

```bash
make test    # 单元测试（含竞态检测）
make vet     # 静态检查
make fmt     # 代码格式化
make cover   # 覆盖率报告
```

## 配置

全部通过环境变量注入（12-Factor），未设置时使用默认值：

| 变量 | 说明 | 默认值 |
|---|---|---|
| `SERVER_ADDR` | HTTP 监听地址 | `:8080` |
| `DB_DRIVER` | 数据库驱动 | `mysql` |
| `DB_DSN` | 数据库连接串 | `root:root@tcp(127.0.0.1:3306)/knowledge?parseTime=true&loc=Local&charset=utf8mb4` |
| `DB_MAX_OPEN_CONNS` | 最大打开连接数 | `25` |
| `DB_MAX_IDLE_CONNS` | 最大空闲连接数 | `10` |
| `DB_CONN_MAX_LIFETIME` | 连接最长存活时间 | `5m` |
| `READ_HEADER_TIMEOUT` | 读请求头超时 | `5s` |
| `READ_TIMEOUT` | 读请求超时 | `10s` |
| `WRITE_TIMEOUT` | 写响应超时 | `15s` |
| `IDLE_TIMEOUT` | keep-alive 空闲超时 | `60s` |
| `SHUTDOWN_TIMEOUT` | 优雅关闭等待时间 | `10s` |

## API

| 方法 | 路径 | 说明 |
|---|---|---|
| `GET` | `/healthz` | 健康检查（含数据库探活） |
| `GET` | `/api/v1/knowledge` | 知识列表，支持 `?type=&title=` 过滤 |

示例：

```bash
curl 'http://localhost:8080/api/v1/knowledge?type=faq&title=部署'
```

响应：

```json
[
  {
    "id": 1,
    "type": "faq",
    "title": "部署流程",
    "content": "...",
    "created_at": "2026-08-18T10:00:00+08:00",
    "updated_at": "2026-08-18T10:00:00+08:00"
  }
]
```
