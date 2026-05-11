package middleware

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/dovgalb/project-rupor/pkg/httpx"
)

// Logger — access-log через slog. Принимает hook для дополнительных attrs (например, user_id).
func Logger(
	logger *slog.Logger,
	hook func(ctx context.Context) []slog.Attr,
) func(http.Handler) http.Handler {
	if hook == nil {
		hook = func(context.Context) []slog.Attr { return nil }
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			sw := wrap(w)
			next.ServeHTTP(sw, r)

			requestID, _ := httpx.RequestIDFromContext(r.Context())
			status := sw.status
			if status == 0 {
				status = http.StatusOK
			}

			attrs := []slog.Attr{
				slog.String("method", r.Method),
				slog.String("path", r.URL.Path),
				slog.Int("status", status),
				slog.Duration("duration", time.Since(start)),
				slog.String("request_id", requestID),
				slog.String("remote_ip", r.RemoteAddr),
				slog.String("user_agent", r.UserAgent()),
			}
			if extra := hook(r.Context()); len(extra) > 0 {
				attrs = append(attrs, extra...)
			}
			logger.LogAttrs(r.Context(), slog.LevelInfo, "http", attrs...)
		})
	}
}
