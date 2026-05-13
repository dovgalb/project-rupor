package usecase_test

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/dovgalb/project-rupor/internal/chat/domain"
	"github.com/dovgalb/project-rupor/internal/chat/usecase"
)

type sendMessageSUT struct {
	uc          *usecase.SendMessage
	messages    *fakeMessageRepo
	membership  *fakeMembershipQuery
	broadcaster *fakeBroadcaster
	clock       *fixedClock
	uuids       *fixedUUID
	now         time.Time
	actor       uuid.UUID
	channel     uuid.UUID
	room        uuid.UUID
	msgID       uuid.UUID
}

func newSendMessageSUT(t *testing.T) *sendMessageSUT {
	t.Helper()
	now := time.Date(2026, 5, 12, 12, 0, 0, 0, time.UTC)
	actor := uuid.New()
	channel := uuid.New()
	room := uuid.New()
	msgID := uuid.New()
	messages := newFakeMessageRepo()
	messages.withChannel(channel, room, usecase.ChannelKindText)
	membership := newFakeMembershipQuery()
	broadcaster := newFakeBroadcaster()
	clock := &fixedClock{now: now}
	uuids := &fixedUUID{next: []uuid.UUID{msgID}}
	return &sendMessageSUT{
		uc:          usecase.NewSendMessage(messages, membership, broadcaster, clock, uuids),
		messages:    messages,
		membership:  membership,
		broadcaster: broadcaster,
		clock:       clock,
		uuids:       uuids,
		now:         now,
		actor:       actor,
		channel:     channel,
		room:        room,
		msgID:       msgID,
	}
}

func TestSendMessage_AsMember_PersistsAndPublishes(t *testing.T) {
	t.Parallel()

	sut := newSendMessageSUT(t)

	out, err := sut.uc.Execute(context.Background(), usecase.SendMessageInput{
		ActorID:   sut.actor,
		ChannelID: sut.channel,
		Text:      "hello",
	})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if out.MessageID != sut.msgID {
		t.Fatalf("MessageID = %v, want %v", out.MessageID, sut.msgID)
	}
	if out.AuthorID != sut.actor {
		t.Fatalf("AuthorID = %v, want %v", out.AuthorID, sut.actor)
	}
	if out.Text != "hello" {
		t.Fatalf("Text = %q, want hello", out.Text)
	}
	if !out.CreatedAt.Equal(sut.now) {
		t.Fatalf("CreatedAt = %v, want %v", out.CreatedAt, sut.now)
	}
	if len(sut.messages.saved) != 1 {
		t.Fatalf("saved = %d, want 1", len(sut.messages.saved))
	}
	evs := sut.broadcaster.channelEvents()
	if len(evs) != 1 || evs[0].EventType != "message.new" {
		t.Fatalf("broadcast: %+v", evs)
	}
}

func TestSendMessage_NotMember_ReturnsErrAccessDenied(t *testing.T) {
	t.Parallel()

	sut := newSendMessageSUT(t)
	sut.membership.withErr(domain.ErrChatAccessDenied)

	_, err := sut.uc.Execute(context.Background(), usecase.SendMessageInput{
		ActorID:   sut.actor,
		ChannelID: sut.channel,
		Text:      "hello",
	})
	if !errors.Is(err, domain.ErrChatAccessDenied) {
		t.Fatalf("got %v, want ErrChatAccessDenied", err)
	}
	if len(sut.messages.saved) != 0 {
		t.Fatalf("Save был вызван, не должен")
	}
	if len(sut.broadcaster.channelEvents()) != 0 {
		t.Fatalf("Publish был вызван, не должен")
	}
}

func TestSendMessage_UnknownChannel_ReturnsErrChannelNotFound(t *testing.T) {
	t.Parallel()

	sut := newSendMessageSUT(t)
	unknownChannel := uuid.New()

	_, err := sut.uc.Execute(context.Background(), usecase.SendMessageInput{
		ActorID:   sut.actor,
		ChannelID: unknownChannel,
		Text:      "hello",
	})
	if !errors.Is(err, domain.ErrChannelNotFound) {
		t.Fatalf("got %v, want ErrChannelNotFound", err)
	}
}

