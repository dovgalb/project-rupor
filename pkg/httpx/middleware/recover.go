package middleware

import (
	"errors"
	"log/slog"
	"net/http"
	"runtime/debug"

	"github.com/dovgalb/project-rupor/pkg/httpx"
)

// Recover ловит panic в downstream-хендлерах, логирует с request_id и stack trace,
// отвечает 500 INTERNAL. http.ErrAbortHandler пере-paniкуется без обработки.
func Recover(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			sw := wrap(w)

			defer func() {
				v := recover()
				if v == nil {
					return
				}
				if e, ok := v.(error); ok && errors.Is(e, http.ErrAbortHandler) {
					panic(v)
				}

				requestID, _ := httpx.RequestIDFromContext(r.Context())
				logger.Error("panic recovered",
					slog.Any("err", v),
					slog.String("stack", string(debug.Stack())),
					slog.String("request_id", requestID),
					slog.String("method", r.Method),
					slog.String("path", r.URL.Path),
				)

				if !sw.headerWritten {
					httpx.WriteJSONError(sw, http.StatusInternalServerError, "INTERNAL", "internal")
					return
				}
				logger.Warn("panic after WriteHeader, response truncated",
					slog.String("request_id", requestID),
				)
			}()

			next.ServeHTTP(sw, r)
		})
	}
}
