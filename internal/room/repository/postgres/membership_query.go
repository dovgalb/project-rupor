package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	chdom "github.com/dovgalb/project-rupor/internal/channel/domain"
	chuc "github.com/dovgalb/project-rupor/internal/channel/usecase"
	"github.com/dovgalb/project-rupor/internal/room/domain"
	"github.com/dovgalb/project-rupor/internal/room/repository/postgres/db"
)

// MembershipQueryAdapter — кросс-доменный адаптер: реализует
// channel/usecase.MembershipQuery поверх room_members. Это единственная
// санкционированная связь room/repository → channel/usecase + channel/domain
// (см. 03-decisions.md §D-07, §D-19).
type MembershipQueryAdapter struct {
	q *db.Queries
}

func NewMembershipQueryAdapter(pool *pgxpool.Pool) *MembershipQueryAdapter {
	return &MembershipQueryAdapter{q: db.New(pool)}
}

func (a *MembershipQueryAdapter) Require(
	ctx context.Context,
	roomID chdom.RoomID,
	userID chdom.UserID,
	req chuc.RoleRequirement,
) error {
	row, err := a.q.GetRoomMember(ctx, db.GetRoomMemberParams{
		RoomID: roomID.UUID(),
		UserID: userID.UUID(),
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return chdom.ErrChannelAccessDenied
		}
		return fmt.Errorf("postgres: membership query: %w", err)
	}
	role, perr := domain.ParseRole(row.Role)
	if perr != nil {
		return fmt.Errorf("postgres: membership query: parse role: %w", perr)
	}
	if !satisfies(role, req) {
		return chdom.ErrChannelInsufficientRole
	}
	return nil
}

func satisfies(role domain.Role, req chuc.RoleRequirement) bool {
	switch req {
	case chuc.RoleAnyMember:
		return role == domain.RoleMember || role == domain.RoleAdmin || role == domain.RoleOwner
	case chuc.RoleAdminOrOwner:
		return role == domain.RoleAdmin || role == domain.RoleOwner
	case chuc.RoleOwnerOnly:
		return role == domain.RoleOwner
	default:
		return false
	}
}
