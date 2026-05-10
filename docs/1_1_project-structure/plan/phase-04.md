---
phase: 4
name: Server Entry Point
layer: composition root
depends_on: [phase-01, phase-03]
plan: ./README.md
---

# Phase 4: Server Entry Point

## Цель

Создать точку входа `cmd/server/main.go`, health-handler, тесты, добавить `chi/v5` в зависимости. После этой фазы `make run` поднимает HTTP-сервер с health-check, `curl /api/v1/health` возвращает 200, `Ctrl+C` корректно завершает за ≤5с.

## Контекст

Что произвели предыдущие фазы:
- Phase 1: `Makefile` с `make run` (`go run ./cmd/server`), `.golangci.yml`.
- Phase 3: пакет `config/` с `Load(Lookuper) (*Config, error)`, `OsLookuper{}`, `ErrConfigInvalid`, `ValidationError{Code, Field, Reason}`. Геттеры: `cfg.ServerPort()`, `cfg.DatabaseURL()`, `cfg.JWTSecret()`.

Эта фаза добавляет единственную внешнюю зависимость в `go.mod`: `github.com/go-chi/chi/v5` (ADR-012 в `../03-decisions.md`).

## Файлы для создания

### `cmd/server/main.go`

**Назначение:** composition root по `prompts/Architecture Layers.txt:60-66`. Загружает конфиг, строит логгер и роутер, запускает HTTP-сервер с graceful shutdown.

**Детали реализации:**

```go
// Команда server — точка входа HTTP API проекта Rupor.
//
// Composition root. Не содержит бизнес-логики, только инициализацию и склейку.
// См. prompts/Architecture Layers.txt и prompts/Clean architecture.txt.
package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/dovgalb/project-rupor/config"
)

const shutdownTimeout = 5 * time.Second

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	cfg, err := config.Load(config.OsLookuper{})
	if err != nil {
		var verr config.ValidationError
		if errors.As(err, &verr) {
			logger.Error("config invalid",
				slog.String("code", verr.Code),
				slog.String("field", verr.Field),
				slog.String("reason", verr.Reason),
			)
		} else {
			logger.Error("config load failed", slog.Any("err", err))
		}
		os.Exit(1)
	}

	if err := run(cfg, logger); err != nil {
		logger.Error("server stopped with error", slog.Any("err", err))
		os.Exit(1)
	}
}

// run собирает HTTP-сервер и блокирует до сигнала или фатальной ошибки.
// Выделена из main() ради тестируемости и переиспользования в smoke-тестах.
func run(cfg *config.Config, logger *slog.Logger) error {
	mux := chi.NewRouter()
	mux.Route("/api/v1", func(r chi.Router) {
		r.Get("/health", healthHandler)
	})

	addr := ":" + strconv.Itoa(cfg.ServerPort())
	srv := &http.Server{
		Addr:              addr,
		Handler:           mux,
		ReadHeaderTimeout: 10 * time.Second,
	}

	// Подписываемся на SIGINT/SIGTERM до запуска ListenAndServe,
	// чтобы не пропустить сигнал, пришедший в момент старта.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	serverErr := make(chan error, 1)
	go func() {
		logger.Info("server starting", slog.String("addr", addr))
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErr <- err
		}
		close(serverErr)
	}()

	select {
	case err := <-serverErr:
		if err != nil {
			return err
		}
	case <-ctx.Done():
		logger.Info("shutdown signal received")
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		return err
	}

	logger.Info("shutdown complete")
	return nil
}
```

**Что важно:**
- `package main` (`prompts/Go style.txt:62-64`).
- `slog.SetDefault(logger)` — чтобы вызовы `slog.Info(...)` из других пакетов в будущем шли в один формат.
- `config.Load` использует `OsLookuper{}` — реальная env. Через инжекцию `Lookuper` тесты обходятся без `os.Setenv`.
- Логирование структурированное (`prompts/Go style.txt:91-94`), JSON-формат для production-ready вывода.
- `ReadHeaderTimeout` — защита от Slowloris-атак, дешёвая практика.
- `signal.NotifyContext` — стандартный Go 1.16+ путь для graceful shutdown.
- `srv.Shutdown(shutdownCtx)` блокирует до завершения активных запросов или таймаута 5с (ADR-014).
- Функция `run` отделена от `main` — её можно вызывать из тестов с фейковой конфигурацией. `main` отвечает только за чтение env, логгер и `os.Exit`.
- `serverErr` канал размером 1 — чтобы горутина не блокировалась при `serverErr <- err` после возврата из `select`.
- `panic` нет — все фатальные ошибки идут через `os.Exit(1)` после структурированного лога (см. `../07-standards.md` Расхождение 1).

