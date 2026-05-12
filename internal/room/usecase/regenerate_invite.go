package usecase

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"

	"github.com/dovgalb/project-rupor/internal/room/domain"
)

const maxInviteCodeRetries = 3

type RegenerateInviteInput struct {
	ActorID uuid.UUID
	RoomID  uuid.UUID
}

type RegenerateInviteOutput struct {
	Invite *domain.Invite
}

type RegenerateInvite struct {
	invites     InviteRepository
	memberships MembershipRepository
	codes       InviteCodeGenerator
	clock       Clock
	uuids       UUIDGenerator
}

func NewRegenerateInvite(
	invites InviteRepository,
	memberships MembershipRepository,
	codes InviteCodeGenerator,
	clock Clock,
	uuids UUIDGenerator,
) *RegenerateInvite {
	return &RegenerateInvite{
		invites:     invites,
		memberships: memberships,
		codes:       codes,
		clock:       clock,
		uuids:       uuids,
	}
}

func (uc *RegenerateInvite) Execute(ctx context.Context, in RegenerateInviteInput) (RegenerateInviteOutput, error) {
	actorID, err := domain.NewUserID(in.ActorID)
	if err != nil {
		return RegenerateInviteOutput{}, err
	}
	roomID, err := domain.NewRoomID(in.RoomID)
	if err != nil {
		return RegenerateInviteOutput{}, err
	}
	m, err := uc.memberships.FindByPair(ctx, roomID, actorID)
	if err != nil {
		return RegenerateInviteOutput{}, err
	}
	if !m.CanGenerateInvite() {
		return RegenerateInviteOutput{}, domain.ErrInsufficientRole
	}

	var lastErr error
	for attempt := 0; attempt < maxInviteCodeRetries; attempt++ {
		code, err := uc.codes.New()
		if err != nil {
			return RegenerateInviteOutput{}, fmt.Errorf("usecase: regenerate_invite: codes.New: %w", err)
		}
		inviteID, err := domain.NewInviteID(uc.uuids.New())
		if err != nil {
			return RegenerateInviteOutput{}, fmt.Errorf("usecase: regenerate_invite: invite id: %w", err)
		}
		invite, err := domain.NewInvite(inviteID, roomID, code, actorID, uc.clock.Now())
		if err != nil {
			return RegenerateInviteOutput{}, fmt.Errorf("usecase: regenerate_invite: new invite: %w", err)
		}
		if err := uc.invites.RegenerateActive(ctx, invite); err != nil {
			if errors.Is(err, ErrInviteCodeCollision) {
				lastErr = err
				continue
			}
			return RegenerateInviteOutput{}, err
		}
		return RegenerateInviteOutput{Invite: invite}, nil
	}
	return RegenerateInviteOutput{}, fmt.Errorf("usecase: regenerate_invite: max retries exceeded: %w", lastErr)
}
