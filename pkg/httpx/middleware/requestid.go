package middleware

import (
	"net/http"

	"github.com/dovgalb/project-rupor/pkg/httpx"
)

const (
	requestIDHeader    = "X-Request-ID"
	requestIDMaxLength = 128
)

// RequestID — middleware, обеспечивающий request_id в каждом запросе.
// Принимает клиентский X-Request-ID при валидности (ASCII-printable, len<=128), иначе генерирует через uuidGen.
func RequestID(uuidGen func() string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			incoming := r.Header.Get(requestIDHeader)
			rid := normalizeRequestID(incoming, uuidGen)

			w.Header().Set(requestIDHeader, rid)
			ctx := httpx.WithRequestID(r.Context(), rid)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func normalizeRequestID(incoming string, uuidGen func() string) string {
	if isValidRequestID(incoming) {
		return incoming
	}
	return uuidGen()
}

func isValidRequestID(s string) bool {
	if s == "" || len(s) > requestIDMaxLength {
		return false
	}
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c < 0x20 || c > 0x7E {
			return false
		}
	}
	return true
}
