---
phase: 3
name: pkg-websocket
layer: infrastructure
depends_on: none
plan: ./README.md
---

# Phase 3: `pkg/websocket/` — Hub, Conn, Topic, Upgrade

## Цель

Создать доменно-агностичный WebSocket-инфраструктурный пакет: hub с подписками и broadcast'ом, обёртку над nhooyr conn, типизированный topic, тонкий wrapper для handshake. Полное покрытие тестами hub'а, включая race-detector.

## Контекст

После этой фазы `pkg/websocket/` готов как библиотека. Зависит ТОЛЬКО от stdlib, `nhooyr.io/websocket`, `github.com/google/uuid`. Не импортирует `internal/*` — это правило в [01-architecture.md §Граф зависимостей](../01-architecture.md) и проверяется `arch_test.go` (добавляется в phase-09).

Сейчас `pkg/websocket/` содержит только `.gitkeep`. В `go.mod` нет `nhooyr.io/websocket` — он добавится в этой фазе. Это согласованное решение ([§ D-12](../03-decisions.md)).

## Файлы для создания

### `pkg/websocket/topic.go`

**Назначение:** Типизированный ключ подписки. Hub оперирует `Topic`, а не голыми строками.

**Детали реализации:**
```go
package websocket

import (
    "fmt"

    "github.com/google/uuid"
)

type TopicKind uint8

const (
    TopicKindUnknown TopicKind = iota
    TopicKindChannel
    TopicKindRoom
)

type Topic struct {
    kind TopicKind
    id   uuid.UUID
}

func ChannelTopic(channelID uuid.UUID) Topic { return Topic{TopicKindChannel, channelID} }
func RoomTopic(roomID uuid.UUID) Topic       { return Topic{TopicKindRoom, roomID} }

func (t Topic) Kind() TopicKind { return t.kind }
func (t Topic) ID() uuid.UUID   { return t.id }

func (t Topic) Key() string {
    switch t.kind {
    case TopicKindChannel: return "channel:" + t.id.String()
    case TopicKindRoom:    return "room:"    + t.id.String()
    default:               return fmt.Sprintf("unknown:%s", t.id)
    }
}
```

### `pkg/websocket/conn.go`

**Назначение:** Обёртка над `*websocket.Conn` из nhooyr. Сериализует write'ы через mutex, хранит userID, предоставляет идемпотентный Close.

**Детали реализации:**
```go
package websocket

import (
    "context"
    "encoding/json"
    "sync"
    "time"

    "github.com/google/uuid"
    ws "nhooyr.io/websocket"
)

type ConnID = uuid.UUID

type Conn struct {
    id       ConnID
    userID   uuid.UUID
    ws       *ws.Conn
    writeMu  sync.Mutex
    closeOnce sync.Once
}

func newConn(userID uuid.UUID, raw *ws.Conn) *Conn {
    return &Conn{id: uuid.New(), userID: userID, ws: raw}
}

func (c *Conn) ID() ConnID         { return c.id }
func (c *Conn) UserID() uuid.UUID  { return c.userID }

// WriteJSON сериализует payload и пишет фрейм. Сериализован под mutex'ом.
func (c *Conn) WriteJSON(ctx context.Context, v any) error {
    c.writeMu.Lock()
    defer c.writeMu.Unlock()
    ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
    defer cancel()
    data, err := json.Marshal(v)
    if err != nil {
        return err
    }
    return c.ws.Write(ctx, ws.MessageText, data)
}

// Read читает один JSON-фрейм. Должен вызываться только из owner-горутины.
func (c *Conn) ReadJSON(ctx context.Context, v any) error {
    _, data, err := c.ws.Read(ctx)
    if err != nil {
        return err
    }
    return json.Unmarshal(data, v)
}

// Close идемпотентен.
func (c *Conn) Close(code ws.StatusCode, reason string) {
    c.closeOnce.Do(func() {
        _ = c.ws.Close(code, reason)
    })
}
```

**Замечания:**
- `WriteJSON` под mutex — [§ D-03](../03-decisions.md).
- `Close` через `sync.Once` — повторный вызов из disconnect-cleanup и shutdown'а безопасен.
- `Read` НЕ под mutex — nhooyr допускает один читатель + один писатель параллельно.

### `pkg/websocket/upgrade.go`

**Назначение:** Тонкий wrapper над `websocket.Accept`. Инкапсулирует accept-опции (OriginPatterns из CORS-config, без InsecureSkipVerify).

