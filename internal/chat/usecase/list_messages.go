package usecase

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/dovgalb/project-rupor/internal/chat/domain"
)

const (
	DefaultLimit = 50
	MinLimit     = 1
	MaxLimit     = 100
)

type ListMessages struct {
	messages   MessageRepository
	membership MembershipQuery
}

func NewListMessages(messages MessageRepository, membership MembershipQuery) *ListMessages {
	return &ListMessages{messages: messages, membership: membership}
}

type ListMessagesInput struct {
	ActorID   uuid.UUID
	ChannelID uuid.UUID
	Before    uuid.UUID // uuid.Nil если курсор не задан
	Limit     int
}

type ListMessagesOutput struct {
	Items      []*domain.Message
	NextBefore uuid.UUID // uuid.Nil если это последняя страница
}

func (uc *ListMessages) Execute(ctx context.Context, in ListMessagesInput) (ListMessagesOutput, error) {
	actorID, err := domain.NewUserID(in.ActorID)
	if err != nil {
		return ListMessagesOutput{}, err
	}
	channelID, err := domain.NewChannelID(in.ChannelID)
	if err != nil {
		return ListMessagesOutput{}, err
	}
	var beforeID domain.MessageID
	if in.Before != uuid.Nil {
		beforeID, err = domain.NewMessageID(in.Before)
		if err != nil {
			return ListMessagesOutput{}, err
		}
	}

	limit := in.Limit
	if limit == 0 {
		limit = DefaultLimit
	}
	if limit < MinLimit || limit > MaxLimit {
		return ListMessagesOutput{}, fmt.Errorf("limit out of range [%d..%d]: got %d",
			MinLimit, MaxLimit, in.Limit)
	}

	if reqErr := uc.membership.Require(ctx, channelID, actorID, RoleAnyMember); reqErr != nil {
		return ListMessagesOutput{}, reqErr
	}

	items, err := uc.messages.ListByChannel(ctx, channelID, beforeID, limit)
	if err != nil {
		return ListMessagesOutput{}, err
	}

	var nextBefore uuid.UUID
	if len(items) == limit && limit > 0 {
		nextBefore = items[len(items)-1].ID().UUID()
	}
	return ListMessagesOutput{Items: items, NextBefore: nextBefore}, nil
}
