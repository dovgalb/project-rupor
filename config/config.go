package config

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	defaultServerPort        = 8080
	defaultJWTAccessTTL      = 15 * time.Minute
	defaultJWTRefreshTTL     = 720 * time.Hour
	defaultCORSAllowedOrigin = "http://localhost:5173"

	maxJWTAccessTTL  = time.Hour
	maxJWTRefreshTTL = 90 * 24 * time.Hour
)

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
	serverPort         int
	databaseURL        string
	jwtSecret          string
	jwtAccessTTL       time.Duration
	jwtRefreshTTL      time.Duration
	corsAllowedOrigins []string
}

func (c *Config) ServerPort() int              { return c.serverPort }
func (c *Config) DatabaseURL() string          { return c.databaseURL }
func (c *Config) JWTSecret() string            { return c.jwtSecret }
func (c *Config) JWTAccessTTL() time.Duration  { return c.jwtAccessTTL }
func (c *Config) JWTRefreshTTL() time.Duration { return c.jwtRefreshTTL }

// CORSAllowedOrigins возвращает копию whitelisted origins.
func (c *Config) CORSAllowedOrigins() []string {
	cp := make([]string, len(c.corsAllowedOrigins))
	copy(cp, c.corsAllowedOrigins)
	return cp
}

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

	accessTTL, err := loadAccessTTL(l)
	if err != nil {
		return nil, err
	}

	refreshTTL, err := loadRefreshTTL(l, accessTTL)
	if err != nil {
		return nil, err
	}

	corsOrigins, err := loadCORSAllowedOrigins(l)
	if err != nil {
		return nil, err
	}

	return &Config{
		serverPort:         port,
		databaseURL:        dbURL,
		jwtSecret:          jwt,
		jwtAccessTTL:       accessTTL,
		jwtRefreshTTL:      refreshTTL,
		corsAllowedOrigins: corsOrigins,
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

func loadAccessTTL(l Lookuper) (time.Duration, error) {
	raw, ok := l.Lookup("JWT_ACCESS_TTL")
	if !ok || raw == "" {
		return defaultJWTAccessTTL, nil
	}

	d, err := time.ParseDuration(raw)
	if err != nil {
		return 0, ValidationError{
			Code:   "CONFIG-004",
			Field:  "JWT_ACCESS_TTL",
			Reason: fmt.Sprintf("must be a duration, got %q", raw),
		}
	}

	if d <= 0 {
		return 0, ValidationError{
			Code:   "CONFIG-004",
			Field:  "JWT_ACCESS_TTL",
			Reason: fmt.Sprintf("must be positive, got %s", d),
		}
	}

	if d > maxJWTAccessTTL {
		return 0, ValidationError{
			Code:   "CONFIG-004",
			Field:  "JWT_ACCESS_TTL",
			Reason: fmt.Sprintf("must be <= %s, got %s", maxJWTAccessTTL, d),
		}
	}

	return d, nil
}

func loadRefreshTTL(l Lookuper, accessTTL time.Duration) (time.Duration, error) {
	raw, ok := l.Lookup("JWT_REFRESH_TTL")
	if !ok || raw == "" {
		if defaultJWTRefreshTTL <= accessTTL {
			return 0, ValidationError{
				Code:   "CONFIG-005",
				Field:  "JWT_REFRESH_TTL",
				Reason: fmt.Sprintf("default %s must be greater than JWT_ACCESS_TTL %s", defaultJWTRefreshTTL, accessTTL),
			}
		}
		return defaultJWTRefreshTTL, nil
	}

	d, err := time.ParseDuration(raw)
	if err != nil {
		return 0, ValidationError{
			Code:   "CONFIG-005",
			Field:  "JWT_REFRESH_TTL",
			Reason: fmt.Sprintf("must be a duration, got %q", raw),
		}
	}

	if d <= 0 {
		return 0, ValidationError{
			Code:   "CONFIG-005",
			Field:  "JWT_REFRESH_TTL",
			Reason: fmt.Sprintf("must be positive, got %s", d),
		}
	}

	if d <= accessTTL {
		return 0, ValidationError{
			Code:   "CONFIG-005",
			Field:  "JWT_REFRESH_TTL",
			Reason: fmt.Sprintf("must be greater than JWT_ACCESS_TTL %s, got %s", accessTTL, d),
		}
	}

	if d > maxJWTRefreshTTL {
		return 0, ValidationError{
			Code:   "CONFIG-005",
			Field:  "JWT_REFRESH_TTL",
			Reason: fmt.Sprintf("must be <= %s, got %s", maxJWTRefreshTTL, d),
		}
	}

	return d, nil
}

func loadCORSAllowedOrigins(l Lookuper) ([]string, error) {
	raw, ok := l.Lookup("CORS_ALLOWED_ORIGINS")
	if !ok || strings.TrimSpace(raw) == "" {
		return []string{defaultCORSAllowedOrigin}, nil
	}

	parts := strings.Split(raw, ",")
	origins := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		if err := validateOrigin(p); err != nil {
			return nil, ValidationError{
				Code:   "CONFIG-006",
				Field:  "CORS_ALLOWED_ORIGINS",
				Reason: fmt.Sprintf("invalid origin %q: %s", p, err.Error()),
			}
		}
		origins = append(origins, p)
	}

	if len(origins) == 0 {
		return nil, ValidationError{
			Code:   "CONFIG-006",
			Field:  "CORS_ALLOWED_ORIGINS",
			Reason: "must contain at least one origin",
		}
	}

	return origins, nil
}

func validateOrigin(s string) error {
	u, err := url.Parse(s)
	if err != nil {
		return err
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return fmt.Errorf("scheme must be http or https, got %q", u.Scheme)
	}
	if u.Host == "" {
		return fmt.Errorf("host is empty")
	}
	if u.Path != "" && u.Path != "/" {
		return fmt.Errorf("must not contain path, got %q", u.Path)
	}
	if u.RawQuery != "" || u.Fragment != "" {
		return fmt.Errorf("must not contain query or fragment")
	}
	return nil
}
