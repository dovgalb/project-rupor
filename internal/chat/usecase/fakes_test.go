package usecase_test

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/dovgalb/project-rupor/internal/chat/domain"
	"github.com/dovgalb/project-rupor/internal/chat/usecase"
)

// ---------- Helpers ----------

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

// ---------- fixedClock / fixedUUID ----------

type fixedClock struct{ now time.Time }

func (c *fixedClock) Now() time.Time { return c.now }

type fixedUUID struct {
	next []uuid.UUID
	idx  int
}

func (g *fixedUUID) New() uuid.UUID {
	id := g.next[g.idx]
	g.idx++
	return id
}

// ---------- fakeMessageRepo ----------

type fakeMessageRepo struct {
	mu           sync.Mutex
	saved        []*domain.Message
	listByChan   map[uuid.UUID][]*domain.Message
	channelInfo  map[uuid.UUID]usecase.ChannelInfo
	saveErr      error
	listErr      error
	channelOfErr error
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
	if r.saveErr != nil {
		return r.saveErr
	}
	r.saved = append(r.saved, m)
	r.listByChan[m.ChannelID().UUID()] = append(r.listByChan[m.ChannelID().UUID()], m)
	return nil
}

func (r *fakeMessageRepo) ListByChannel(_ context.Context, channelID domain.ChannelID, before domain.MessageID, limit int) ([]*domain.Message, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.listErr != nil {
		return nil, r.listErr
	}
	items := r.listByChan[channelID.UUID()]
	if !before.IsZero() {
		// drop everything up to and including `before`
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
	if r.channelOfErr != nil {
		return usecase.ChannelInfo{}, r.channelOfErr
	}
	info, ok := r.channelInfo[channelID.UUID()]
	if !ok {
		return usecase.ChannelInfo{}, domain.ErrChannelNotFound
	}
	return info, nil
}

func (r *fakeMessageRepo) withChannel(channelID uuid.UUID, roomID uuid.UUID, kind string) {
	chID, _ := domain.NewChannelID(channelID)
	rmID, _ := domain.NewRoomID(roomID)
	r.channelInfo[channelID] = usecase.ChannelInfo{
		ChannelID: chID,
		RoomID:    rmID,
		Kind:      kind,
	}
}

// ---------- fakeMembershipQuery ----------

type fakeMembershipQuery struct {
	mu  sync.Mutex
	err error
}

func newFakeMembershipQuery() *fakeMembershipQuery {
	return &fakeMembershipQuery{}
}

func (m *fakeMembershipQuery) withErr(err error) *fakeMembershipQuery {
	m.err = err
	return m
}

func (m *fakeMembershipQuery) Require(_ context.Context, _ domain.ChannelID, _ domain.UserID, _ usecase.RoleRequirement) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.err
}

// ---------- fakeBroadcaster ----------

type publishedEvent struct {
	Kind      string // "channel" | "room"
	TopicID   uuid.UUID
	EventType string
	Payload   any
}

type fakeBroadcaster struct {
	mu        sync.Mutex
	events    []publishedEvent
	panicNext bool
}

func newFakeBroadcaster() *fakeBroadcaster {
	return &fakeBroadcaster{}
}

func (b *fakeBroadcaster) PublishToChannel(channelID domain.ChannelID, eventType string, payload any) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.panicNext {
		b.panicNext = false
		panic(errors.New("forced broadcast panic"))
	}
	b.events = append(b.events, publishedEvent{Kind: "channel", TopicID: channelID.UUID(), EventType: eventType, Payload: payload})
}

func (b *fakeBroadcaster) PublishToRoom(roomID domain.RoomID, eventType string, payload any) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.events = append(b.events, publishedEvent{Kind: "room", TopicID: roomID.UUID(), EventType: eventType, Payload: payload})
}

func (b *fakeBroadcaster) channelEvents() []publishedEvent {
	b.mu.Lock()
	defer b.mu.Unlock()
	var out []publishedEvent
	for _, e := range b.events {
		if e.Kind == "channel" {
			out = append(out, e)
		}
	}
	return out
}
