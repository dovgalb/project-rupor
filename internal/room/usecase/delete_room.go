package usecase

import (
	"context"

	"github.com/google/uuid"

	"github.com/dovgalb/project-rupor/internal/room/domain"
)

type DeleteRoomInput struct {
	ActorID uuid.UUID
	RoomID  uuid.UUID
}

type DeleteRoom struct {
	rooms       RoomRepository
	memberships MembershipRepository
}

func NewDeleteRoom(rooms RoomRepository, memberships MembershipRepository) *DeleteRoom {
	return &DeleteRoom{rooms: rooms, memberships: memberships}
}

func (uc *DeleteRoom) Execute(ctx context.Context, in DeleteRoomInput) error {
	actorID, err := domain.NewUserID(in.ActorID)
	if err != nil {
		return err
	}
	roomID, err := domain.NewRoomID(in.RoomID)
	if err != nil {
		return err
	}
	m, err := uc.memberships.FindByPair(ctx, roomID, actorID)
	if err != nil {
		return err
	}
	if !m.CanDeleteRoom() {
		return domain.ErrInsufficientRole
	}
	if err := uc.rooms.Delete(ctx, roomID); err != nil {
		return err
	}
	return nil
}
