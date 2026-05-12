package domain_test

import (
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/dovgalb/project-rupor/internal/room/domain"
)

func TestNewMembership_Valid_Constructs(t *testing.T) {
	t.Parallel()

	roomID := mustRoomID(t, uuid.New())
	userID := mustUserID(t, uuid.New())

	m, err := domain.NewMembership(roomID, userID, domain.RoleAdmin, defaultBuilderTime)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if m.RoomID() != roomID {
		t.Fatalf("RoomID mismatch")
	}
	if m.UserID() != userID {
		t.Fatalf("UserID mismatch")
	}
	if m.Role() != domain.RoleAdmin {
		t.Fatalf("Role mismatch")
	}
	if !m.JoinedAt().Equal(defaultBuilderTime) {
		t.Fatalf("JoinedAt mismatch")
	}
}

func TestNewMembership_ZeroRoomID_ReturnsErr(t *testing.T) {
	t.Parallel()

	_, err := domain.NewMembership(
		domain.RoomID{},
		mustUserID(t, uuid.New()),
		domain.RoleMember,
		defaultBuilderTime,
	)
	if !errors.Is(err, domain.ErrInvalidRoomID) {
		t.Fatalf("got %v, want ErrInvalidRoomID", err)
	}
}

func TestNewMembership_ZeroUserID_ReturnsErr(t *testing.T) {
	t.Parallel()

	_, err := domain.NewMembership(
		mustRoomID(t, uuid.New()),
		domain.UserID{},
		domain.RoleMember,
		defaultBuilderTime,
	)
	if !errors.Is(err, domain.ErrInvalidUserID) {
		t.Fatalf("got %v, want ErrInvalidUserID", err)
	}
}

func TestNewMembership_InvalidRole_ReturnsErrInvalidRole(t *testing.T) {
	t.Parallel()

	_, err := domain.NewMembership(
		mustRoomID(t, uuid.New()),
		mustUserID(t, uuid.New()),
		domain.Role(99),
		defaultBuilderTime,
	)
	if !errors.Is(err, domain.ErrInvalidRole) {
		t.Fatalf("got %v, want ErrInvalidRole", err)
	}
}

func TestNewMembership_ZeroJoinedAt_ReturnsErr(t *testing.T) {
	t.Parallel()

	_, err := domain.NewMembership(
		mustRoomID(t, uuid.New()),
		mustUserID(t, uuid.New()),
		domain.RoleMember,
		time.Time{},
	)
	if !errors.Is(err, domain.ErrInvalidCreatedAt) {
		t.Fatalf("got %v, want ErrInvalidCreatedAt", err)
	}
}

// TestMembership_Permissions_TableDriven — табличный тест по матрице 04-testing.md.
func TestMembership_Permissions_TableDriven(t *testing.T) {
	t.Parallel()

	cases := []struct {
		role     domain.Role
		method   string
		callFn   func(*domain.Membership) bool
		expected bool
	}{
		// CanReadRoom — true для всех.
		{domain.RoleOwner, "CanReadRoom", (*domain.Membership).CanReadRoom, true},
		{domain.RoleAdmin, "CanReadRoom", (*domain.Membership).CanReadRoom, true},
		{domain.RoleMember, "CanReadRoom", (*domain.Membership).CanReadRoom, true},

		// CanReadMembers — true для всех.
		{domain.RoleOwner, "CanReadMembers", (*domain.Membership).CanReadMembers, true},
		{domain.RoleAdmin, "CanReadMembers", (*domain.Membership).CanReadMembers, true},
		{domain.RoleMember, "CanReadMembers", (*domain.Membership).CanReadMembers, true},

		// CanReadChannels — true для всех.
		{domain.RoleOwner, "CanReadChannels", (*domain.Membership).CanReadChannels, true},
		{domain.RoleAdmin, "CanReadChannels", (*domain.Membership).CanReadChannels, true},
		{domain.RoleMember, "CanReadChannels", (*domain.Membership).CanReadChannels, true},

		// CanCreateChannel — owner/admin.
		{domain.RoleOwner, "CanCreateChannel", (*domain.Membership).CanCreateChannel, true},
		{domain.RoleAdmin, "CanCreateChannel", (*domain.Membership).CanCreateChannel, true},
		{domain.RoleMember, "CanCreateChannel", (*domain.Membership).CanCreateChannel, false},

		// CanDeleteChannel — owner/admin.
		{domain.RoleOwner, "CanDeleteChannel", (*domain.Membership).CanDeleteChannel, true},
		{domain.RoleAdmin, "CanDeleteChannel", (*domain.Membership).CanDeleteChannel, true},
		{domain.RoleMember, "CanDeleteChannel", (*domain.Membership).CanDeleteChannel, false},

		// CanGenerateInvite — owner/admin.
		{domain.RoleOwner, "CanGenerateInvite", (*domain.Membership).CanGenerateInvite, true},
		{domain.RoleAdmin, "CanGenerateInvite", (*domain.Membership).CanGenerateInvite, true},
		{domain.RoleMember, "CanGenerateInvite", (*domain.Membership).CanGenerateInvite, false},

		// CanDeleteRoom — только owner.
		{domain.RoleOwner, "CanDeleteRoom", (*domain.Membership).CanDeleteRoom, true},
		{domain.RoleAdmin, "CanDeleteRoom", (*domain.Membership).CanDeleteRoom, false},
		{domain.RoleMember, "CanDeleteRoom", (*domain.Membership).CanDeleteRoom, false},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.role.String()+"_"+tc.method, func(t *testing.T) {
			t.Parallel()
			m := NewMembershipBuilder(t).WithRole(tc.role).Build()
			if got := tc.callFn(m); got != tc.expected {
				t.Fatalf("%s on %s = %v, want %v", tc.method, tc.role, got, tc.expected)
			}
		})
	}
}

