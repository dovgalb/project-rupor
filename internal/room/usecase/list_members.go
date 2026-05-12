package usecase

import (
	"context"

	"github.com/google/uuid"

	"github.com/dovgalb/project-rupor/internal/room/domain"
)

type ListMembersInput struct {
	ActorID uuid.UUID
	RoomID  uuid.UUID
}

type ListMembersOutput struct {
	Items []*domain.Membership
}

type ListMembers struct {
	memberships MembershipRepository
}

func NewListMembers(memberships MembershipRepository) *ListMembers {
	return &ListMembers{memberships: memberships}
}

func (uc *ListMembers) Execute(ctx context.Context, in ListMembersInput) (ListMembersOutput, error) {
	actorID, err := domain.NewUserID(in.ActorID)
	if err != nil {
		return ListMembersOutput{}, err
	}
	roomID, err := domain.NewRoomID(in.RoomID)
	if err != nil {
		return ListMembersOutput{}, err
	}
	// Сначала проверка членства (ErrNotMember). CanReadMembers — true для всех ролей.
	if _, ferr := uc.memberships.FindByPair(ctx, roomID, actorID); ferr != nil {
		return ListMembersOutput{}, ferr
	}
	items, err := uc.memberships.ListByRoom(ctx, roomID)
	if err != nil {
		return ListMembersOutput{}, err
	}
	return ListMembersOutput{Items: items}, nil
}
