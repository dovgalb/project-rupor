package usecase

import (
	"context"

	"github.com/google/uuid"

	"github.com/dovgalb/project-rupor/internal/channel/domain"
)

type ListChannelsInput struct {
	ActorID uuid.UUID
	RoomID  uuid.UUID
}

type ListChannelsOutput struct {
	Items []*domain.Channel
}

type ListChannels struct {
	channels   ChannelRepository
	membership MembershipQuery
}

func NewListChannels(channels ChannelRepository, membership MembershipQuery) *ListChannels {
	return &ListChannels{channels: channels, membership: membership}
}

func (uc *ListChannels) Execute(ctx context.Context, in ListChannelsInput) (ListChannelsOutput, error) {
	actorID, err := domain.NewUserID(in.ActorID)
	if err != nil {
		return ListChannelsOutput{}, err
	}
	roomID, err := domain.NewRoomID(in.RoomID)
	if err != nil {
		return ListChannelsOutput{}, err
	}
	if reqErr := uc.membership.Require(ctx, roomID, actorID, RoleAnyMember); reqErr != nil {
		return ListChannelsOutput{}, reqErr
	}
	items, err := uc.channels.ListByRoom(ctx, roomID)
	if err != nil {
		return ListChannelsOutput{}, err
	}
	return ListChannelsOutput{Items: items}, nil
}
