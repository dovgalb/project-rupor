package domain

import (
	"strings"
	"unicode"
	"unicode/utf8"
)

const (
	MessageTextMinLen = 1
	MessageTextMaxLen = 4000
)

type MessageText struct{ value string }

func NewMessageText(raw string) (MessageText, error) {
	s := strings.TrimSpace(raw)
	n := utf8.RuneCountInString(s)
	if n < MessageTextMinLen || n > MessageTextMaxLen {
		return MessageText{}, ErrInvalidMessageText
	}
	for _, r := range s {
		// Cc (control) и Cf (format) запрещены, кроме whitespace \n \r \t.
		if unicode.IsControl(r) {
			if r == '\n' || r == '\r' || r == '\t' {
				continue
			}
			return MessageText{}, ErrInvalidMessageText
		}
		if unicode.In(r, unicode.Cf) {
			return MessageText{}, ErrInvalidMessageText
		}
	}
	return MessageText{value: s}, nil
}

func (t MessageText) String() string { return t.value }
