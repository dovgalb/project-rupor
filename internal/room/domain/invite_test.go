package domain_test

import (
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/dovgalb/project-rupor/internal/room/domain"
)

func TestNewInvite_Valid_ConstructsActive(t *testing.T) {
	t.Parallel()

	id := mustInviteID(t, uuid.New())
	roomID := mustRoomID(t, uuid.New())
	code := mustInviteCode(t, "ABCDEFGH")
	createdBy := mustUserID(t, uuid.New())

	inv, err := domain.NewInvite(id, roomID, code, createdBy, defaultBuilderTime)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !inv.IsActive() {
		t.Fatalf("new invite should be active")
	}
	if inv.IsRevoked() {
		t.Fatalf("new invite should not be revoked")
	}
	if !inv.RevokedAt().IsZero() {
		t.Fatalf("RevokedAt should be zero on creation")
	}
}

func TestNewInvite_ZeroFields_ReturnErrors(t *testing.T) {
	t.Parallel()

	validID := mustInviteID(t, uuid.New())
	validRoom := mustRoomID(t, uuid.New())
	validCode := mustInviteCode(t, "ABCDEFGH")
	validUser := mustUserID(t, uuid.New())

	cases := []struct {
		name      string
		id        domain.InviteID
		roomID    domain.RoomID
		code      domain.InviteCode
		createdBy domain.UserID
		createdAt time.Time
		wantErr   error
	}{
		{
			name:      "zero id",
			id:        domain.InviteID{},
			roomID:    validRoom,
			code:      validCode,
			createdBy: validUser,
			createdAt: defaultBuilderTime,
			wantErr:   domain.ErrInvalidInviteID,
		},
		{
			name:      "zero roomID",
			id:        validID,
			roomID:    domain.RoomID{},
			code:      validCode,
			createdBy: validUser,
			createdAt: defaultBuilderTime,
			wantErr:   domain.ErrInvalidRoomID,
		},
		{
			name:      "zero createdBy",
			id:        validID,
			roomID:    validRoom,
			code:      validCode,
			createdBy: domain.UserID{},
			createdAt: defaultBuilderTime,
			wantErr:   domain.ErrInvalidUserID,
		},
		{
			name:      "zero createdAt",
			id:        validID,
			roomID:    validRoom,
			code:      validCode,
			createdBy: validUser,
			createdAt: time.Time{},
			wantErr:   domain.ErrInvalidCreatedAt,
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, err := domain.NewInvite(tc.id, tc.roomID, tc.code, tc.createdBy, tc.createdAt)
			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("got %v, want %v", err, tc.wantErr)
			}
		})
	}
}

func TestInvite_Revoke_SetsRevokedAt(t *testing.T) {
	t.Parallel()

	inv := NewInviteBuilder(t).Build()
	revokeAt := defaultBuilderTime.Add(time.Hour)

	if err := inv.Revoke(revokeAt); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if inv.IsActive() {
		t.Fatalf("invite should not be active after Revoke")
	}
	if !inv.IsRevoked() {
		t.Fatalf("invite should be revoked")
	}
	if !inv.RevokedAt().Equal(revokeAt) {
		t.Fatalf("RevokedAt = %v, want %v", inv.RevokedAt(), revokeAt)
	}
}

func TestInvite_Revoke_AlreadyRevoked_ReturnsErr(t *testing.T) {
	t.Parallel()

	revokeAt := defaultBuilderTime.Add(time.Hour)
	inv := NewInviteBuilder(t).Revoked(revokeAt).Build()

	err := inv.Revoke(defaultBuilderTime.Add(2 * time.Hour))
	if !errors.Is(err, domain.ErrInviteAlreadyRevoked) {
		t.Fatalf("got %v, want ErrInviteAlreadyRevoked", err)
	}
	if !inv.RevokedAt().Equal(revokeAt) {
		t.Fatalf("RevokedAt mutated: got %v, want %v", inv.RevokedAt(), revokeAt)
	}
}

func TestInvite_Revoke_ZeroNow_ReturnsErr(t *testing.T) {
	t.Parallel()

	inv := NewInviteBuilder(t).Build()
	err := inv.Revoke(time.Time{})
	if !errors.Is(err, domain.ErrInvalidCreatedAt) {
		t.Fatalf("got %v, want ErrInvalidCreatedAt", err)
	}
	if inv.IsRevoked() {
		t.Fatalf("invite should remain active after failed Revoke")
	}
}

func TestReconstructInvite_RevokedRow_RestoresState(t *testing.T) {
	t.Parallel()

	revokeAt := defaultBuilderTime.Add(time.Hour)
	inv, err := domain.ReconstructInvite(
		mustInviteID(t, uuid.New()),
		mustRoomID(t, uuid.New()),
		mustInviteCode(t, "ABCDEFGH"),
		mustUserID(t, uuid.New()),
		defaultBuilderTime,
		revokeAt,
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !inv.IsRevoked() {
		t.Fatalf("reconstructed invite should be revoked")
	}
	if !inv.RevokedAt().Equal(revokeAt) {
		t.Fatalf("RevokedAt = %v, want %v", inv.RevokedAt(), revokeAt)
	}
}

func TestReconstructInvite_ActiveRow_RestoresActive(t *testing.T) {
	t.Parallel()

	inv, err := domain.ReconstructInvite(
		mustInviteID(t, uuid.New()),
		mustRoomID(t, uuid.New()),
		mustInviteCode(t, "ABCDEFGH"),
		mustUserID(t, uuid.New()),
		defaultBuilderTime,
		time.Time{},
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !inv.IsActive() {
		t.Fatalf("reconstructed invite with zero revokedAt should be active")
	}
}
