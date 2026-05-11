package middleware

import "net/http"

const (
	corsAllowMethods = "GET, POST, PUT, DELETE, OPTIONS"
	corsAllowHeaders = "Authorization, Content-Type, X-Request-ID"
	corsMaxAge       = "600"
)

// CORS обрабатывает preflight OPTIONS и cross-origin запросы для whitelisted origins.
// Wildcard "*" не поддерживается — сравнение строгое.
func CORS(allowedOrigins []string, allowCredentials bool) func(http.Handler) http.Handler {
	set := makeOriginSet(allowedOrigins)

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")
			match := false
			if origin != "" {
				_, match = set[origin]
			}

			if match {
				w.Header().Set("Access-Control-Allow-Origin", origin)
				w.Header().Add("Vary", "Origin")
				if allowCredentials {
					w.Header().Set("Access-Control-Allow-Credentials", "true")
				}
			}

			if r.Method == http.MethodOptions && match {
				w.Header().Set("Access-Control-Allow-Methods", corsAllowMethods)
				w.Header().Set("Access-Control-Allow-Headers", corsAllowHeaders)
				w.Header().Set("Access-Control-Max-Age", corsMaxAge)
				w.WriteHeader(http.StatusNoContent)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func makeOriginSet(origins []string) map[string]struct{} {
	set := make(map[string]struct{}, len(origins))
	for _, o := range origins {
		set[o] = struct{}{}
	}
	return set
}