**Соответствие дизайну:**
- `../01-architecture.md` раздел «Компоненты `cmd/server/`» — структура и порядок инициализации.
- `../02-behavior.md` UC1 (запуск+health) и UC2 (graceful shutdown) — sequence-диаграммы.
- `../08-api-contract.md` — `/api/v1/health` зарегистрирован под префиксом `/api/v1`.

### `cmd/server/health.go`

**Назначение:** обработчик `GET /api/v1/health`, отделён от `main.go` для отдельного тестирования.

**Детали реализации:**

```go
package main

import (
	"net/http"
)

// healthHandler отвечает на GET /api/v1/health.
// Не зависит от БД (см. ADR-006 в docs/project-structure/03-decisions.md).
func healthHandler(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	// Тело фиксировано контрактом docs/project-structure/08-api-contract.md.
	_, _ = w.Write([]byte(`{"status":"ok"}` + "\n"))
}
```

**Что важно:**
- `package main` — тот же пакет, что и `main.go`.
- Тело ответа — литерал, без `encoding/json`. Это сознательное упрощение: для одного фиксированного ответа парсинг/сериализация — оверхед. Если контракт расширится (`version`, `git_sha`), переключимся на `json.NewEncoder`.
- `_, _ = w.Write(...)` — ошибку записи в response writer игнорировать допустимо (`prompts/Go style.txt:42-43`: «`_ = x.Close()` допустимо для read-only ресурсов» — здесь аналогично, но логирование добавило бы шум на каждый разорванный коннект).
- Никакого `r *http.Request` использования — параметр под `_`, чтобы линтер `unused-parameter` не ругался (он отключён в Phase 1, но для будущего).

**Соответствие дизайну:**
- `../08-api-contract.md` — точный JSON `{"status":"ok"}`, статус 200, Content-Type.
- `../02-behavior.md` UC1 — sequence health-check.

### `cmd/server/health_test.go`

**Назначение:** white-box тест на `healthHandler`.

**Детали реализации:**

```go
package main

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHealthHandler_Returns200(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/health", nil)
	rr := httptest.NewRecorder()

	healthHandler(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
	}

	gotCT := rr.Header().Get("Content-Type")
	wantCT := "application/json; charset=utf-8"
	if gotCT != wantCT {
		t.Fatalf("Content-Type = %q, want %q", gotCT, wantCT)
	}

	body, err := io.ReadAll(rr.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	if !strings.Contains(string(body), `"status":"ok"`) {
		t.Fatalf("body = %q, want contain 'status:ok'", string(body))
	}
}
```

**Что важно:**
- `package main` (white-box) — нужно, чтобы вызвать приватный `healthHandler`. Решение зафиксировано в `../07-standards.md` Расхождение 4.
- `httptest.NewRecorder()` — стандартный путь для тестирования HTTP-handler без поднятия сервера.
- `t.Parallel()` (`prompts/Tests Style.txt:28-30`).
- Точное сравнение Content-Type (а не `Contains`) — контракт фиксированный.
- Для тела используется `Contains` — на случай возможных trailing-`\n`/whitespace, чтобы тест не ломался на форматных мелочах.

**Соответствие дизайну:**
- `../04-testing.md` — `TestHealthHandler_Returns200`.

## Файлы для модификации

### `go.mod`

**Что меняется:** добавляется `require github.com/go-chi/chi/v5 vX.Y.Z` через `go get github.com/go-chi/chi/v5`.

**Команда оператору:**
```bash
go get github.com/go-chi/chi/v5
go mod tidy
```

**Ожидаемый результат:** `go.mod` содержит:
```go
module github.com/dovgalb/project-rupor

go 1.25

require github.com/go-chi/chi/v5 v5.X.Y
```

### `go.sum`

