package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	chatdom "github.com/dovgalb/project-rupor/internal/chat/domain"
	chatuc "github.com/dovgalb/project-rupor/internal/chat/usecase"
	roomdom "github.com/dovgalb/project-rupor/internal/room/domain"
	"github.com/dovgalb/project-rupor/internal/room/repository/postgres/db"
)

// MembershipQueryChatAdapter — реализация chat/usecase.MembershipQuery
// поверх room-схемы. Отдельная структура от MembershipQueryAdapter
// (для channel-домена), чтобы возвращать chat-доменные ошибки.
type MembershipQueryChatAdapter struct {
	q *db.Queries
}

func NewMembershipQueryChatAdapter(pool *pgxpool.Pool) *MembershipQueryChatAdapter {
	return &MembershipQueryChatAdapter{q: db.New(pool)}
}

func (a *MembershipQueryChatAdapter) Require(
	ctx context.Context,
	channelID chatdom.ChannelID,
	userID chatdom.UserID,
	req chatuc.RoleRequirement,
) error {
	rawRole, err := a.q.GetMemberForChannel(ctx, db.GetMemberForChannelParams{
		ID:     channelID.UUID(),
		UserID: userID.UUID(),
	})
	if err == nil {
		role, parseErr := roomdom.ParseRole(rawRole)
		if parseErr != nil {
			return fmt.Errorf("postgres: parse role: %w", parseErr)
		}
		if !chatSatisfies(req, role) {
			return chatdom.ErrChatInsufficientRole
		}
		return nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return fmt.Errorf("postgres: get member for channel: %w", err)
	}

	// Не нашли membership: различаем «канала нет» и «канал есть, но user — не член».
	exists, err := a.q.ChannelExists(ctx, channelID.UUID())
	if err != nil {
		return fmt.Errorf("postgres: channel exists: %w", err)
	}
	if !exists {
		return chatdom.ErrChannelNotFound
	}
	return chatdom.ErrChatAccessDenied
}

func chatSatisfies(req chatuc.RoleRequirement, role roomdom.Role) bool {
	switch req {
	case chatuc.RoleAnyMember:
		return role == roomdom.RoleMember || role == roomdom.RoleAdmin || role == roomdom.RoleOwner
	case chatuc.RoleAdminOrOwner:
		return role == roomdom.RoleAdmin || role == roomdom.RoleOwner
	case chatuc.RoleOwnerOnly:
		return role == roomdom.RoleOwner
	default:
		return false
	}
}
