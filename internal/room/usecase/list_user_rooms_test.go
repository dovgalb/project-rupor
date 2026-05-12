package usecase_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/dovgalb/project-rupor/internal/room/domain"
	"github.com/dovgalb/project-rupor/internal/room/usecase"
)

func TestListUserRooms_ReturnsRoomsWithRoles(t *testing.T) {
	t.Parallel()

	rooms := newFakeRoomRepo()
	uc := usecase.NewListUserRooms(rooms)
	actor := uuid.New()
	now := time.Date(2026, 5, 11, 12, 0, 0, 0, time.UTC)

	r1 := mustRoom(t, uuid.New(), uuid.New(), "r1", now)
	r2 := mustRoom(t, uuid.New(), uuid.New(), "r2", now)
	rooms.rooms[r1.ID().UUID()] = r1
	rooms.rooms[r2.ID().UUID()] = r2
	rooms.memberRoles[mkey{room: r1.ID().UUID(), user: actor}] = domain.RoleOwner
	rooms.memberRoles[mkey{room: r2.ID().UUID(), user: actor}] = domain.RoleAdmin

	out, err := uc.Execute(context.Background(), usecase.ListUserRoomsInput{ActorID: actor})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if len(out.Items) != 2 {
		t.Fatalf("Items len = %d, want 2", len(out.Items))
	}
}

func TestListUserRooms_NoRooms_ReturnsEmptyItems(t *testing.T) {
	t.Parallel()

	rooms := newFakeRoomRepo()
	uc := usecase.NewListUserRooms(rooms)

	out, err := uc.Execute(context.Background(), usecase.ListUserRoomsInput{ActorID: uuid.New()})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if len(out.Items) != 0 {
		t.Fatalf("Items len = %d, want 0", len(out.Items))
	}
}
