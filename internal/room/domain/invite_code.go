package domain

import "strings"

const (
	inviteCodeLength  = 8
	crockfordAlphabet = "0123456789ABCDEFGHJKMNPQRSTVWXYZ"
)

type InviteCode struct{ value string }

func NewInviteCode(raw string) (InviteCode, error) {
	s := strings.ToUpper(strings.TrimSpace(raw))
	if len(s) != inviteCodeLength {
		return InviteCode{}, ErrInvalidInviteCode
	}
	for i := 0; i < len(s); i++ {
		if !strings.ContainsRune(crockfordAlphabet, rune(s[i])) {
			return InviteCode{}, ErrInvalidInviteCode
		}
	}
	return InviteCode{value: s}, nil
}

func (c InviteCode) String() string { return c.value }

// InviteCodeAlphabet возвращает алфавит Crockford base32 без I/L/O/U.
// Используется генераторами кодов и тестами.
func InviteCodeAlphabet() string { return crockfordAlphabet }
