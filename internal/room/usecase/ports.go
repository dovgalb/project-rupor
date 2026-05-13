package usecase

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"

	"github.com/dovgalb/project-rupor/internal/room/domain"
)

type RoomRepository interface {
	SaveWithOwner(ctx context.Context, room *domain.Room, ownerMembership *domain.Membership) error
	FindByID(ctx context.Context, id domain.RoomID) (*domain.Room, error)
	ListByMember(ctx context.Context, userID domain.UserID) ([]RoomWithRole, error)
	Delete(ctx context.Context, id domain.RoomID) error
}

type RoomWithRole struct {
	Room *domain.Room
	Role domain.Role
}

type MembershipRepository interface {
	FindByPair(ctx context.Context, roomID domain.RoomID, userID domain.UserID) (*domain.Membership, error)
	ListByRoom(ctx context.Context, roomID domain.RoomID) ([]*domain.Membership, error)
	Add(ctx context.Context, m *domain.Membership) error
}

type InviteRepository interface {
	// RegenerateActive в одной транзакции отзывает все активные коды комнаты
	// и вставляет новый invite. На коллизию по invites_active_code (partial unique)
	// возвращает ErrInviteCodeCollision для управляемого ретрая в RegenerateInvite.
	RegenerateActive(ctx context.Context, invite *domain.Invite) error
	FindActiveByCode(ctx context.Context, code domain.InviteCode) (*domain.Invite, error)
}

type InviteCodeGenerator interface {
	New() (domain.InviteCode, error)
}

type Clock interface {
	Now() time.Time
}

type UUIDGenerator interface {
	New() uuid.UUID
}

// ErrInviteCodeCollision — внутренний sentinel для управляемого ретрая в RegenerateInvite.
// Не доменная ошибка, не должна всплыть наружу transport-слоя.
var ErrInviteCodeCollision = errors.New("usecase: invite code collision (retry)")

// RoomEventsPublisher — публикация доменных событий из room/usecase.
// Контракт: best-effort, ошибки не пробрасываются, реализация не блокирует.
type RoomEventsPublisher interface {
	PublishMemberJoined(roomID domain.RoomID, userID domain.UserID, joinedAt time.Time)
}
