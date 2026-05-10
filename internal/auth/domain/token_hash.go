package domain

const tokenHashSize = 32

type TokenHash struct{ value [tokenHashSize]byte }

func NewTokenHash(raw []byte) (TokenHash, error) {
	if len(raw) != tokenHashSize {
		return TokenHash{}, ErrInvalidTokenHash
	}
	var h TokenHash
	copy(h.value[:], raw)
	return h, nil
}

func (h TokenHash) Bytes() [tokenHashSize]byte { return h.value }

func (h TokenHash) Equal(o TokenHash) bool { return h.value == o.value }
