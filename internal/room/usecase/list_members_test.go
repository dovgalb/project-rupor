package usecase_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/dovgalb/project-rupor/internal/room/domain"
	"github.com/dovgalb/project-rupor/internal/room/usecase"
)

func TestListMembers_AsMember_ReturnsAll(t *testing.T) {
	t.Parallel()

	memberships := newFakeMembershipRepo()
	uc := usecase.NewListMembers(memberships)
	actor := uuid.New()
	roomUUID := uuid.New()
	now := time.Date(2026, 5, 11, 12, 0, 0, 0, time.UTC)

	memberships.put(mustMembership(t, roomUUID, actor, domain.RoleMember, now))
	memberships.put(mustMembership(t, roomUUID, uuid.New(), domain.RoleOwner, now))
	memberships.put(mustMembership(t, roomUUID, uuid.New(), domain.RoleAdmin, now))

	out, err := uc.Execute(context.Background(), usecase.ListMembersInput{
		ActorID: actor,
		RoomID:  roomUUID,
	})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if len(out.Items) != 3 {
		t.Fatalf("Items len = %d, want 3", len(out.Items))
	}
}

func TestListMembers_AsAdmin_ReturnsAll(t *testing.T) {
	t.Parallel()

	memberships := newFakeMembershipRepo()
	uc := usecase.NewListMembers(memberships)
	actor := uuid.New()
	roomUUID := uuid.New()
	now := time.Date(2026, 5, 11, 12, 0, 0, 0, time.UTC)
	memberships.put(mustMembership(t, roomUUID, actor, domain.RoleAdmin, now))

	out, err := uc.Execute(context.Background(), usecase.ListMembersInput{
		ActorID: actor,
		RoomID:  roomUUID,
	})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if len(out.Items) != 1 {
		t.Fatalf("Items len = %d, want 1", len(out.Items))
	}
}

func TestListMembers_NotMember_ReturnsErrNotMember(t *testing.T) {
	t.Parallel()

	memberships := newFakeMembershipRepo()
	uc := usecase.NewListMembers(memberships)

	_, err := uc.Execute(context.Background(), usecase.ListMembersInput{
		ActorID: uuid.New(),
		RoomID:  uuid.New(),
	})
	if !errors.Is(err, domain.ErrNotMember) {
		t.Fatalf("got %v, want ErrNotMember", err)
	}
}
