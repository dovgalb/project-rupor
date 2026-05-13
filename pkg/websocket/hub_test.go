package websocket_test

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	ws "github.com/coder/websocket"
	"github.com/google/uuid"

	pkgws "github.com/dovgalb/project-rupor/pkg/websocket"
)

// silentLogger подавляет логирование во время тестов.
func silentLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{Level: slog.LevelError}))
}

// testHarness — окружение для одного теста: hub + httptest-сервер + канал акцептов.
type testHarness struct {
	t       *testing.T
	hub     *pkgws.Hub
	server  *httptest.Server
	accepts chan accept
}

type accept struct {
	conn    *pkgws.Conn
	cleanup func()
	rawCtx  context.Context
}

// newTestHarness поднимает сервер, который на каждый /ws запрос делает Upgrade,
// регистрирует Conn в hub и отдаёт его тесту через канал accepts.
func newTestHarness(t *testing.T) *testHarness {
	t.Helper()
	h := &testHarness{
		t:       t,
		hub:     pkgws.NewHub(silentLogger()),
		accepts: make(chan accept, 32),
	}
	handler := http.NewServeMux()
	handler.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		raw, err := ws.Accept(w, r, &ws.AcceptOptions{InsecureSkipVerify: true})
		if err != nil {
			return
		}
		raw.SetReadLimit(64 * 1024)
		conn := pkgws.NewConn(uuid.New(), raw)
		cleanup := h.hub.Register(conn)
		ctx, cancel := context.WithCancel(r.Context())
		// read-loop: держит conn живым и реагирует на close-фрейм клиента.
		go func() {
			defer cancel()
			for {
				if _, _, err := raw.Read(ctx); err != nil {
					return
				}
			}
		}()
		h.accepts <- accept{conn: conn, cleanup: cleanup, rawCtx: ctx}
		<-ctx.Done()
		cleanup()
	})
	h.server = httptest.NewServer(handler)
	t.Cleanup(func() {
		h.hub.Shutdown(context.Background())
		h.server.Close()
		close(h.accepts)
	})
	return h
}

// dial подключается клиентом и возвращает accept (для server-side операций) и client-conn.
func (h *testHarness) dial(ctx context.Context) (accept, *ws.Conn) {
	h.t.Helper()
	u, err := url.Parse(h.server.URL)
	if err != nil {
		h.t.Fatalf("parse url: %v", err)
	}
	u.Scheme = "ws"
	u.Path = "/ws"
	c, _, err := ws.Dial(ctx, u.String(), nil)
	if err != nil {
		h.t.Fatalf("dial: %v", err)
	}
	c.SetReadLimit(64 * 1024)
	a := <-h.accepts
	h.t.Cleanup(func() { _ = c.CloseNow() })
	return a, c
}

// readFrame читает один JSON-фрейм с client-стороны и парсит в map.
func readFrame(t *testing.T, ctx context.Context, c *ws.Conn) map[string]any {
	t.Helper()
	_, data, err := c.Read(ctx)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	var v map[string]any
	if err := json.Unmarshal(data, &v); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	return v
}

func TestHub_PublishToChannel_DeliversToSubscribers(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	h := newTestHarness(t)
	a, c := h.dial(ctx)

	topic := pkgws.ChannelTopic(uuid.New())
	h.hub.Subscribe(a.conn, topic)

	h.hub.Publish(ctx, topic, "message.new", map[string]any{"id": "m-1"})

	frame := readFrame(t, ctx, c)
	if frame["type"] != "message.new" {
		t.Fatalf("type = %v, want message.new", frame["type"])
	}
	data, ok := frame["data"].(map[string]any)
	if !ok || data["id"] != "m-1" {
		t.Fatalf("data = %v, want {id: m-1}", frame["data"])
	}
}

func TestHub_MultipleSubscribers_ReceiveSameFrame(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	h := newTestHarness(t)
	a1, c1 := h.dial(ctx)
	a2, c2 := h.dial(ctx)

	topic := pkgws.ChannelTopic(uuid.New())
	h.hub.Subscribe(a1.conn, topic)
	h.hub.Subscribe(a2.conn, topic)

	h.hub.Publish(ctx, topic, "evt", map[string]any{"n": float64(1)})

	for i, c := range []*ws.Conn{c1, c2} {
		frame := readFrame(t, ctx, c)
		if frame["type"] != "evt" {
			t.Fatalf("conn %d: type = %v", i, frame["type"])
		}
	}
}

func TestHub_PublishToTopic_DoesNotLeakToOtherTopics(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	h := newTestHarness(t)
	a1, c1 := h.dial(ctx)
	a2, c2 := h.dial(ctx)

	topicA := pkgws.ChannelTopic(uuid.New())
	topicB := pkgws.ChannelTopic(uuid.New())
	h.hub.Subscribe(a1.conn, topicA)
	h.hub.Subscribe(a2.conn, topicB)

	h.hub.Publish(ctx, topicA, "for-a", nil)

	frame := readFrame(t, ctx, c1)
	if frame["type"] != "for-a" {
		t.Fatalf("c1: type = %v", frame["type"])
	}

	// c2 не должен получить ничего за короткий read-deadline.
	shortCtx, cancel2 := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel2()
	_, _, err := c2.Read(shortCtx)
	if err == nil {
		t.Fatal("c2: ожидали отсутствие фрейма (read timeout), но фрейм пришёл")
	}
}