**Детали реализации:**
```go
package websocket

import (
    "net/http"

    ws "nhooyr.io/websocket"
)

type UpgradeOptions struct {
    OriginPatterns []string  // из cfg.CORSAllowedOrigins() (без схемы; nhooyr ждёт host:port)
    MaxFrameBytes  int       // default 65536 (64 KB)
}

func Upgrade(w http.ResponseWriter, r *http.Request, opts UpgradeOptions) (*ws.Conn, error) {
    accept := &ws.AcceptOptions{
        OriginPatterns:     opts.OriginPatterns,
        InsecureSkipVerify: false,
    }
    conn, err := ws.Accept(w, r, accept)
    if err != nil {
        return nil, err
    }
    limit := opts.MaxFrameBytes
    if limit == 0 {
        limit = 65536
    }
    conn.SetReadLimit(int64(limit))
    return conn, nil
}
```

**Замечания:**
- OriginPatterns — список pattern'ов в формате nhooyr (`example.com`, `*.example.com`). На уровне adapter'а в `cmd/server/main.go` конвертация из `https://app.example.com` → `app.example.com` (отрезаем scheme).
- ReadLimit 64 KB — [§ Размеры и таймауты](../08-api-contract.md).

### `pkg/websocket/hub.go`

**Назначение:** Центральная структура для регистрации подключений, подписок и broadcast'а.

**Детали реализации:**

```go
package websocket

import (
    "context"
    "encoding/json"
    "log/slog"
    "sync"

    ws "nhooyr.io/websocket"
)

type Hub struct {
    logger *slog.Logger

    mu          sync.RWMutex
    conns       map[ConnID]*Conn
    subscribers map[string]map[ConnID]*Conn  // topic.Key() → conns

    shutdownOnce sync.Once
    shutdownCh   chan struct{}
}

func NewHub(logger *slog.Logger) *Hub {
    return &Hub{
        logger:      logger,
        conns:       map[ConnID]*Conn{},
        subscribers: map[string]map[ConnID]*Conn{},
        shutdownCh:  make(chan struct{}),
    }
}

// Register регистрирует Conn в hub. Возвращает функцию-cleanup для disconnect.
// cleanup идемпотентен.
func (h *Hub) Register(c *Conn) (cleanup func()) {
    h.mu.Lock()
    h.conns[c.ID()] = c
    h.mu.Unlock()

    return func() {
        h.mu.Lock()
        delete(h.conns, c.ID())
        for topicKey, subs := range h.subscribers {
            delete(subs, c.ID())
            if len(subs) == 0 {
                delete(h.subscribers, topicKey)
            }
        }
        h.mu.Unlock()
        c.Close(ws.StatusNormalClosure, "bye")
    }
}

// Subscribe идемпотентно подписывает Conn на topic.
func (h *Hub) Subscribe(c *Conn, t Topic) {
    h.mu.Lock()
    defer h.mu.Unlock()
    key := t.Key()
    subs, ok := h.subscribers[key]
    if !ok {
        subs = map[ConnID]*Conn{}
        h.subscribers[key] = subs
    }
    subs[c.ID()] = c
}

// Unsubscribe снимает подписку. No-op если её не было.
func (h *Hub) Unsubscribe(c *Conn, t Topic) {
    h.mu.Lock()
    defer h.mu.Unlock()
    key := t.Key()
    if subs, ok := h.subscribers[key]; ok {
        delete(subs, c.ID())
        if len(subs) == 0 {
            delete(h.subscribers, key)
        }
    }
}

// Publish сериализует payload один раз и рассылает всем подписчикам topic'а.
// Ошибки записи логируются и не блокируют рассылку остальным.
func (h *Hub) Publish(ctx context.Context, t Topic, eventType string, payload any) {
    select {
    case <-h.shutdownCh:
        return
    default:
    }
    frame := map[string]any{"type": eventType, "data": payload}
    data, err := json.Marshal(frame)
    if err != nil {
        h.logger.Error("hub: marshal frame", slog.Any("err", err))
        return
    }

    h.mu.RLock()
    subs := make([]*Conn, 0, len(h.subscribers[t.Key()]))
    for _, c := range h.subscribers[t.Key()] {
        subs = append(subs, c)
    }
    h.mu.RUnlock()

    h.logger.Info("hub: publish", slog.String("topic", t.Key()), slog.String("type", eventType), slog.Int("subscribers", len(subs)))

    for _, c := range subs {
        if err := c.writeRaw(ctx, data); err != nil {
            h.logger.Warn("hub: write to conn failed",
                slog.String("conn_id", c.ID().String()),
                slog.Any("err", err))
            // best-effort: коннект, возможно, мёртв — disconnect-горутина почистит подписки.
        }
    }
}

// Shutdown закрывает все коннекты с close-code 1001. Идемпотентен.
func (h *Hub) Shutdown(ctx context.Context) {
    h.shutdownOnce.Do(func() {
        close(h.shutdownCh)
        h.mu.Lock()
        conns := make([]*Conn, 0, len(h.conns))
        for _, c := range h.conns {
            conns = append(conns, c)
        }
        h.conns = map[ConnID]*Conn{}
        h.subscribers = map[string]map[ConnID]*Conn{}
        h.mu.Unlock()

        for _, c := range conns {
            c.Close(ws.StatusGoingAway, "going away")
        }
    })
}
```

