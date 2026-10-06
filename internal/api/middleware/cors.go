package middleware

import (
	"net/http"
	"strings"
)

func CORS(allowedOrigins []string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := strings.TrimSuffix(r.Header.Get("Origin"), "/")
			
			allowOrigin := ""
			for _, o := range allowedOrigins {
				o = strings.TrimSuffix(o, "/")
				if o == "*" {
					allowOrigin = origin
					break
				}
				if o == origin {
					allowOrigin = o
					break
				}
			}
				if o == origin {
					allowOrigin = o
					break
				}
			}

			// Special handling for Tailscale origins if not explicitly listed
			if allowOrigin == "" && origin != "" {
				if strings.HasSuffix(origin, ".ts.net") || 
				   strings.Contains(origin, "hornet") || 
				   strings.Contains(origin, "localhost") {
					allowOrigin = origin
				}
			}

			if allowOrigin != "" {
				w.Header().Set("Access-Control-Allow-Origin", allowOrigin)
				w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS, PATCH")
				w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
				w.Header().Set("Access-Control-Allow-Credentials", "true")
			}

			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
