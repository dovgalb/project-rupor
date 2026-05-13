package usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/dovgalb/project-rupor/internal/chat/domain"
)

// SendMessage — сценарий отправки нового сообщения в текстовый канал.
type SendMessage struct {
	messages    MessageRepository
	membership  MembershipQuery
	broadcaster Broadcaster
	clock       Clock
	uuids       UUIDGenerator
}

// NewSendMessage собирает сценарий SendMessage из его зависимостей.
func NewSendMessage(
	messages MessageRepository,
	membership MembershipQuery,
	broadcaster Broadcaster,
	clock Clock,
	uuids UUIDGenerator,
) *SendMessage {
	return &SendMessage{
		messages:    messages,
		membership:  membership,
		broadcaster: broadcaster,
		clock:       clock,
		uuids:       uuids,
	}
}

// SendMessageInput — входные данные сценария SendMessage.
type SendMessageInput struct {
	ActorID   uuid.UUID
	ChannelID uuid.UUID
	Text      string
}

// SendMessageOutput — результат сценария: сохранённое сообщение.
type SendMessageOutput struct {
	MessageID uuid.UUID
	ChannelID uuid.UUID
	AuthorID  uuid.UUID
	Text      string
	CreatedAt time.Time
}

// Execute проверяет членство, сохраняет сообщение и публикует событие message.new в шину (best-effort).
func (uc *SendMessage) Execute(ctx context.Context, in SendMessageInput) (SendMessageOutput, error) {
	actorID, err := domain.NewUserID(in.ActorID)
	if err != nil {
		return SendMessageOutput{}, err
	}
	channelID, err := domain.NewChannelID(in.ChannelID)
	if err != nil {
		return SendMessageOutput{}, err
	}
	text, err := domain.NewMessageText(in.Text)
	if err != nil {
		return SendMessageOutput{}, err
	}

	// Order: сначала проверка членства, потом данные канала. Это не утекает
	// инфу о существовании канала пользователю не-члену (он получит
	// ErrChatAccessDenied, не ErrChannelNotFound).
	if reqErr := uc.membership.Require(ctx, channelID, actorID, RoleAnyMember); reqErr != nil {
		return SendMessageOutput{}, reqErr
	}

	info, err := uc.messages.ChannelOf(ctx, channelID)
	if err != nil {
		return SendMessageOutput{}, err
	}
	if info.Kind != ChannelKindText {
		return SendMessageOutput{}, domain.ErrChannelNotText
	}

	msgID, err := domain.NewMessageID(uc.uuids.New())
	if err != nil {
		return SendMessageOutput{}, fmt.Errorf("send message: generate id: %w", err)
	}
	now := uc.clock.Now().UTC()
	msg, err := domain.NewMessage(msgID, channelID, actorID, text, now)
	if err != nil {
		return SendMessageOutput{}, fmt.Errorf("send message: construct: %w", err)
	}

	if saveErr := uc.messages.Save(ctx, msg); saveErr != nil {
		return SendMessageOutput{}, saveErr
	}

	// Broadcast — best-effort. Паника или ошибка в hub'е не должна
	// сломать ответ клиенту: сообщение уже сохранено.
	func() {
		defer func() { _ = recover() }()
		uc.broadcaster.PublishToChannel(channelID, "message.new", messagePayload(msg))
	}()

	return SendMessageOutput{
		MessageID: msg.ID().UUID(),
		ChannelID: msg.ChannelID().UUID(),
		AuthorID:  msg.AuthorID().UUID(),
		Text:      msg.Text().String(),
		CreatedAt: msg.CreatedAt(),
	}, nil
}

// messagePayload собирает payload события `message.new` для realtime-broadcast.
// Snake_case согласован с 05-events.md.
func messagePayload(m *domain.Message) map[string]any {
	return map[string]any{
		"id":         m.ID().String(),
		"channel_id": m.ChannelID().String(),
		"author_id":  m.AuthorID().String(),
		"text":       m.Text().String(),
		"created_at": m.CreatedAt().Format(time.RFC3339Nano),
	}
}
