package domain_test

import (
	"errors"
	"testing"

	"github.com/dovgalb/project-rupor/internal/channel/domain"
)

func TestParseChannelKind_Valid_TableDriven(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name  string
		input string
		want  domain.ChannelKind
	}{
		{"text", "text", domain.ChannelKindText},
		{"voice", "voice", domain.ChannelKindVoice},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := domain.ParseChannelKind(tc.input)
			if err != nil {
				t.Fatalf("ParseChannelKind(%q): %v", tc.input, err)
			}
			if got != tc.want {
				t.Fatalf("ParseChannelKind(%q) = %v, want %v", tc.input, got, tc.want)
			}
		})
	}
}

func TestParseChannelKind_Invalid_ReturnsErr(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name  string
		input string
	}{
		{"empty", ""},
		{"video", "video"},
		{"Text (mixed case)", "Text"},
		{"TEXT (upper)", "TEXT"},
		{"voice with space", "voice "},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, err := domain.ParseChannelKind(tc.input)
			if !errors.Is(err, domain.ErrInvalidChannelKind) {
				t.Fatalf("got %v, want ErrInvalidChannelKind", err)
			}
		})
	}
}

func TestChannelKind_String_TableDriven(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		kind domain.ChannelKind
		want string
	}{
		{"text", domain.ChannelKindText, "text"},
		{"voice", domain.ChannelKindVoice, "voice"},
		{"invalid", domain.ChannelKind(99), "invalid"},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := tc.kind.String(); got != tc.want {
				t.Fatalf("ChannelKind(%d).String() = %q, want %q", tc.kind, got, tc.want)
			}
		})
	}
}

func TestChannelKind_RoundTrip(t *testing.T) {
	t.Parallel()

	kinds := []domain.ChannelKind{domain.ChannelKindText, domain.ChannelKindVoice}
	for _, k := range kinds {
		k := k
		t.Run(k.String(), func(t *testing.T) {
			t.Parallel()
			parsed, err := domain.ParseChannelKind(k.String())
			if err != nil {
				t.Fatalf("ParseChannelKind(%q): %v", k.String(), err)
			}
			if parsed != k {
				t.Fatalf("round-trip: got %v, want %v", parsed, k)
			}
		})
	}
}

func TestChannelKind_IsValid(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		kind domain.ChannelKind
		want bool
	}{
		{"text", domain.ChannelKindText, true},
		{"voice", domain.ChannelKindVoice, true},
		{"unknown", domain.ChannelKind(42), false},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := tc.kind.IsValid(); got != tc.want {
				t.Fatalf("ChannelKind(%d).IsValid() = %v, want %v", tc.kind, got, tc.want)
			}
		})
	}
}