func TestHub_Disconnect_RemovesSubscriptions(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	h := newTestHarness(t)
	a1, c1 := h.dial(ctx)
	a2, c2 := h.dial(ctx)

	topic := pkgws.ChannelTopic(uuid.New())
	h.hub.Subscribe(a1.conn, topic)
	h.hub.Subscribe(a2.conn, topic)

	// Закрываем первого клиента и ждём, пока server-side read-loop
	// в test-harness увидит close и вызовет cleanup, который снимет подписку.
	// Probe: read c1 должен вернуть не-DeadlineExceeded ошибку (соединение мертво).
	_ = c1.Close(ws.StatusNormalClosure, "leaving")
	_ = a1

	waitUntil(t, 2*time.Second, func() bool {
		probeCtx, probeCancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
		defer probeCancel()
		_, _, err := c1.Read(probeCtx)
		return err != nil && !errors.Is(err, context.DeadlineExceeded)
	})

	h.hub.Publish(ctx, topic, "after-disconnect", nil)

	frame := readFrame(t, ctx, c2)
	if frame["type"] != "after-disconnect" {
		t.Fatalf("c2: type = %v", frame["type"])
	}
}

func TestHub_DoubleSubscribe_NoDuplicateDelivery(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	h := newTestHarness(t)
	a, c := h.dial(ctx)

	topic := pkgws.ChannelTopic(uuid.New())
	h.hub.Subscribe(a.conn, topic)
	h.hub.Subscribe(a.conn, topic)

	h.hub.Publish(ctx, topic, "once", nil)

	frame := readFrame(t, ctx, c)
	if frame["type"] != "once" {
		t.Fatalf("type = %v", frame["type"])
	}

	// Второго фрейма быть не должно.
	shortCtx, cancel2 := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel2()
	_, _, err := c.Read(shortCtx)
	if err == nil {
		t.Fatal("ожидали один фрейм, пришёл второй")
	}
}

func TestHub_Shutdown_ClosesAllConns(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	h := newTestHarness(t)
	_, c1 := h.dial(ctx)
	_, c2 := h.dial(ctx)

	// Стартуем read до Shutdown — иначе у клиента нет reader'а,
	// и close-handshake nhooyr на server-стороне зависнет до таймаута.
	type readResult struct {
		err error
	}
	results := make(chan readResult, 2)
	for _, c := range []*ws.Conn{c1, c2} {
		c := c
		go func() {
			_, _, err := c.Read(ctx)
			results <- readResult{err: err}
		}()
	}

	h.hub.Shutdown(ctx)

	for i := 0; i < 2; i++ {
		select {
		case r := <-results:
			if r.err == nil {
				t.Fatalf("conn %d: ожидали закрытие, получили nil", i)
			}
			if got := ws.CloseStatus(r.err); got != ws.StatusGoingAway {
				t.Fatalf("conn %d: status = %v, want StatusGoingAway (err=%v)", i, got, r.err)
			}
		case <-time.After(3 * time.Second):
			t.Fatal("read не завершился за 3 секунды")
		}
	}
}

func TestHub_ConcurrentSubscribePublish_NoRace(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	h := newTestHarness(t)
	a, c := h.dial(ctx)

	// drain reader, чтобы не блокировать рассылку.
	var received int64
	done := make(chan struct{})
	go func() {
		for {
			_, _, err := c.Read(ctx)
			if err != nil {
				close(done)
				return
			}
			atomic.AddInt64(&received, 1)
		}
	}()

	var wg sync.WaitGroup
	topics := make([]pkgws.Topic, 8)
	for i := range topics {
		topics[i] = pkgws.ChannelTopic(uuid.New())
	}
	for i := 0; i < 8; i++ {
		wg.Add(2)
		go func(idx int) {
			defer wg.Done()
			for j := 0; j < 50; j++ {
				h.hub.Subscribe(a.conn, topics[idx])
			}
		}(i)
		go func(idx int) {
			defer wg.Done()
			for j := 0; j < 50; j++ {
				h.hub.Publish(ctx, topics[idx], "evt", nil)
			}
		}(i)
	}
	wg.Wait()
	// Достаточно отсутствия гонки. Конкретное значение received зависит от тайминга.
	_ = atomic.LoadInt64(&received)
}

func TestHub_SlowClient_DoesNotBlockOtherDelivery(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()

	h := newTestHarness(t)
	aSlow, cSlow := h.dial(ctx)
	aFast, cFast := h.dial(ctx)

	topic := pkgws.ChannelTopic(uuid.New())
	h.hub.Subscribe(aSlow.conn, topic)
	h.hub.Subscribe(aFast.conn, topic)

	// Fast-читатель в фоне.
	got := make(chan string, 1)
	go func() {
		frame := readFrame(t, ctx, cFast)
		s, _ := frame["type"].(string)
		got <- s
	}()

	// Slow-читатель не читает; забьём его буфер большим payload до тех пор,
	// пока write не зафейлится по таймауту (5s, см. writeTimeout).
	big := strings.Repeat("x", 32*1024)
	publishCtx, publishCancel := context.WithTimeout(ctx, 7*time.Second)
	defer publishCancel()
	for i := 0; i < 200; i++ {
		h.hub.Publish(publishCtx, topic, "evt", big)
	}

	select {
	case s := <-got:
		if s != "evt" {
			t.Fatalf("fast: got type=%q", s)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("fast client не получил ни одного фрейма")
	}

	_ = cSlow
}

func waitUntil(t *testing.T, max time.Duration, pred func() bool) {
	t.Helper()
	deadline := time.Now().Add(max)
	for time.Now().Before(deadline) {
		if pred() {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
}
