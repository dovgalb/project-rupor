package domain

import (
	"regexp"
	"strings"
)

const (
	minUsernameLength = 3
	maxUsernameLength = 32
)

var usernameRe = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)

type Username struct{ value string }

func NewUsername(raw string) (Username, error) {
	s := strings.TrimSpace(raw)
	l := len(s)
	if l < minUsernameLength || l > maxUsernameLength {
		return Username{}, ErrInvalidUsername
	}
	if !usernameRe.MatchString(s) {
		return Username{}, ErrInvalidUsername
	}
	return Username{value: s}, nil
}

func (u Username) String() string { return u.value }
