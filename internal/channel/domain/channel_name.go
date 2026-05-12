package domain

import (
	"strings"
	"unicode"
	"unicode/utf8"
)

const (
	minChannelNameLength = 1
	maxChannelNameLength = 64
)

type ChannelName struct{ value string }

func NewChannelName(raw string) (ChannelName, error) {
	s := strings.TrimSpace(raw)
	// Длина в рунах.
	n := utf8.RuneCountInString(s)
	if n < minChannelNameLength || n > maxChannelNameLength {
		return ChannelName{}, ErrInvalidChannelName
	}
	for _, r := range s {
		// Cc (control) и Cf (format) — защита от управляющих символов
		// и UI-spoofing (ZWSP, RLO, BOM, LRM).
		if unicode.IsControl(r) || unicode.In(r, unicode.Cf) {
			return ChannelName{}, ErrInvalidChannelName
		}
	}
	return ChannelName{value: s}, nil
}

func (n ChannelName) String() string { return n.value }
