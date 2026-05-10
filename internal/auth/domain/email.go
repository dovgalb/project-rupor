package domain

import (
	"regexp"
	"strings"
)

const maxEmailLength = 254

var emailRe = regexp.MustCompile(`^[^@\s]+@[^@\s]+\.[^@\s]+$`)

type Email struct{ value string }

func NewEmail(raw string) (Email, error) {
	s := strings.ToLower(strings.TrimSpace(raw))
	if s == "" || len(s) > maxEmailLength {
		return Email{}, ErrInvalidEmail
	}
	if !emailRe.MatchString(s) {
		return Email{}, ErrInvalidEmail
	}
	return Email{value: s}, nil
}

func (e Email) String() string { return e.value }
