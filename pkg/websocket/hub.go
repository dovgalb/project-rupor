package websocket

import (
	"context"
	"encoding/json"
	"log/slog"
	"sync"

	ws "github.com/coder/websocket"
)

type Hub struct {
	logger *slog.Logger

	mu          sync.RWMutex
	conns       map[ConnID]*Conn
	subscribers map[string]map[ConnID]*Conn

	shutdownOnce sync.Once
	shutdownCh   chan struct{}
}

func NewHub(logger *slog.Logger) *Hub {
	if logger == nil {
		logger = slog.Default()
	}
	return &Hub{
		logger:      logger,
		conns:       map[ConnID]*Conn{},
		subscribers: map[string]map[ConnID]*Conn{},
		shutdownCh:  make(chan struct{}),
	}
}

// Register регистрирует Conn и возвращает идемпотентный cleanup для disconnect.
func (h *Hub) Register(c *Conn) (cleanup func()) {
	h.mu.Lock()
	h.conns[c.ID()] = c
	h.mu.Unlock()

	var once sync.Once
	return func() {
		once.Do(func() {
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
		})
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

// Publish сериализует payload один раз и рассылает подписчикам topic'а.
// Ошибки доставки логируются, рассылка остальным не прерывается.
// Жизненный цикл доставки отвязан от ctx вызывающего: hub берёт на себя
// ответственность за best-effort delivery, единственный таймаут — per-write
// 5s внутри Conn.writeRaw. Отмена ctx у publisher'а (например, HTTP-request
// ended) не должна обрывать рассылку остальным.
// MVP-ограничение: цикл рассылки последовательный. Slow consumer может
// задержать соседей по топику до writeTimeout каждого. TODO(phase-7+): per-conn
// outbound queue с bounded backpressure либо goroutine per write.
func (h *Hub) Publish(_ context.Context, t Topic, eventType string, payload any) {
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

	h.logger.Debug("hub: publish",
		slog.String("topic", t.Key()),
		slog.String("type", eventType),
		slog.Int("subscribers", len(subs)),
	)

	for _, c := range subs {
		// Используем фоновой ctx — write имеет собственный timeout внутри.
		if err := c.writeRaw(context.Background(), data); err != nil {
			h.logger.Warn("hub: write to conn failed",
				slog.String("conn_id", c.ID().String()),
				slog.Any("err", err),
			)
		}
	}
}

// Shutdown закрывает все подключения с close-code 1001. Идемпотентен.
// Close-handshake идёт параллельно по всем conn'ам. Ожидание ограничено
// дедлайном ctx: если ctx отменился раньше, чем все close завершились,
// функция возвращается, оставив зависшие close-горутины в фоне.
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

		done := make(chan struct{})
		var wg sync.WaitGroup
		for _, c := range conns {
			wg.Add(1)
			go func(c *Conn) {
				defer wg.Done()
				c.Close(ws.StatusGoingAway, "going away")
			}(c)
		}
		go func() {
			wg.Wait()
			close(done)
		}()

		select {
		case <-done:
		case <-ctx.Done():
			h.logger.Warn("hub: shutdown deadline exceeded, returning with close in background",
				slog.Any("err", ctx.Err()),
			)
		}
	})
}
