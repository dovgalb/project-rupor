package config_test

import (
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/dovgalb/project-rupor/config"
)

type mapLookuper map[string]string

func (m mapLookuper) Lookup(key string) (string, bool) {
	v, ok := m[key]
	return v, ok
}

func validEnv() mapLookuper {
	return mapLookuper{
		"JWT_SECRET":   "super-secret",
		"DATABASE_URL": "postgres://user:pass@localhost:5432/db?sslmode=disable",
		"SERVER_PORT":  "8080",
	}
}

func TestConfig_Load_Valid(t *testing.T) {
	t.Parallel()

	cfg, err := config.Load(validEnv())
	if err != nil {
		t.Fatalf("Load: unexpected error: %v", err)
	}
	if cfg.ServerPort() != 8080 {
		t.Fatalf("ServerPort = %d, want 8080", cfg.ServerPort())
	}
	if cfg.DatabaseURL() != "postgres://user:pass@localhost:5432/db?sslmode=disable" {
		t.Fatalf("DatabaseURL = %q", cfg.DatabaseURL())
	}
	if cfg.JWTSecret() != "super-secret" {
		t.Fatalf("JWTSecret = %q", cfg.JWTSecret())
	}
}

func TestConfig_Load_DefaultPort(t *testing.T) {
	t.Parallel()

	env := validEnv()
	delete(env, "SERVER_PORT")

	cfg, err := config.Load(env)
	if err != nil {
		t.Fatalf("Load: unexpected error: %v", err)
	}
	if cfg.ServerPort() != 8080 {
		t.Fatalf("ServerPort = %d, want default 8080", cfg.ServerPort())
	}
}

func TestConfig_Load_MissingJWTSecret(t *testing.T) {
	t.Parallel()

	env := validEnv()
	delete(env, "JWT_SECRET")

	_, err := config.Load(env)
	if !errors.Is(err, config.ErrConfigInvalid) {
		t.Fatalf("err is not ErrConfigInvalid: %v", err)
	}
	var verr config.ValidationError
	if !errors.As(err, &verr) {
		t.Fatalf("err is not ValidationError: %v", err)
	}
	if verr.Code != "CONFIG-001" {
		t.Fatalf("Code = %q, want CONFIG-001", verr.Code)
	}
	if verr.Field != "JWT_SECRET" {
		t.Fatalf("Field = %q, want JWT_SECRET", verr.Field)
	}
}

func TestConfig_Load_MissingDatabaseURL(t *testing.T) {
	t.Parallel()

	env := validEnv()
	delete(env, "DATABASE_URL")

	_, err := config.Load(env)
	var verr config.ValidationError
	if !errors.As(err, &verr) {
		t.Fatalf("err is not ValidationError: %v", err)
	}
	if verr.Code != "CONFIG-002" {
		t.Fatalf("Code = %q, want CONFIG-002", verr.Code)
	}
}

func TestConfig_Load_InvalidPortFormat(t *testing.T) {
	t.Parallel()

	env := validEnv()
	env["SERVER_PORT"] = "not-a-number"

	_, err := config.Load(env)
	var verr config.ValidationError
	if !errors.As(err, &verr) {
		t.Fatalf("err is not ValidationError: %v", err)
	}
	if verr.Code != "CONFIG-003" {
		t.Fatalf("Code = %q, want CONFIG-003", verr.Code)
	}
	if verr.Field != "SERVER_PORT" {
		t.Fatalf("Field = %q", verr.Field)
	}
}

func TestConfig_Load_DefaultAccessTTL(t *testing.T) {
	t.Parallel()

	env := validEnv()
	delete(env, "JWT_ACCESS_TTL")

	cfg, err := config.Load(env)
	if err != nil {
		t.Fatalf("Load: unexpected error: %v", err)
	}
	if cfg.JWTAccessTTL() != 15*time.Minute {
		t.Fatalf("JWTAccessTTL = %s, want 15m", cfg.JWTAccessTTL())
	}
}

func TestConfig_Load_DefaultRefreshTTL(t *testing.T) {
	t.Parallel()

	env := validEnv()
	delete(env, "JWT_REFRESH_TTL")

	cfg, err := config.Load(env)
	if err != nil {
		t.Fatalf("Load: unexpected error: %v", err)
	}
	if cfg.JWTRefreshTTL() != 720*time.Hour {
		t.Fatalf("JWTRefreshTTL = %s, want 720h", cfg.JWTRefreshTTL())
	}
}

