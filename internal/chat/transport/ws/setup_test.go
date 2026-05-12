package wschat_test

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sync"
	"testing"
	"time"

	ws "github.com/coder/websocket"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	authdom "github.com/dovgalb/project-rupor/internal/auth/domain"
	jwtrepo "github.com/dovgalb/project-rupor/internal/auth/repository/jwt"
	"github.com/dovgalb/project-rupor/internal/chat/domain"
	wschat "github.com/dovgalb/project-rupor/internal/chat/transport/ws"
	"github.com/dovgalb/project-rupor/internal/chat/usecase"
	pws "github.com/dovgalb/project-rupor/pkg/websocket"
)

const testAccessTTL = 15 * time.Minute

var testJWTSecret = []byte("test-secret-1234567890")

// ---------- helpers ----------

func silentLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{Level: slog.LevelError}))
}

type realClock struct{ now time.Time }

func (c *realClock) Now() time.Time { return c.now }

type seqUUID struct {
	mu  sync.Mutex
	nxt []uuid.UUID
}

func (g *seqUUID) New() uuid.UUID {
	g.mu.Lock()
	defer g.mu.Unlock()
	if len(g.nxt) > 0 {
		id := g.nxt[0]
		g.nxt = g.nxt[1:]
		return id
	}
	return uuid.New()
}

// ---------- fakeMessageRepo ----------

type fakeMessageRepo struct {
	mu          sync.Mutex
	saved       []*domain.Message
	channelInfo map[uuid.UUID]usecase.ChannelInfo
}

func newFakeMessageRepo() *fakeMessageRepo {
	return &fakeMessageRepo{channelInfo: map[uuid.UUID]usecase.ChannelInfo{}}
}

func (r *fakeMessageRepo) Save(_ context.Context, m *domain.Message) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.saved = append(r.saved, m)
	return nil
}

func (r *fakeMessageRepo) ListByChannel(_ context.Context, _ domain.ChannelID, _ domain.MessageID, _ int) ([]*domain.Message, error) {
	return nil, nil
}

func (r *fakeMessageRepo) ChannelOf(_ context.Context, channelID domain.ChannelID) (usecase.ChannelInfo, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	info, ok := r.channelInfo[channelID.UUID()]
	if !ok {
		return usecase.ChannelInfo{}, domain.ErrChannelNotFound
	}
	return info, nil
}

func (r *fakeMessageRepo) withChannel(channelID, roomID uuid.UUID, kind string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	chID, _ := domain.NewChannelID(channelID)
	rmID, _ := domain.NewRoomID(roomID)
	r.channelInfo[channelID] = usecase.ChannelInfo{ChannelID: chID, RoomID: rmID, Kind: kind}
}

// ---------- fakeMembershipQuery ----------

type mqKey struct {
	channel uuid.UUID
	user    uuid.UUID
}

type fakeMembershipQuery struct {
	mu      sync.Mutex
	allowed map[mqKey]bool
}

func newFakeMembershipQuery() *fakeMembershipQuery {
	return &fakeMembershipQuery{allowed: map[mqKey]bool{}}
}

func (m *fakeMembershipQuery) allow(channelID, userID uuid.UUID) *fakeMembershipQuery {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.allowed[mqKey{channel: channelID, user: userID}] = true
	return m
}

func (m *fakeMembershipQuery) Require(_ context.Context, channelID domain.ChannelID, userID domain.UserID, _ usecase.RoleRequirement) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.allowed[mqKey{channel: channelID.UUID(), user: userID.UUID()}] {
		return nil
	}
	return domain.ErrChatAccessDenied
}

// ---------- fakeMembershipReader (room list) ----------

type fakeMembershipReader struct {
	mu        sync.Mutex
	roomsByID map[uuid.UUID][]uuid.UUID
	err       error
}

func newFakeMembershipReader() *fakeMembershipReader {
	return &fakeMembershipReader{roomsByID: map[uuid.UUID][]uuid.UUID{}}
}

func (r *fakeMembershipReader) ListRoomIDsByUser(_ context.Context, userID uuid.UUID) ([]uuid.UUID, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.err != nil {
		return nil, r.err
	}
	return append([]uuid.UUID(nil), r.roomsByID[userID]...), nil
}

// ---------- fakeBroadcaster (для SendMessage usecase) ----------

type fakeBroadcaster struct {
	mu     sync.Mutex
	hub    *pws.Hub
	events []string
}

func newFakeBroadcaster(hub *pws.Hub) *fakeBroadcaster {
	return &fakeBroadcaster{hub: hub}
}

