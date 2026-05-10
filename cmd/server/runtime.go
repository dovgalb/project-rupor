package main

import (
	cryptorand "crypto/rand"
	"time"

	"github.com/google/uuid"
)

type realClock struct{}

func (realClock) Now() time.Time { return time.Now().UTC() }

type realUUID struct{}

func (realUUID) New() uuid.UUID { return uuid.New() }

type cryptoRand struct{}

func (cryptoRand) Read(n int) ([]byte, error) {
	b := make([]byte, n)
	if _, err := cryptorand.Read(b); err != nil {
		return nil, err
	}
	return b, nil
}