func TestConfig_Load_InvalidAccessTTL(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		raw  string
	}{
		{"не парсится", "abc"},
		{"отрицательный", "-1m"},
		{"ноль", "0s"},
		{"больше верхней границы 1h", "2h"},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			env := validEnv()
			env["JWT_ACCESS_TTL"] = tc.raw

			_, err := config.Load(env)
			if !errors.Is(err, config.ErrConfigInvalid) {
				t.Fatalf("raw %q: err is not ErrConfigInvalid: %v", tc.raw, err)
			}
			var verr config.ValidationError
			if !errors.As(err, &verr) {
				t.Fatalf("raw %q: err is not ValidationError: %v", tc.raw, err)
			}
			if verr.Code != "CONFIG-004" {
				t.Fatalf("raw %q: Code = %q, want CONFIG-004", tc.raw, verr.Code)
			}
		})
	}
}

func TestConfig_Load_InvalidRefreshTTL(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		raw  string
	}{
		{"не парсится", "abc"},
		{"отрицательный", "-1h"},
		{"больше верхней границы 90d", "2400h"},
		{"меньше access default 15m", "1m"},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			env := validEnv()
			env["JWT_REFRESH_TTL"] = tc.raw

			_, err := config.Load(env)
			if !errors.Is(err, config.ErrConfigInvalid) {
				t.Fatalf("raw %q: err is not ErrConfigInvalid: %v", tc.raw, err)
			}
			var verr config.ValidationError
			if !errors.As(err, &verr) {
				t.Fatalf("raw %q: err is not ValidationError: %v", tc.raw, err)
			}
			if verr.Code != "CONFIG-005" {
				t.Fatalf("raw %q: Code = %q, want CONFIG-005", tc.raw, verr.Code)
			}
		})
	}
}

func TestConfig_Load_PortOutOfRange(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		port string
	}{
		{"ноль", "0"},
		{"отрицательный", "-1"},
		{"сразу выше диапазона", "65536"},
		{"сильно выше", "100000"},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			env := validEnv()
			env["SERVER_PORT"] = tc.port

			_, err := config.Load(env)
			var verr config.ValidationError
			if !errors.As(err, &verr) {
				t.Fatalf("port %q: err is not ValidationError: %v", tc.port, err)
			}
			if verr.Code != "CONFIG-003" {
				t.Fatalf("port %q: Code = %q, want CONFIG-003", tc.port, verr.Code)
			}
		})
	}
}

func TestConfig_Load_DefaultCORS(t *testing.T) {
	t.Parallel()

	cfg, err := config.Load(validEnv())
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	got := cfg.CORSAllowedOrigins()
	want := []string{"http://localhost:5173"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("CORSAllowedOrigins() = %v, want %v", got, want)
	}
}

func TestConfig_Load_CSVOrigins(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		env  string
		want []string
	}{
		{"single", "http://a.com", []string{"http://a.com"}},
		{"two", "http://a.com,https://b.com:8080", []string{"http://a.com", "https://b.com:8080"}},
		{"with_spaces", "http://a.com , https://b.com", []string{"http://a.com", "https://b.com"}},
		{"trailing_comma", "http://a.com,", []string{"http://a.com"}},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			env := validEnv()
			env["CORS_ALLOWED_ORIGINS"] = tc.env
			cfg, err := config.Load(env)
			if err != nil {
				t.Fatalf("Load: %v", err)
			}
			got := cfg.CORSAllowedOrigins()
			if !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("got %v, want %v", got, tc.want)
			}
		})
	}
}

func TestConfig_Load_EmptyCORS(t *testing.T) {
	t.Parallel()

	cases := []struct{ name, env string }{
		{"only_comma", ","},
		{"only_commas_and_spaces", " , , "},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			env := validEnv()
			env["CORS_ALLOWED_ORIGINS"] = tc.env
			_, err := config.Load(env)
			var verr config.ValidationError
			if !errors.As(err, &verr) {
				t.Fatalf("err = %v, want ValidationError", err)
			}
			if verr.Code != "CONFIG-006" {
				t.Fatalf("Code = %q, want CONFIG-006", verr.Code)
			}
			if verr.Field != "CORS_ALLOWED_ORIGINS" {
				t.Fatalf("Field = %q", verr.Field)
			}
		})
	}
}

func TestConfig_Load_InvalidOrigin(t *testing.T) {
	t.Parallel()

	cases := []struct{ name, value string }{
		{"no_scheme", "localhost:5173"},
		{"with_path", "http://x.com/path"},
		{"with_query", "http://x.com?foo=bar"},
		{"ftp_scheme", "ftp://x.com"},
		{"empty_host", "http://"},
		{"mixed_valid_invalid", "http://x.com,not-a-url"},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			env := validEnv()
			env["CORS_ALLOWED_ORIGINS"] = tc.value
			_, err := config.Load(env)
			var verr config.ValidationError
			if !errors.As(err, &verr) {
				t.Fatalf("err = %v, want ValidationError", err)
			}
			if verr.Code != "CONFIG-006" {
				t.Fatalf("Code = %q, want CONFIG-006", verr.Code)
			}
		})
	}
}
