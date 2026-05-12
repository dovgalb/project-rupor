package domain

import "time"

type Membership struct {
	roomID   RoomID
	userID   UserID
	role     Role
	joinedAt time.Time
}

func NewMembership(roomID RoomID, userID UserID, role Role, joinedAt time.Time) (*Membership, error) {
	if roomID.IsZero() {
		return nil, ErrInvalidRoomID
	}
	if userID.IsZero() {
		return nil, ErrInvalidUserID
	}
	if !role.IsValid() {
		return nil, ErrInvalidRole
	}
	if joinedAt.IsZero() {
		return nil, ErrInvalidCreatedAt
	}
	return &Membership{
		roomID:   roomID,
		userID:   userID,
		role:     role,
		joinedAt: joinedAt,
	}, nil
}

func ReconstructMembership(roomID RoomID, userID UserID, role Role, joinedAt time.Time) (*Membership, error) {
	return NewMembership(roomID, userID, role, joinedAt)
}

func (m *Membership) RoomID() RoomID      { return m.roomID }
func (m *Membership) UserID() UserID      { return m.userID }
func (m *Membership) Role() Role          { return m.role }
func (m *Membership) JoinedAt() time.Time { return m.joinedAt }

// Promote повышает member → admin. Для admin/owner — no-op (без ошибки).
func (m *Membership) Promote() error {
	if m.role == RoleMember {
		m.role = RoleAdmin
	}
	return nil
}

// Demote понижает admin → member. Для member — no-op. Owner понизить нельзя.
func (m *Membership) Demote() error {
	switch m.role {
	case RoleAdmin:
		m.role = RoleMember
		return nil
	case RoleMember:
		return nil
	case RoleOwner:
		return ErrCannotDemoteOwner
	default:
		return ErrInvalidRole
	}
}

func (m *Membership) CanReadRoom() bool     { return m.role.IsValid() }
func (m *Membership) CanReadMembers() bool  { return m.role.IsValid() }
func (m *Membership) CanReadChannels() bool { return m.role.IsValid() }

func (m *Membership) CanCreateChannel() bool {
	return m.role == RoleAdmin || m.role == RoleOwner
}

func (m *Membership) CanDeleteChannel() bool {
	return m.role == RoleAdmin || m.role == RoleOwner
}

func (m *Membership) CanGenerateInvite() bool {
	return m.role == RoleAdmin || m.role == RoleOwner
}

func (m *Membership) CanDeleteRoom() bool {
	return m.role == RoleOwner
}

// CanKick: owner может kick admin/member; admin — только member; member — никого.
// Owner-on-owner (себя) — запрещено.
func (m *Membership) CanKick(target Membership) bool {
	switch m.role {
	case RoleOwner:
		return target.role == RoleAdmin || target.role == RoleMember
	case RoleAdmin:
		return target.role == RoleMember
	default:
		return false
	}
}