**Что меняется:** создаётся автоматически после `go mod tidy`. Содержит хеши `chi/v5` и его транзитивных зависимостей (у chi их нет).

## Ключевые решения

- **ADR-002** (один бинарник `cmd/server/`) — реализуется здесь.
- **ADR-006** (health не зависит от БД) — `healthHandler` действительно не делает никаких I/O.
- **ADR-012** (chi/v5 как единственная новая зависимость) — `go get` именно этого пакета и никаких других.
- **ADR-014** (graceful shutdown 5с) — `shutdownTimeout = 5 * time.Second`.

См. `../03-decisions.md` для полного контекста.

## Verification

- [ ] `go build ./...` — exit 0, бинарник собирается через `make build` в `bin/server`
- [ ] `go.mod` содержит ровно одну запись `require`: `github.com/go-chi/chi/v5`
- [ ] `go.sum` существует и tracked
- [ ] `go test ./cmd/server/... -race -count=1 -v` — `TestHealthHandler_Returns200` проходит
- [ ] `golangci-lint run ./cmd/server/...` — 0 issues
- [ ] `gofmt -l cmd/server/` — пустой вывод
- [ ] `goimports -l -local github.com/dovgalb/project-rupor cmd/server/` — пустой вывод
- [ ] **Smoke 1 (config-fail).** Запуск `unset DATABASE_URL JWT_SECRET; go run ./cmd/server` — exit 1 за <500мс, в stderr есть `code=CONFIG-001` или `code=CONFIG-002` (порядок проверки JWT первый, поэтому 001).
- [ ] **Smoke 2 (port valid).** `JWT_SECRET=x DATABASE_URL=postgres://... SERVER_PORT=8081 make run` — сервер слушает на :8081, в логе `server starting addr=:8081`.
- [ ] **Smoke 3 (health).** При запущенном сервере `curl -i http://localhost:$PORT/api/v1/health` — `HTTP/1.1 200 OK`, `Content-Type: application/json; charset=utf-8`, тело `{"status":"ok"}`.
- [ ] **Smoke 4 (404).** `curl -i http://localhost:$PORT/api/v1/unknown` — `HTTP/1.1 404 Not Found`.
- [ ] **Smoke 5 (graceful shutdown).** Запустить сервер, в другом терминале `kill -INT $(pgrep -f 'cmd/server')`. В лог пишется `shutdown signal received` затем `shutdown complete`, exit 0, общее время <1с (без активных запросов).
- [ ] **Smoke 6 (graceful timeout).** Запустить сервер с health-handler, который `time.Sleep(10s)` (или другим способом эмулировать длинный запрос — этот шаг manual). Послать `kill -TERM` во время запроса. Либо запрос дотягивает за 5с, либо `Shutdown` возвращает `context.DeadlineExceeded` и exit 1. (Опционально, можно пропустить, если health быстрый.)
- [ ] `cmd/server/` импорты: только stdlib + `github.com/go-chi/chi/v5` + `github.com/dovgalb/project-rupor/config`. Проверить: `go list -f '{{.Imports}}' ./cmd/server`.
- [ ] Coverage: `go test ./cmd/server -cover` — на одном тесте ≥30% (`healthHandler` покрыт целиком, `main` и `run` — нет, и это нормально для composition root).

## Что НЕ делает эта фаза

- Не пишет тест на `run()` или `main()` — composition root тестируется через smoke (manual или integration), а не unit (`prompts/Tests Style.txt:120-129`). Опциональный integration-тест `TestServer_GracefulShutdown` под `//go:build integration` — будущая работа.
- Не подключает middleware (`chi/middleware.Logger`, `chi/middleware.Recoverer`) — Q9 в `../03-decisions.md` решит это в Phase 5 или позже. На фазе 1.1 access-логи и recoverer не критичны.
- Не делает access-log — пишется только `server starting` / `shutdown ...`. Запросы не логируются. Будущая работа (Q9).
- Не открывает соединение с PostgreSQL — `cfg.DatabaseURL()` загружен, но не используется. Это норма для Phase 1.1 (см. ADR-006).
- Не добавляет `/livez` / `/readyz` — единственный health-endpoint `/api/v1/health` (см. ADR-006 Open Question Q-readyz, не оформлен).
