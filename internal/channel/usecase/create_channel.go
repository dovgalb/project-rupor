package usecase

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/dovgalb/project-rupor/internal/channel/domain"
)

type CreateChannelInput struct {
	ActorID uuid.UUID
	RoomID  uuid.UUID
	Name    string
	Kind    string
}

type CreateChannelOutput struct {
	Channel *domain.Channel
}

type CreateChannel struct {
	channels   ChannelRepository
	membership MembershipQuery
	clock      Clock
	uuids      UUIDGenerator
}

func NewCreateChannel(
	channels ChannelRepository,
	membership MembershipQuery,
	clock Clock,
	uuids UUIDGenerator,
) *CreateChannel {
	return &CreateChannel{channels: channels, membership: membership, clock: clock, uuids: uuids}
}

func (uc *CreateChannel) Execute(ctx context.Context, in CreateChannelInput) (CreateChannelOutput, error) {
	actorID, err := domain.NewUserID(in.ActorID)
	if err != nil {
		return CreateChannelOutput{}, err
	}
	roomID, err := domain.NewRoomID(in.RoomID)
	if err != nil {
		return CreateChannelOutput{}, err
	}
	name, err := domain.NewChannelName(in.Name)
	if err != nil {
		return CreateChannelOutput{}, err
	}
	kind, err := domain.ParseChannelKind(in.Kind)
	if err != nil {
		return CreateChannelOutput{}, err
	}
	if reqErr := uc.membership.Require(ctx, roomID, actorID, RoleAdminOrOwner); reqErr != nil {
		return CreateChannelOutput{}, reqErr
	}
	chID, err := domain.NewChannelID(uc.uuids.New())
	if err != nil {
		return CreateChannelOutput{}, fmt.Errorf("usecase: create_channel: new channel id: %w", err)
	}
	ch, err := domain.NewChannel(chID, roomID, name, kind, uc.clock.Now())
	if err != nil {
		return CreateChannelOutput{}, fmt.Errorf("usecase: create_channel: new channel: %w", err)
	}
	if err := uc.channels.Save(ctx, ch); err != nil {
		return CreateChannelOutput{}, err
	}
	return CreateChannelOutput{Channel: ch}, nil
}
