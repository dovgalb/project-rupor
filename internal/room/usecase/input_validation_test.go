package usecase_test

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"

	"github.com/dovgalb/project-rupor/internal/room/domain"
	"github.com/dovgalb/project-rupor/internal/room/usecase"
)

// Все use case'ы парсят ActorID/RoomID/Code на входе. Эти тесты покрывают
// общие ветки невалидных идентификаторов на границе usecase, до похода в репозиторий.

func TestCreateRoom_ZeroActorID_ReturnsErrInvalidUserID(t *testing.T) {
	t.Parallel()
	sut := newCreateRoomSUT(t)
	_, err := sut.uc.Execute(context.Background(), usecase.CreateRoomInput{
		ActorID: uuid.Nil,
		Name:    "general",
	})
	if !errors.Is(err, domain.ErrInvalidUserID) {
		t.Fatalf("got %v, want ErrInvalidUserID", err)
	}
}

func TestGetRoom_ZeroActorID_ReturnsErrInvalidUserID(t *testing.T) {
	t.Parallel()
	sut := newGetRoomSUT(t)
	_, err := sut.uc.Execute(context.Background(), usecase.GetRoomInput{
		ActorID: uuid.Nil,
		RoomID:  uuid.New(),
	})
	if !errors.Is(err, domain.ErrInvalidUserID) {
		t.Fatalf("got %v, want ErrInvalidUserID", err)
	}
}

func TestGetRoom_ZeroRoomID_ReturnsErrInvalidRoomID(t *testing.T) {
	t.Parallel()
	sut := newGetRoomSUT(t)
	_, err := sut.uc.Execute(context.Background(), usecase.GetRoomInput{
		ActorID: uuid.New(),
		RoomID:  uuid.Nil,
	})
	if !errors.Is(err, domain.ErrInvalidRoomID) {
		t.Fatalf("got %v, want ErrInvalidRoomID", err)
	}
}

func TestListUserRooms_ZeroActorID_ReturnsErr(t *testing.T) {
	t.Parallel()
	uc := usecase.NewListUserRooms(newFakeRoomRepo())
	_, err := uc.Execute(context.Background(), usecase.ListUserRoomsInput{ActorID: uuid.Nil})
	if !errors.Is(err, domain.ErrInvalidUserID) {
		t.Fatalf("got %v, want ErrInvalidUserID", err)
	}
}

func TestDeleteRoom_ZeroActorID_ReturnsErr(t *testing.T) {
	t.Parallel()
	sut := newDeleteRoomSUT(t)
	err := sut.uc.Execute(context.Background(), usecase.DeleteRoomInput{
		ActorID: uuid.Nil,
		RoomID:  uuid.New(),
	})
	if !errors.Is(err, domain.ErrInvalidUserID) {
		t.Fatalf("got %v, want ErrInvalidUserID", err)
	}
}

func TestDeleteRoom_ZeroRoomID_ReturnsErr(t *testing.T) {
	t.Parallel()
	sut := newDeleteRoomSUT(t)
	err := sut.uc.Execute(context.Background(), usecase.DeleteRoomInput{
		ActorID: uuid.New(),
		RoomID:  uuid.Nil,
	})
	if !errors.Is(err, domain.ErrInvalidRoomID) {
		t.Fatalf("got %v, want ErrInvalidRoomID", err)
	}
}

func TestListMembers_ZeroActorID_ReturnsErr(t *testing.T) {
	t.Parallel()
	uc := usecase.NewListMembers(newFakeMembershipRepo())
	_, err := uc.Execute(context.Background(), usecase.ListMembersInput{
		ActorID: uuid.Nil,
		RoomID:  uuid.New(),
	})
	if !errors.Is(err, domain.ErrInvalidUserID) {
		t.Fatalf("got %v, want ErrInvalidUserID", err)
	}
}

func TestListMembers_ZeroRoomID_ReturnsErr(t *testing.T) {
	t.Parallel()
	uc := usecase.NewListMembers(newFakeMembershipRepo())
	_, err := uc.Execute(context.Background(), usecase.ListMembersInput{
		ActorID: uuid.New(),
		RoomID:  uuid.Nil,
	})
	if !errors.Is(err, domain.ErrInvalidRoomID) {
		t.Fatalf("got %v, want ErrInvalidRoomID", err)
	}
}

func TestRegenerateInvite_ZeroActorID_ReturnsErr(t *testing.T) {
	t.Parallel()
	sut := newRegenInviteSUT(t, nil, nil)
	_, err := sut.uc.Execute(context.Background(), usecase.RegenerateInviteInput{
		ActorID: uuid.Nil,
		RoomID:  uuid.New(),
	})
	if !errors.Is(err, domain.ErrInvalidUserID) {
		t.Fatalf("got %v, want ErrInvalidUserID", err)
	}
}

func TestRegenerateInvite_ZeroRoomID_ReturnsErr(t *testing.T) {
	t.Parallel()
	sut := newRegenInviteSUT(t, nil, nil)
	_, err := sut.uc.Execute(context.Background(), usecase.RegenerateInviteInput{
		ActorID: uuid.New(),
		RoomID:  uuid.Nil,
	})
	if !errors.Is(err, domain.ErrInvalidRoomID) {
		t.Fatalf("got %v, want ErrInvalidRoomID", err)
	}
}

func TestJoinByCode_ZeroActorID_ReturnsErr(t *testing.T) {
	t.Parallel()
	sut := newJoinByCodeSUT(t)
	_, err := sut.uc.Execute(context.Background(), usecase.JoinByCodeInput{
		ActorID: uuid.Nil,
		Code:    "ABCDEFGH",
	})
	if !errors.Is(err, domain.ErrInvalidUserID) {
		t.Fatalf("got %v, want ErrInvalidUserID", err)
	}
}
