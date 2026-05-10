package domain

import "time"

type User struct {
	id           UserID
	email        Email
	username     Username
	passwordHash PasswordHash
	createdAt    time.Time
}

func NewUser(id UserID, email Email, username Username, hash PasswordHash, createdAt time.Time) (*User, error) {
	if id.IsZero() {
		return nil, ErrInvalidUserID
	}
	if createdAt.IsZero() {
		return nil, ErrInvalidCreatedAt
	}
	return &User{
		id:           id,
		email:        email,
		username:     username,
		passwordHash: hash,
		createdAt:    createdAt,
	}, nil
}

func ReconstructUser(id UserID, email Email, username Username, hash PasswordHash, createdAt time.Time) (*User, error) {
	return NewUser(id, email, username, hash, createdAt)
}

func (u *User) ID() UserID                 { return u.id }
func (u *User) Email() Email               { return u.email }
func (u *User) Username() Username         { return u.username }
func (u *User) PasswordHash() PasswordHash { return u.passwordHash }
func (u *User) CreatedAt() time.Time       { return u.createdAt }
