package usecase

import (
	"context"

	"github.com/google/uuid"

	"github.com/dovgalb/project-rupor/internal/room/domain"
)

type GetRoomInput struct {
	ActorID uuid.UUID
	RoomID  uuid.UUID
}

type GetRoomOutput struct {
	Room *domain.Room
}

type GetRoom struct {
	rooms       RoomRepository
	memberships MembershipRepository
}

func NewGetRoom(rooms RoomRepository, memberships MembershipRepository) *GetRoom {
	return &GetRoom{rooms: rooms, memberships: memberships}
}

func (uc *GetRoom) Execute(ctx context.Context, in GetRoomInput) (GetRoomOutput, error) {
	actorID, err := domain.NewUserID(in.ActorID)
	if err != nil {
		return GetRoomOutput{}, err
	}
	roomID, err := domain.NewRoomID(in.RoomID)
	if err != nil {
		return GetRoomOutput{}, err
	}
	// Факт существования membership = доступ. CanReadRoom() == true для всех ролей.
	if _, ferr := uc.memberships.FindByPair(ctx, roomID, actorID); ferr != nil {
		return GetRoomOutput{}, ferr
	}
	room, err := uc.rooms.FindByID(ctx, roomID)
	if err != nil {
		return GetRoomOutput{}, err
	}
	return GetRoomOutput{Room: room}, nil
}
