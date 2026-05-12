package usecase

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/dovgalb/project-rupor/internal/room/domain"
)

type CreateRoomInput struct {
	ActorID uuid.UUID
	Name    string
}

type CreateRoomOutput struct {
	Room *domain.Room
}

type CreateRoom struct {
	rooms RoomRepository
	clock Clock
	uuids UUIDGenerator
}

func NewCreateRoom(rooms RoomRepository, clock Clock, uuids UUIDGenerator) *CreateRoom {
	return &CreateRoom{rooms: rooms, clock: clock, uuids: uuids}
}

func (uc *CreateRoom) Execute(ctx context.Context, in CreateRoomInput) (CreateRoomOutput, error) {
	actorID, err := domain.NewUserID(in.ActorID)
	if err != nil {
		return CreateRoomOutput{}, err
	}
	name, err := domain.NewRoomName(in.Name)
	if err != nil {
		return CreateRoomOutput{}, err
	}
	roomID, err := domain.NewRoomID(uc.uuids.New())
	if err != nil {
		return CreateRoomOutput{}, fmt.Errorf("usecase: create_room: new room id: %w", err)
	}
	now := uc.clock.Now()
	room, err := domain.NewRoom(roomID, actorID, name, now)
	if err != nil {
		return CreateRoomOutput{}, fmt.Errorf("usecase: create_room: new room: %w", err)
	}
	owner, err := domain.NewMembership(roomID, actorID, domain.RoleOwner, now)
	if err != nil {
		return CreateRoomOutput{}, fmt.Errorf("usecase: create_room: new owner membership: %w", err)
	}
	if err := uc.rooms.SaveWithOwner(ctx, room, owner); err != nil {
		return CreateRoomOutput{}, fmt.Errorf("usecase: create_room: save: %w", err)
	}
	return CreateRoomOutput{Room: room}, nil
}
