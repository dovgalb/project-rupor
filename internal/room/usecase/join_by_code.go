package usecase

import (
	"context"
	"errors"

	"github.com/google/uuid"

	"github.com/dovgalb/project-rupor/internal/room/domain"
)

type JoinByCodeInput struct {
	ActorID uuid.UUID
	Code    string
}

type JoinByCodeOutput struct {
	Room *domain.Room
}

type JoinByCode struct {
	invites     InviteRepository
	memberships MembershipRepository
	rooms       RoomRepository
	clock       Clock
	events      RoomEventsPublisher
}

func NewJoinByCode(
	invites InviteRepository,
	memberships MembershipRepository,
	rooms RoomRepository,
	clock Clock,
	events RoomEventsPublisher,
) *JoinByCode {
	return &JoinByCode{
		invites:     invites,
		memberships: memberships,
		rooms:       rooms,
		clock:       clock,
		events:      events,
	}
}

func (uc *JoinByCode) Execute(ctx context.Context, in JoinByCodeInput) (JoinByCodeOutput, error) {
	actorID, err := domain.NewUserID(in.ActorID)
	if err != nil {
		return JoinByCodeOutput{}, err
	}
	code, err := domain.NewInviteCode(in.Code)
	if err != nil {
		return JoinByCodeOutput{}, err
	}
	invite, err := uc.invites.FindActiveByCode(ctx, code)
	if err != nil {
		return JoinByCodeOutput{}, err
	}
	// Membership уже есть → конфликт. ErrNotMember → продолжаем join.
	_, mErr := uc.memberships.FindByPair(ctx, invite.RoomID(), actorID)
	if mErr == nil {
		return JoinByCodeOutput{}, domain.ErrAlreadyMember
	}
	if !errors.Is(mErr, domain.ErrNotMember) {
		return JoinByCodeOutput{}, mErr
	}
	m, err := domain.NewMembership(invite.RoomID(), actorID, domain.RoleMember, uc.clock.Now())
	if err != nil {
		return JoinByCodeOutput{}, err
	}
	if addErr := uc.memberships.Add(ctx, m); addErr != nil {
		return JoinByCodeOutput{}, addErr
	}
	// Best-effort publish: panic/error в publisher'е не должны откатить join.
	func() {
		defer func() { _ = recover() }()
		uc.events.PublishMemberJoined(invite.RoomID(), actorID, m.JoinedAt())
	}()
	room, err := uc.rooms.FindByID(ctx, invite.RoomID())
	if err != nil {
		return JoinByCodeOutput{}, err
	}
	return JoinByCodeOutput{Room: room}, nil
}