func (b *fakeBroadcaster) PublishToChannel(channelID domain.ChannelID, eventType string, payload any) {
	b.mu.Lock()
	b.events = append(b.events, eventType+":channel:"+channelID.String())
	b.mu.Unlock()
	b.hub.Publish(context.Background(), pws.ChannelTopic(channelID.UUID()), eventType, payload)
}

func (b *fakeBroadcaster) PublishToRoom(roomID domain.RoomID, eventType string, payload any) {
	b.mu.Lock()
	b.events = append(b.events, eventType+":room:"+roomID.String())
	b.mu.Unlock()
	b.hub.Publish(context.Background(), pws.RoomTopic(roomID.UUID()), eventType, payload)
}

// ---------- testEnv ----------

type testEnv struct {
	server     *httptest.Server
	hub        *pws.Hub
	messages   *fakeMessageRepo
	membership *fakeMembershipQuery
	rooms      *fakeMembershipReader
	clock      *realClock
	uuids      *seqUUID
	issuer     *jwtrepo.TokenIssuer
}

func setupServer(t *testing.T) *testEnv {
	t.Helper()
	now := time.Date(2026, 5, 12, 12, 0, 0, 0, time.UTC)
	clock := &realClock{now: now}
	uuids := &seqUUID{}
	issuer := jwtrepo.NewTokenIssuer(testJWTSecret, testAccessTTL)
	hub := pws.NewHub(silentLogger())
	messages := newFakeMessageRepo()
	membership := newFakeMembershipQuery()
	rooms := newFakeMembershipReader()
	broadcaster := newFakeBroadcaster(hub)

	sendMessage := usecase.NewSendMessage(messages, membership, broadcaster, clock, uuids)

	r := chi.NewRouter()
	wschat.RegisterWSRoute(r, wschat.WSDeps{
		Hub:                hub,
		SendMessage:        sendMessage,
		MembershipForChat:  membership,
		MembershipForRooms: rooms,
		TokenIssuer:        issuer,
		Clock:              clock,
	})

	srv := httptest.NewServer(r)
	t.Cleanup(func() {
		hub.Shutdown(context.Background())
		srv.Close()
	})

	return &testEnv{
		server: srv, hub: hub, messages: messages, membership: membership,
		rooms: rooms, clock: clock, uuids: uuids, issuer: issuer,
	}
}

func issueTestToken(t *testing.T, env *testEnv, userID uuid.UUID) string {
	t.Helper()
	uid, err := authdom.NewUserID(userID)
	if err != nil {
		t.Fatalf("auth NewUserID: %v", err)
	}
	tok, _, err := env.issuer.IssueAccess(uid, env.clock.now)
	if err != nil {
		t.Fatalf("IssueAccess: %v", err)
	}
	return tok
}

func issueExpiredToken(t *testing.T, env *testEnv, userID uuid.UUID) string {
	t.Helper()
	uid, err := authdom.NewUserID(userID)
	if err != nil {
		t.Fatalf("auth NewUserID: %v", err)
	}
	pastIssuer := jwtrepo.NewTokenIssuer(testJWTSecret, time.Nanosecond)
	tok, _, err := pastIssuer.IssueAccess(uid, env.clock.now.Add(-time.Hour))
	if err != nil {
		t.Fatalf("IssueAccess: %v", err)
	}
	return tok
}

func dialWS(t *testing.T, env *testEnv, ctx context.Context, token string) (*ws.Conn, *http.Response) {
	t.Helper()
	u, err := url.Parse(env.server.URL)
	if err != nil {
		t.Fatalf("parse url: %v", err)
	}
	u.Scheme = "ws"
	u.Path = "/ws"
	if token != "" {
		q := u.Query()
		q.Set("token", token)
		u.RawQuery = q.Encode()
	}
	c, resp, err := ws.Dial(ctx, u.String(), nil)
	if err != nil {
		// в случае 401 ws.Dial вернёт err но и resp с status'ом.
		return nil, resp
	}
	c.SetReadLimit(64 * 1024)
	t.Cleanup(func() { _ = c.CloseNow() })
	return c, resp
}

func readJSON(t *testing.T, ctx context.Context, c *ws.Conn) map[string]any {
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

func writeJSON(t *testing.T, ctx context.Context, c *ws.Conn, v any) {
	t.Helper()
	data, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if err := c.Write(ctx, ws.MessageText, data); err != nil {
		t.Fatalf("write: %v", err)
	}
}