**Дополнительный метод в `conn.go`:**
```go
// writeRaw используется hub'ом для broadcast'а уже сериализованного фрейма.
func (c *Conn) writeRaw(ctx context.Context, data []byte) error {
    c.writeMu.Lock()
    defer c.writeMu.Unlock()
    ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
    defer cancel()
    return c.ws.Write(ctx, ws.MessageText, data)
}
```

**Замечания:**
- `RWMutex` — публикация под `RLock`, изменения под `Lock`. Подписчики копируются в локальный slice под `RLock`, итерация — без удержания lock'а (см. [§ Risk: гонка subscribe/publish](../03-decisions.md)).
- `Publish` сериализует один раз перед итерацией — экономия CPU.
- Ошибка `writeRaw` логируется WARN, не пробрасывается — best-effort ([§ D-08](../03-decisions.md)).
- `Shutdown` через `sync.Once` — идемпотентен.

## Тесты

`pkg/websocket/hub_test.go` (package `websocket_test`). Все тесты используют `httptest.NewServer` поверх минимального handler'а, который ничего не делает, кроме `Upgrade(w, r)` и регистрации в hub. Клиент — тот же `nhooyr.io/websocket`.

Тесты из [../04-testing.md §Hub](../04-testing.md):
- `TestHub_PublishToChannel_DeliversToSubscribers`
- `TestHub_MultipleSubscribers_ReceiveSameFrame`
- `TestHub_PublishToTopic_DoesNotLeakToOtherTopics`
- `TestHub_Disconnect_RemovesSubscriptions`
- `TestHub_DoubleSubscribe_NoDuplicateDelivery`
- `TestHub_Shutdown_ClosesAllConns`
- `TestHub_ConcurrentSubscribePublish_NoRace` — race detector обязателен
- `TestHub_SlowClient_GetsDisconnected` — slow consumer (намеренная пауза до timeout'а)

`go test -race ./pkg/websocket/...` ОБЯЗАТЕЛЬНО зелёный.

## Файлы для модификации

| Файл | Что меняется |
|---|---|
| `go.mod` | + `require nhooyr.io/websocket v1.8.11` (или последняя стабильная) |
| `go.sum` | автоматически (`go mod tidy`) |

Запустить: `go get nhooyr.io/websocket@latest && go mod tidy`. Затем убедиться, что других новых зависимостей не появилось — если появились, остановиться и проверить (правило `prompts/Go style.txt:111-114`).

## Ключевые решения

- **Hub агностичен к domain** — `Topic` параметризуется через `Kind` + `UUID`, hub не знает ни про channel, ни про room. См. [§ D-02](../03-decisions.md).
- **Sync.Mutex на write в Conn** — [§ D-03](../03-decisions.md).
- **Best-effort delivery** — [§ D-08](../03-decisions.md). Ошибки логируются, не пробрасываются.
- **Sync.Once на Shutdown и Close** — повторный вызов безопасен.

## Verification

- [ ] `go build ./pkg/websocket/...` без ошибок.
- [ ] `go test ./pkg/websocket/...` зелёный.
- [ ] `go test -race ./pkg/websocket/...` зелёный.
- [ ] `pkg/websocket/` не импортирует `internal/*` (визуально проверить imports; arch-тест добавится в phase-09).
- [ ] `go vet ./pkg/websocket/...` чистый.
- [ ] `nhooyr.io/websocket` — единственная новая зависимость в `go.mod` после `go mod tidy`.
- [ ] Все 8 тестов hub'а покрывают сценарии из [../04-testing.md §Hub](../04-testing.md).
