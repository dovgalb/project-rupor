package wschat

import (
	"context"

	"github.com/google/uuid"
)

// MembershipReader — миниатюрный порт для авто-подписки на room-topic'и
// при connect. Реализуется тонким адаптером поверх room-репозитория.
type MembershipReader interface {
	ListRoomIDsByUser(ctx context.Context, userID uuid.UUID) ([]uuid.UUID, error)
}