func TestSendMessage_VoiceChannel_ReturnsErrChannelNotText(t *testing.T) {
	t.Parallel()

	sut := newSendMessageSUT(t)
	sut.messages.withChannel(sut.channel, sut.room, usecase.ChannelKindVoice)

	_, err := sut.uc.Execute(context.Background(), usecase.SendMessageInput{
		ActorID:   sut.actor,
		ChannelID: sut.channel,
		Text:      "hello",
	})
	if !errors.Is(err, domain.ErrChannelNotText) {
		t.Fatalf("got %v, want ErrChannelNotText", err)
	}
}

func TestSendMessage_EmptyText_ReturnsErrInvalidMessageText(t *testing.T) {
	t.Parallel()

	sut := newSendMessageSUT(t)
	_, err := sut.uc.Execute(context.Background(), usecase.SendMessageInput{
		ActorID:   sut.actor,
		ChannelID: sut.channel,
		Text:      "",
	})
	if !errors.Is(err, domain.ErrInvalidMessageText) {
		t.Fatalf("got %v, want ErrInvalidMessageText", err)
	}
}

func TestSendMessage_TooLong_ReturnsErrInvalidMessageText(t *testing.T) {
	t.Parallel()

	sut := newSendMessageSUT(t)
	_, err := sut.uc.Execute(context.Background(), usecase.SendMessageInput{
		ActorID:   sut.actor,
		ChannelID: sut.channel,
		Text:      strings.Repeat("a", domain.MessageTextMaxLen+1),
	})
	if !errors.Is(err, domain.ErrInvalidMessageText) {
		t.Fatalf("got %v, want ErrInvalidMessageText", err)
	}
}

func TestSendMessage_SaveFails_DoesNotPublish(t *testing.T) {
	t.Parallel()

	sut := newSendMessageSUT(t)
	sut.messages.saveErr = errors.New("db down")

	_, err := sut.uc.Execute(context.Background(), usecase.SendMessageInput{
		ActorID:   sut.actor,
		ChannelID: sut.channel,
		Text:      "hello",
	})
	if err == nil || err.Error() != "db down" {
		t.Fatalf("got %v, want db down", err)
	}
	if len(sut.broadcaster.channelEvents()) != 0 {
		t.Fatalf("Publish был вызван при ошибке Save")
	}
}

func TestSendMessage_PublishPanics_ReturnsSuccess(t *testing.T) {
	t.Parallel()

	sut := newSendMessageSUT(t)
	sut.broadcaster.panicNext = true

	out, err := sut.uc.Execute(context.Background(), usecase.SendMessageInput{
		ActorID:   sut.actor,
		ChannelID: sut.channel,
		Text:      "hello",
	})
	if err != nil {
		t.Fatalf("Execute: %v (panic в broadcaster должна быть проглочена)", err)
	}
	if out.MessageID != sut.msgID {
		t.Fatalf("MessageID mismatch")
	}
	if len(sut.messages.saved) != 1 {
		t.Fatalf("Save не был вызван")
	}
}

func TestSendMessage_ChannelOfFails_ReturnsError(t *testing.T) {
	t.Parallel()

	sut := newSendMessageSUT(t)
	sut.messages.channelOfErr = errors.New("db down")

	_, err := sut.uc.Execute(context.Background(), usecase.SendMessageInput{
		ActorID:   sut.actor,
		ChannelID: sut.channel,
		Text:      "hello",
	})
	if err == nil {
		t.Fatal("ожидали ошибку")
	}
}

func TestSendMessage_MembershipFails_ReturnsError(t *testing.T) {
	t.Parallel()

	sut := newSendMessageSUT(t)
	sut.membership.withErr(errors.New("db down"))

	_, err := sut.uc.Execute(context.Background(), usecase.SendMessageInput{
		ActorID:   sut.actor,
		ChannelID: sut.channel,
		Text:      "hello",
	})
	if err == nil {
		t.Fatal("ожидали ошибку")
	}
	if len(sut.messages.saved) != 0 {
		t.Fatal("Save был вызван при ошибке Require")
	}
}
