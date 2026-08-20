# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Knowledge Service — a Go backend API for managing knowledge entries, built with the standard library `net/http` and `database/sql` (MySQL). No external web frameworks or ORMs are used.

- Module path: `knowledge`
- Go version: 1.26+
- Single dependency: `github.com/go-sql-driver/mysql`

## Common Commands

```bash
make build    # compile binary to bin/knowledge-server
make run      # run the server locally
make test     # run all unit tests with race detection
go test -race ./internal/service/... -run TestListKnowledge   # run a single test
go test -race -run TestListKnowledge ./...                    # run matching tests across packages
make vet      # go vet static check
make fmt      # gofmt -s -w .
make cover    # generate coverage report (coverage.html)
make tidy     # go mod tidy
make clean    # remove build artifacts
```

Run server with custom config:
```bash
APP_ENV=production DB_DSN="user:pass@tcp(localhost:3306)/knowledge?parseTime=true" make run
```

## Architecture

Layered architecture with single-direction dependency: `handler → service → repository`. All layers communicate through interfaces defined by the consumer (not the implementer), enabling easy mocking in tests.

### Dependency injection

`cmd/server/main.go` is the composition root. It loads config, opens the DB, constructs repos → services → handlers, wires up the `http.ServeMux`, and wraps with middleware. No package-level global state.

### Routing

Uses Go 1.22+ enhanced `http.ServeMux` with method patterns (e.g. `"GET /api/v1/knowledge"`). Routes are registered in `newRouter()` in `cmd/server/main.go`.

### Layers

- **`internal/handler/`** — HTTP layer: parses requests, calls service, encodes JSON responses. Defines its own service interface (`KnowledgeService`) so it can be tested in isolation. Error handling maps `*service.ValidationError` → 400; everything else → 500 (internal details logged only, never leaked to clients).
- **`internal/service/`** — Business logic. Defines its own repository interface (`KnowledgeRepository`) following "accept interfaces, return structs". Wraps errors with `%w` for `errors.Is`/`errors.As` upstream. Validation errors use the sentinel type `*service.ValidationError`.
- **`internal/repository/`** — MySQL data access via `database/sql`. Explicit column lists (no `SELECT *`), parameterized queries, always calls `rows.Close()` with defer and checks `rows.Err()` after iteration.
- **`internal/model/`** — Domain structs (`Knowledge`, `KnowledgeQuery`). Query params are a separate struct from the entity.
- **`internal/middleware/`** — `Recover` (panic → 500 + log), `AccessLog` (method/path/status/duration), and a `Chain` helper. Order in `Chain` is outer-first.
- **`internal/config/`** — Loads config from `configs/config.yaml` with multi-environment support (selected via `APP_ENV`). Environment variables override YAML values for 12-Factor compatibility. Duration values parsed via `time.ParseDuration`.

### Error conventions

- Wrap with `fmt.Errorf("...: %w", err)` at each layer boundary.
- `*service.ValidationError` is the only typed error that crosses the service→handler boundary for 4xx responses.
- Handler never returns raw error messages to clients for 5xx — it logs them and responds with a generic message.

### Testing

- Unit tests live alongside source as `*_test.go` with `_test` package suffix (black-box).
- Service tests use in-memory fake implementations of the repository interface — no database needed.
- Tests use standard `testing` package only (no test frameworks).

### Configuration

Configuration is loaded from `configs/config.yaml` with multi-environment support. Priority (low → high): code defaults → YAML → environment variables.

**Environment selection** — set `APP_ENV` to choose the environment segment in YAML:
- `development` (default) — local dev, text logs at debug level
- `staging` — pre-production, JSON logs at info level
- `production` — live environment, JSON logs at info level, larger connection pool

**Config file path** — override with `CONFIG_FILE` (default: `configs/config.yaml`).

**Environment variables** override corresponding YAML fields:
- `APP_ENV` — environment name (development/staging/production)
- `CONFIG_FILE` — path to config YAML
- `SERVER_ADDR=:8080`
- `DB_DSN=root:root@tcp(127.0.0.1:3306)/knowledge?parseTime=true&loc=Local&charset=utf8mb4`
- `DB_MAX_OPEN_CONNS=25`, `DB_MAX_IDLE_CONNS=10`
- `LOG_LEVEL` — debug/info/warn/error
- `LOG_FORMAT` — text/json
- Graceful shutdown timeout: 10s

Run with a specific environment:
```bash
APP_ENV=production make run          # Linux/macOS
$env:APP_ENV='production'; make run  # PowerShell
```

### Logging

Structured JSON logs via `log/slog` on stdout. Logger is injected (not a global) into handlers and middleware; `slog.SetDefault` is set at startup for any ad-hoc usage.

### Database schema

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
