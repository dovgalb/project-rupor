package domain

import (
	"strings"
	"unicode"
	"unicode/utf8"
)

const (
	minRoomNameLength = 1
	maxRoomNameLength = 64
)

type RoomName struct{ value string }

func NewRoomName(raw string) (RoomName, error) {
	s := strings.TrimSpace(raw)
	// Длина считается в рунах: 1..64 (см. 01-architecture.md §3.1.2).
	n := utf8.RuneCountInString(s)
	if n < minRoomNameLength || n > maxRoomNameLength {
		return RoomName{}, ErrInvalidRoomName
	}
	for _, r := range s {
		// Cc (control) ловит \x00..\x1F, \x7F..\x9F.
		// Cf (format) дополнительно ловит невидимые/направляющие коды (ZWSP, RLO, BOM, LRM)
		// — защита от UI-spoofing и невидимых дублей имён.
		if unicode.IsControl(r) || unicode.In(r, unicode.Cf) {
			return RoomName{}, ErrInvalidRoomName
		}
	}
	return RoomName{value: s}, nil
}

func (n RoomName) String() string { return n.value }
