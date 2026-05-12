package usecase_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/dovgalb/project-rupor/internal/chat/domain"
	"github.com/dovgalb/project-rupor/internal/chat/usecase"
)

type listMessagesSUT struct {
	uc         *usecase.ListMessages
	messages   *fakeMessageRepo
	membership *fakeMembershipQuery
	actor      uuid.UUID
	channel    uuid.UUID
	room       uuid.UUID
	now        time.Time
}

func newListMessagesSUT(t *testing.T) *listMessagesSUT {
	t.Helper()
	now := time.Date(2026, 5, 12, 12, 0, 0, 0, time.UTC)
	actor := uuid.New()
	channel := uuid.New()
	room := uuid.New()
	messages := newFakeMessageRepo()
	messages.withChannel(channel, room, usecase.ChannelKindText)
	membership := newFakeMembershipQuery()
	return &listMessagesSUT{
		uc:         usecase.NewListMessages(messages, membership),
		messages:   messages,
		membership: membership,
		actor:      actor,
		channel:    channel,
		room:       room,
		now:        now,
	}
}

func (s *listMessagesSUT) seed(t *testing.T, n int) []*domain.Message {
	t.Helper()
	var out []*domain.Message
	for i := 0; i < n; i++ {
		m := mustMessage(t, uuid.New(), s.channel, s.actor, "msg-"+uuid.NewString()[:4], s.now.Add(time.Duration(i)*time.Second))
		s.messages.listByChan[s.channel] = append(s.messages.listByChan[s.channel], m)
		out = append(out, m)
	}
	return out
}

func TestListMessages_AsMember_ReturnsItems(t *testing.T) {
	t.Parallel()

	sut := newListMessagesSUT(t)
	sut.seed(t, 3)

	out, err := sut.uc.Execute(context.Background(), usecase.ListMessagesInput{
		ActorID:   sut.actor,
		ChannelID: sut.channel,
		Limit:     10,
	})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if len(out.Items) != 3 {
		t.Fatalf("len(items) = %d, want 3", len(out.Items))
	}
	if out.NextBefore != uuid.Nil {
		t.Fatalf("NextBefore should be Nil for non-full page")
	}
}

func TestListMessages_WithBeforeCursor_ReturnsBefore(t *testing.T) {
	t.Parallel()

	sut := newListMessagesSUT(t)
	seeded := sut.seed(t, 5)

	out, err := sut.uc.Execute(context.Background(), usecase.ListMessagesInput{
		ActorID:   sut.actor,
		ChannelID: sut.channel,
		Before:    seeded[1].ID().UUID(),
		Limit:     10,
	})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	// fake возвращает элементы после cursor'а в порядке вставки.
	if len(out.Items) != 3 {
		t.Fatalf("len(items) = %d, want 3", len(out.Items))
	}
}

func TestListMessages_NotMember_ReturnsErrAccessDenied(t *testing.T) {
	t.Parallel()

	sut := newListMessagesSUT(t)
	sut.membership.withErr(domain.ErrChatAccessDenied)

	_, err := sut.uc.Execute(context.Background(), usecase.ListMessagesInput{
		ActorID:   sut.actor,
		ChannelID: sut.channel,
		Limit:     10,
	})
	if !errors.Is(err, domain.ErrChatAccessDenied) {
		t.Fatalf("got %v, want ErrChatAccessDenied", err)
	}
}

func TestListMessages_EmptyChannel_ReturnsEmpty(t *testing.T) {
	t.Parallel()

	sut := newListMessagesSUT(t)

	out, err := sut.uc.Execute(context.Background(), usecase.ListMessagesInput{
		ActorID:   sut.actor,
		ChannelID: sut.channel,
		Limit:     10,
	})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if len(out.Items) != 0 {
		t.Fatalf("len(items) = %d, want 0", len(out.Items))
	}
	if out.NextBefore != uuid.Nil {
		t.Fatal("NextBefore should be Nil for empty page")
	}
}

func TestListMessages_LimitZero_UsesDefault(t *testing.T) {
	t.Parallel()

	sut := newListMessagesSUT(t)
	sut.seed(t, 60)

	out, err := sut.uc.Execute(context.Background(), usecase.ListMessagesInput{
		ActorID:   sut.actor,
		ChannelID: sut.channel,
		Limit:     0, // → DefaultLimit = 50
	})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if len(out.Items) != usecase.DefaultLimit {
		t.Fatalf("len(items) = %d, want %d", len(out.Items), usecase.DefaultLimit)
	}
}

func TestListMessages_LimitOutOfRange_ReturnsError(t *testing.T) {
	t.Parallel()

	sut := newListMessagesSUT(t)

	cases := []int{-1, 101, 1000}
	for _, lim := range cases {
		lim := lim
		t.Run("limit", func(t *testing.T) {
			t.Parallel()
			_, err := sut.uc.Execute(context.Background(), usecase.ListMessagesInput{
				ActorID:   sut.actor,
				ChannelID: sut.channel,
				Limit:     lim,
			})
			if err == nil {
				t.Fatalf("limit=%d: ожидали ошибку", lim)
			}
		})
	}
}

func TestListMessages_NextBeforeCursor_WhenFullPage(t *testing.T) {
	t.Parallel()

	sut := newListMessagesSUT(t)
	seeded := sut.seed(t, 10)

	out, err := sut.uc.Execute(context.Background(), usecase.ListMessagesInput{
		ActorID:   sut.actor,
		ChannelID: sut.channel,
		Limit:     5,
	})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if len(out.Items) != 5 {
		t.Fatalf("len(items) = %d, want 5", len(out.Items))
	}
	want := seeded[4].ID().UUID()
	if out.NextBefore != want {
		t.Fatalf("NextBefore = %v, want %v", out.NextBefore, want)
	}
}

func TestListMessages_UnknownChannel_RepoCalledStillReturnsEmpty(t *testing.T) {
	t.Parallel()

	// ListMessages не вызывает ChannelOf — он вызывает Require + ListByChannel.
	// Для несуществующего канала Require вернёт ErrChannelNotFound (см. контракт MembershipQuery).
	sut := newListMessagesSUT(t)
	sut.membership.withErr(domain.ErrChannelNotFound)

	_, err := sut.uc.Execute(context.Background(), usecase.ListMessagesInput{
		ActorID:   sut.actor,
		ChannelID: sut.channel,
		Limit:     10,
	})
	if !errors.Is(err, domain.ErrChannelNotFound) {
		t.Fatalf("got %v, want ErrChannelNotFound", err)
	}
}
