package domain

import (
	"strings"
	"unicode"
	"unicode/utf8"
)

// Границы длины текста сообщения в рунах после TrimSpace.
const (
	MessageTextMinLen = 1
	MessageTextMaxLen = 4000
)

// MessageText — value object текста сообщения с инвариантами по длине и допустимым символам.
type MessageText struct{ value string }

// NewMessageText нормализует входную строку (TrimSpace) и проверяет инварианты:
// длина в [MessageTextMinLen..MessageTextMaxLen] и отсутствие управляющих/форматных символов
// кроме \n, \r, \t.
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

// String возвращает нормализованный текст сообщения.
func (t MessageText) String() string { return t.value }
