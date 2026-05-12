package usecase

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/dovgalb/project-rupor/internal/channel/domain"
)

type ChannelRepository interface {
	Save(ctx context.Context, ch *domain.Channel) error
	ListByRoom(ctx context.Context, roomID domain.RoomID) ([]*domain.Channel, error)
	DeleteInRoom(ctx context.Context, channelID domain.ChannelID, roomID domain.RoomID) error
}

type RoleRequirement int

const (
	RoleAnyMember    RoleRequirement = iota // owner | admin | member
	RoleAdminOrOwner                        // admin | owner
	RoleOwnerOnly                           // owner
)

// MembershipQuery — кросс-доменный порт, реализуется адаптером в room/repository/postgres.
// Контракт ошибок:
//
//	nil                              — пользователь имеет требуемую роль (или выше).
//	domain.ErrChannelAccessDenied    — пользователь не является членом комнаты.
//	domain.ErrChannelInsufficientRole — является членом, но роль ниже требуемой.
//	обёрнутая через fmt.Errorf       — техническая ошибка.
type MembershipQuery interface {
	Require(ctx context.Context, roomID domain.RoomID, userID domain.UserID, req RoleRequirement) error
}

type Clock interface {
	Now() time.Time
}

type UUIDGenerator interface {
	New() uuid.UUID
}
