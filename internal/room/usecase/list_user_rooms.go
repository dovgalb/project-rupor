package usecase

import (
	"context"

	"github.com/google/uuid"

	"github.com/dovgalb/project-rupor/internal/room/domain"
)

type ListUserRoomsInput struct {
	ActorID uuid.UUID
}

type ListUserRoomsOutput struct {
	Items []RoomWithRole
}

type ListUserRooms struct {
	rooms RoomRepository
}

func NewListUserRooms(rooms RoomRepository) *ListUserRooms {
	return &ListUserRooms{rooms: rooms}
}

func (uc *ListUserRooms) Execute(ctx context.Context, in ListUserRoomsInput) (ListUserRoomsOutput, error) {
	actorID, err := domain.NewUserID(in.ActorID)
	if err != nil {
		return ListUserRoomsOutput{}, err
	}
	items, err := uc.rooms.ListByMember(ctx, actorID)
	if err != nil {
		return ListUserRoomsOutput{}, err
	}
	return ListUserRoomsOutput{Items: items}, nil
}
