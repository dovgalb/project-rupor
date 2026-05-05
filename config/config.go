package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
)

const defaultServerPort = 8080

var ErrConfigInvalid = errors.New("config: invalid")

type ValidationError struct {
	Code   string
	Field  string
	Reason string
}

func (e ValidationError) Error() string {
	return fmt.Sprintf("config: %s: %s (%s)", e.Field, e.Reason, e.Code)
}

func (e ValidationError) Is(target error) bool {
	return target == ErrConfigInvalid
}

type Config struct {
	serverPort  int
	databaseURL string
	jwtSecret   string
}

func (c *Config) ServerPort() int     { return c.serverPort }
func (c *Config) DatabaseURL() string { return c.databaseURL }
func (c *Config) JWTSecret() string   { return c.jwtSecret }

type Lookuper interface {
	Lookup(key string) (string, bool)
}

type OsLookuper struct{}

func (OsLookuper) Lookup(key string) (string, bool) {
	return os.LookupEnv(key)
}

func Load(l Lookuper) (*Config, error) {
	jwt, ok := l.Lookup("JWT_SECRET")
	if !ok || jwt == "" {
		return nil, ValidationError{
			Code:   "CONFIG-001",
			Field:  "JWT_SECRET",
			Reason: "required",
		}
	}

	dbURL, ok := l.Lookup("DATABASE_URL")
	if !ok || dbURL == "" {
		return nil, ValidationError{
			Code:   "CONFIG-002",
			Field:  "DATABASE_URL",
			Reason: "required",
		}
	}

	port, err := loadPort(l)
	if err != nil {
		return nil, err
	}

	return &Config{
		serverPort:  port,
		databaseURL: dbURL,
		jwtSecret:   jwt,
	}, nil
}

func loadPort(l Lookuper) (int, error) {
	raw, ok := l.Lookup("SERVER_PORT")
	if !ok || raw == "" {
		return defaultServerPort, nil
	}

	port, err := strconv.Atoi(raw)
	if err != nil {
		return 0, ValidationError{
			Code:   "CONFIG-003",
			Field:  "SERVER_PORT",
			Reason: fmt.Sprintf("must be integer, got %q", raw),
		}
	}

	if port < 1 || port > 65535 {
		return 0, ValidationError{
			Code:   "CONFIG-003",
			Field:  "SERVER_PORT",
			Reason: fmt.Sprintf("must be in 1..65535, got %d", port),
		}
	}

	return port, nil
}