func TestMembership_Promote_MemberToAdmin_Succeeds(t *testing.T) {
	t.Parallel()

	m := NewMembershipBuilder(t).Member().Build()
	if err := m.Promote(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if m.Role() != domain.RoleAdmin {
		t.Fatalf("Role = %v, want RoleAdmin", m.Role())
	}
}

func TestMembership_Promote_AdminNoOp(t *testing.T) {
	t.Parallel()

	m := NewMembershipBuilder(t).Admin().Build()
	if err := m.Promote(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if m.Role() != domain.RoleAdmin {
		t.Fatalf("Role = %v, want RoleAdmin (no-op)", m.Role())
	}
}

func TestMembership_Promote_OwnerNoOp(t *testing.T) {
	t.Parallel()

	m := NewMembershipBuilder(t).Owner().Build()
	if err := m.Promote(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if m.Role() != domain.RoleOwner {
		t.Fatalf("Role = %v, want RoleOwner (no-op)", m.Role())
	}
}

func TestMembership_Demote_AdminToMember_Succeeds(t *testing.T) {
	t.Parallel()

	m := NewMembershipBuilder(t).Admin().Build()
	if err := m.Demote(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if m.Role() != domain.RoleMember {
		t.Fatalf("Role = %v, want RoleMember", m.Role())
	}
}

func TestMembership_Demote_MemberNoOp(t *testing.T) {
	t.Parallel()

	m := NewMembershipBuilder(t).Member().Build()
	if err := m.Demote(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if m.Role() != domain.RoleMember {
		t.Fatalf("Role = %v, want RoleMember (no-op)", m.Role())
	}
}

func TestMembership_Demote_Owner_ReturnsErrCannotDemoteOwner(t *testing.T) {
	t.Parallel()

	m := NewMembershipBuilder(t).Owner().Build()
	err := m.Demote()
	if !errors.Is(err, domain.ErrCannotDemoteOwner) {
		t.Fatalf("got %v, want ErrCannotDemoteOwner", err)
	}
	if m.Role() != domain.RoleOwner {
		t.Fatalf("Role mutated after failed demote: %v", m.Role())
	}
}

// TestMembership_CanKick_TableDriven — матрица actor.role × target.role из 04-testing.md.
func TestMembership_CanKick_TableDriven(t *testing.T) {
	t.Parallel()

	cases := []struct {
		actor  domain.Role
		target domain.Role
		want   bool
	}{
		{domain.RoleOwner, domain.RoleOwner, false},
		{domain.RoleOwner, domain.RoleAdmin, true},
		{domain.RoleOwner, domain.RoleMember, true},
		{domain.RoleAdmin, domain.RoleOwner, false},
		{domain.RoleAdmin, domain.RoleAdmin, false},
		{domain.RoleAdmin, domain.RoleMember, true},
		{domain.RoleMember, domain.RoleOwner, false},
		{domain.RoleMember, domain.RoleAdmin, false},
		{domain.RoleMember, domain.RoleMember, false},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.actor.String()+"_kicks_"+tc.target.String(), func(t *testing.T) {
			t.Parallel()
			actor := NewMembershipBuilder(t).WithRole(tc.actor).Build()
			target := NewMembershipBuilder(t).WithRole(tc.target).Build()
			if got := actor.CanKick(*target); got != tc.want {
				t.Fatalf("CanKick(actor=%s, target=%s) = %v, want %v",
					tc.actor, tc.target, got, tc.want)
			}
		})
	}
}
