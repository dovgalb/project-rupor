package domain

type Role uint8

const (
	RoleMember Role = iota
	RoleAdmin
	RoleOwner
)

const (
	roleStringMember = "member"
	roleStringAdmin  = "admin"
	roleStringOwner  = "owner"
	roleStringNone   = "invalid"
)

func (r Role) String() string {
	switch r {
	case RoleMember:
		return roleStringMember
	case RoleAdmin:
		return roleStringAdmin
	case RoleOwner:
		return roleStringOwner
	default:
		return roleStringNone
	}
}

func (r Role) IsValid() bool {
	return r == RoleMember || r == RoleAdmin || r == RoleOwner
}

func ParseRole(raw string) (Role, error) {
	switch raw {
	case roleStringMember:
		return RoleMember, nil
	case roleStringAdmin:
		return RoleAdmin, nil
	case roleStringOwner:
		return RoleOwner, nil
	default:
		return 0, ErrInvalidRole
	}
}
