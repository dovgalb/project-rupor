package postgres

import (
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/dovgalb/project-rupor/internal/channel/domain"
	"github.com/dovgalb/project-rupor/internal/channel/repository/postgres/db"
)

func mustChannelID(t *testing.T, raw uuid.UUID) domain.ChannelID {
	t.Helper()
	id, err := domain.NewChannelID(raw)
	if err != nil {
		t.Fatalf("NewChannelID: %v", err)
	}
	return id
}

func mustRoomID(t *testing.T, raw uuid.UUID) domain.RoomID {
	t.Helper()
	id, err := domain.NewRoomID(raw)
	if err != nil {
		t.Fatalf("NewRoomID: %v", err)
	}
	return id
}

func mustChannelName(t *testing.T, raw string) domain.ChannelName {
	t.Helper()
	n, err := domain.NewChannelName(raw)
	if err != nil {
		t.Fatalf("NewChannelName: %v", err)
	}
	return n
}

func TestChannelMapper_RoundTrip_BothKinds(t *testing.T) {
	t.Parallel()

	createdAt := time.Date(2026, 5, 11, 12, 0, 0, 0, time.UTC)
	cases := []struct {
		name string
		kind domain.ChannelKind
	}{
		{"text", domain.ChannelKindText},
		{"voice", domain.ChannelKindVoice},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			src, err := domain.NewChannel(
				mustChannelID(t, uuid.New()),
				mustRoomID(t, uuid.New()),
				mustChannelName(t, "general"),
				tc.kind,
				createdAt,
			)
			if err != nil {
				t.Fatalf("NewChannel: %v", err)
			}
			params := domainToInsertChannelParams(src)
			row := db.Channel(params)

			got, err := channelRowToDomain(row)
			if err != nil {
				t.Fatalf("channelRowToDomain: %v", err)
			}
			if got.ID() != src.ID() {
				t.Fatalf("ID mismatch")
			}
			if got.RoomID() != src.RoomID() {
				t.Fatalf("RoomID mismatch")
			}
			if got.Name() != src.Name() {
				t.Fatalf("Name mismatch")
			}
			if got.Kind() != src.Kind() {
				t.Fatalf("Kind mismatch")
			}
			if !got.CreatedAt().Equal(src.CreatedAt()) {
				t.Fatalf("CreatedAt mismatch")
			}
		})
	}
}
