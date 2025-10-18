package middleware

import (
	"log/slog"
	"net/http"
	"runtime/debug"
	"time"
)

// Middleware is a function that wraps an http.Handler
type Middleware func(http.Handler) http.Handler

// Chain applies multiple middleware to a handler
func Chain(h http.Handler, middlewares ...Middleware) http.Handler {
	for i := len(middlewares) - 1; i >= 0; i-- {
		h = middlewares[i](h)
	}
	return h
}

// Logger logs HTTP requests
func Logger(log *slog.Logger) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			wrapped := &responseWriter{
				ResponseWriter: w,
				statusCode:     http.StatusOK,
			}

			next.ServeHTTP(wrapped, r)

			log.Info("request",
				"method", r.Method,
				"path", r.URL.Path,
				"status", wrapped.statusCode,
				"duration_ms", time.Since(start).Milliseconds(),
				"ip", r.RemoteAddr,
			)
		})
	}
}

// Recover recovers from panics and logs them
func Recover(log *slog.Logger) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if err := recover(); err != nil {
					log.Error("panic recovered",
						"error", err,
						"path", r.URL.Path,
						"method", r.Method,
						"stack", string(debug.Stack()),
					)

					http.Error(w, "internal server error", http.StatusInternalServerError)
				}
			}()

			next.ServeHTTP(w, r)
		})
	}
}

// CORSWithOrigins creates a CORS middleware with configurable allowed origins
func CORSWithOrigins(allowedOrigins []string) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")

			// Check if origin is allowed
			allowed := false
			for _, allowedOrigin := range allowedOrigins {
				if allowedOrigin == "*" || allowedOrigin == origin {
					allowed = true
					break
				}
			}

			// Set CORS headers if origin is allowed
			if allowed {
				if origin != "" && !contains(allowedOrigins, "*") {
					w.Header().Set("Access-Control-Allow-Origin", origin)
					w.Header().Set("Vary", "Origin")
				} else if contains(allowedOrigins, "*") {
					w.Header().Set("Access-Control-Allow-Origin", "*")
				}
				w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
				w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
			}

			if r.Method == http.MethodOptions {
				if allowed {
					w.WriteHeader(http.StatusNoContent)
				} else {
					w.WriteHeader(http.StatusForbidden)
				}
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// CORS is the default CORS middleware with wildcard origin (for backward compatibility)
// Deprecated: Use CORSWithOrigins instead
func CORS(next http.Handler) http.Handler {
	return CORSWithOrigins([]string{"*"})(next)
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// responseWriter wraps http.ResponseWriter to capture status code
type responseWriter struct {
	http.ResponseWriter
	statusCode int
	written    bool
}

func (rw *responseWriter) WriteHeader(code int) {
	if !rw.written {
		rw.statusCode = code
		rw.ResponseWriter.WriteHeader(code)
		rw.written = true
	}
}

func (rw *responseWriter) Write(b []byte) (int, error) {
	if !rw.written {
		rw.WriteHeader(http.StatusOK)
	}
	return rw.ResponseWriter.Write(b)
}
