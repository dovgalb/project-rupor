package usecase

import (
	"context"

	"github.com/google/uuid"

	"github.com/dovgalb/project-rupor/internal/channel/domain"
)

type DeleteChannelInput struct {
	ActorID   uuid.UUID
	RoomID    uuid.UUID
	ChannelID uuid.UUID
}

type DeleteChannel struct {
	channels   ChannelRepository
	membership MembershipQuery
}

func NewDeleteChannel(channels ChannelRepository, membership MembershipQuery) *DeleteChannel {
	return &DeleteChannel{channels: channels, membership: membership}
}

func (uc *DeleteChannel) Execute(ctx context.Context, in DeleteChannelInput) error {
	actorID, err := domain.NewUserID(in.ActorID)
	if err != nil {
		return err
	}
	roomID, err := domain.NewRoomID(in.RoomID)
	if err != nil {
		return err
	}
	channelID, err := domain.NewChannelID(in.ChannelID)
	if err != nil {
		return err
	}
	if err := uc.membership.Require(ctx, roomID, actorID, RoleAdminOrOwner); err != nil {
		return err
	}
	if err := uc.channels.DeleteInRoom(ctx, channelID, roomID); err != nil {
		return err
	}
	return nil
}
