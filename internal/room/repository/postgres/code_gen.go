package postgres

import (
	"fmt"
	"io"

	"github.com/dovgalb/project-rupor/internal/room/domain"
)

const inviteCodeLength = 8

// Base32CodeGen — генератор инвайт-кодов Crockford base32 длины 8.
// Принимает io.Reader (в проде — crypto/rand.Reader); в тестах — bytes.Reader.
// modulo bias на 32-буквенном алфавите принят сознательно (см. 03-decisions.md §D-14).
type Base32CodeGen struct {
	rand io.Reader
}

func NewBase32CodeGen(rand io.Reader) *Base32CodeGen {
	return &Base32CodeGen{rand: rand}
}

func (g *Base32CodeGen) New() (domain.InviteCode, error) {
	alphabet := domain.InviteCodeAlphabet()
	buf := make([]byte, inviteCodeLength)
	if _, err := io.ReadFull(g.rand, buf); err != nil {
		return domain.InviteCode{}, fmt.Errorf("code gen: read rand: %w", err)
	}
	out := make([]byte, inviteCodeLength)
	for i, b := range buf {
		out[i] = alphabet[int(b)%len(alphabet)]
	}
	return domain.NewInviteCode(string(out))
}
