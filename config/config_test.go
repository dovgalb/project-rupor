package config_test

import (
	"errors"
	"testing"

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
