package usecase

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/dovgalb/project-rupor/internal/chat/domain"
)

type RoleRequirement int

const (
	RoleAnyMember RoleRequirement = iota
	RoleAdminOrOwner
	RoleOwnerOnly
)

// MembershipQuery — проверка прав на действие в канале.
// Контракт ошибок:
//
//	nil                            — пользователь имеет требуемую роль.
//	domain.ErrChatAccessDenied     — не член комнаты канала.
//	domain.ErrChatInsufficientRole — член, но роль ниже требуемой.
//	domain.ErrChannelNotFound      — канал не существует.
//	обёрнутая fmt.Errorf           — техническая ошибка.
type MembershipQuery interface {
	Require(ctx context.Context, channelID domain.ChannelID, userID domain.UserID, req RoleRequirement) error
}

const (
	ChannelKindText  = "text"
	ChannelKindVoice = "voice"
)

// ChannelInfo — результат MessageRepository.ChannelOf.
type ChannelInfo struct {
	ChannelID domain.ChannelID
	RoomID    domain.RoomID
	Kind      string
}

// MessageRepository — порт хранения сообщений.
type MessageRepository interface {
	Save(ctx context.Context, m *domain.Message) error
	ListByChannel(ctx context.Context, channelID domain.ChannelID, before domain.MessageID, limit int) ([]*domain.Message, error)
	ChannelOf(ctx context.Context, channelID domain.ChannelID) (ChannelInfo, error)
}

// Broadcaster — порт публикации realtime-событий.
// Контракт: реализация не блокирует, ошибки не пробрасывает (best-effort).
type Broadcaster interface {
	PublishToChannel(channelID domain.ChannelID, eventType string, payload any)
	PublishToRoom(roomID domain.RoomID, eventType string, payload any)
}

type Clock interface{ Now() time.Time }

type UUIDGenerator interface{ New() uuid.UUID }
