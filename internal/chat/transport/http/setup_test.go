package httpchat_test

import (
	"context"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	authdom "github.com/dovgalb/project-rupor/internal/auth/domain"
	jwtrepo "github.com/dovgalb/project-rupor/internal/auth/repository/jwt"
	"github.com/dovgalb/project-rupor/internal/chat/domain"
	httpchat "github.com/dovgalb/project-rupor/internal/chat/transport/http"
	"github.com/dovgalb/project-rupor/internal/chat/usecase"
)

const testAccessTTL = 15 * time.Minute

var testJWTSecret = []byte("test-secret-1234567890")

// ---------- helpers ----------

func mustMessageID(t *testing.T, raw uuid.UUID) domain.MessageID {
	t.Helper()
	id, err := domain.NewMessageID(raw)
	if err != nil {
		t.Fatalf("mustMessageID: %v", err)
	}
	return id
}

func mustChannelID(t *testing.T, raw uuid.UUID) domain.ChannelID {
	t.Helper()
	id, err := domain.NewChannelID(raw)
	if err != nil {
		t.Fatalf("mustChannelID: %v", err)
	}
	return id
}

func mustUserID(t *testing.T, raw uuid.UUID) domain.UserID {
	t.Helper()
	id, err := domain.NewUserID(raw)
	if err != nil {
		t.Fatalf("mustUserID: %v", err)
	}
	return id
}

func mustMessageText(t *testing.T, raw string) domain.MessageText {
	t.Helper()
	v, err := domain.NewMessageText(raw)
	if err != nil {
		t.Fatalf("mustMessageText: %v", err)
	}
	return v
}

func mustMessage(t *testing.T, id, channelID, authorID uuid.UUID, text string, createdAt time.Time) *domain.Message {
	t.Helper()
	m, err := domain.NewMessage(
		mustMessageID(t, id),
		mustChannelID(t, channelID),
		mustUserID(t, authorID),
		mustMessageText(t, text),
		createdAt,
	)
	if err != nil {
		t.Fatalf("mustMessage: %v", err)
	}
	return m
}

// ---------- realClock ----------

type realClock struct{ now time.Time }

func (c *realClock) Now() time.Time { return c.now }

// ---------- fakeMessageRepo ----------

type fakeMessageRepo struct {
	mu          sync.Mutex
	listByChan  map[uuid.UUID][]*domain.Message
	channelInfo map[uuid.UUID]usecase.ChannelInfo
}

func newFakeMessageRepo() *fakeMessageRepo {
	return &fakeMessageRepo{
		listByChan:  map[uuid.UUID][]*domain.Message{},
		channelInfo: map[uuid.UUID]usecase.ChannelInfo{},
	}
}

func (r *fakeMessageRepo) Save(_ context.Context, m *domain.Message) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.listByChan[m.ChannelID().UUID()] = append(r.listByChan[m.ChannelID().UUID()], m)
	return nil
}

func (r *fakeMessageRepo) ListByChannel(_ context.Context, channelID domain.ChannelID, before domain.MessageID, limit int) ([]*domain.Message, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	items := r.listByChan[channelID.UUID()]
	if !before.IsZero() {
		var filtered []*domain.Message
		seen := false
		for _, m := range items {
			if seen {
				filtered = append(filtered, m)
				continue
			}
			if m.ID() == before {
				seen = true
			}
		}
		items = filtered
	}
	if len(items) > limit {
		items = items[:limit]
	}
	return items, nil
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
	r.channelInfo[channelID] = usecase.ChannelInfo{
		ChannelID: chID,
		RoomID:    rmID,
		Kind:      kind,
	}
}

func (r *fakeMessageRepo) put(m *domain.Message) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.listByChan[m.ChannelID().UUID()] = append(r.listByChan[m.ChannelID().UUID()], m)
}

// ---------- fakeMembershipQuery ----------

type mqKey struct {
	channel uuid.UUID
	user    uuid.UUID
}

type fakeMembershipQuery struct {
	mu      sync.Mutex
	allowed map[mqKey]bool
	unknown map[uuid.UUID]bool // channels помеченные как not found
}

func newFakeMembershipQuery() *fakeMembershipQuery {
	return &fakeMembershipQuery{
		allowed: map[mqKey]bool{},
		unknown: map[uuid.UUID]bool{},
	}
}

func (m *fakeMembershipQuery) allow(channelID, userID uuid.UUID) *fakeMembershipQuery {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.allowed[mqKey{channel: channelID, user: userID}] = true
	return m
}

func (m *fakeMembershipQuery) markChannelUnknown(channelID uuid.UUID) *fakeMembershipQuery {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.unknown[channelID] = true
	return m
}

func (m *fakeMembershipQuery) Require(_ context.Context, channelID domain.ChannelID, userID domain.UserID, _ usecase.RoleRequirement) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.unknown[channelID.UUID()] {
		return domain.ErrChannelNotFound
	}
	if m.allowed[mqKey{channel: channelID.UUID(), user: userID.UUID()}] {
		return nil
	}
	return domain.ErrChatAccessDenied
}

// ---------- testEnv ----------

type testEnv struct {
	server     *httptest.Server
	messages   *fakeMessageRepo
	membership *fakeMembershipQuery
	clock      *realClock
	issuer     *jwtrepo.TokenIssuer
}

func setupServer(t *testing.T) *testEnv {
	t.Helper()
	now := time.Date(2026, 5, 12, 12, 0, 0, 0, time.UTC)
	clock := &realClock{now: now}
	issuer := jwtrepo.NewTokenIssuer(testJWTSecret, testAccessTTL)

	messages := newFakeMessageRepo()
	membership := newFakeMembershipQuery()

	listMessages := usecase.NewListMessages(messages, membership)

	r := chi.NewRouter()
	httpchat.RegisterRoutes(r, httpchat.Deps{
		ListMessages: listMessages,
		TokenIssuer:  issuer,
		Clock:        clock,
	})

	srv := httptest.NewServer(r)
	t.Cleanup(srv.Close)

	return &testEnv{
		server: srv, messages: messages, membership: membership,
		clock: clock, issuer: issuer,
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
	// Эмитируем токен «в прошлом» с короткой TTL.
	pastIssuer := jwtrepo.NewTokenIssuer(testJWTSecret, time.Nanosecond)
	tok, _, err := pastIssuer.IssueAccess(uid, env.clock.now.Add(-time.Hour))
	if err != nil {
		t.Fatalf("IssueAccess: %v", err)
	}
	return tok
}

type errResp struct {
	Error struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}
