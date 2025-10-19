package middleware

import (
	"context"
	"log/slog"
	"net/http"
	"runtime/debug"
	"strconv"
	"strings"
	"time"

	"github.com/yourorg/timeservice/pkg/auth"
	"github.com/yourorg/timeservice/pkg/metrics"
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

// Prometheus adds Prometheus metrics instrumentation to HTTP handlers
func Prometheus(m *metrics.Metrics) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			// Track in-flight requests
			m.HTTPRequestsInFlight.Inc()
			defer m.HTTPRequestsInFlight.Dec()

			// Wrap response writer to capture status code and response size
			wrapped := &metricsResponseWriter{
				ResponseWriter: w,
				statusCode:     http.StatusOK,
			}

			// Track request size
			requestSize := computeApproximateRequestSize(r)

			// Serve the request
			next.ServeHTTP(wrapped, r)

			// Record metrics
			duration := time.Since(start).Seconds()
			status := strconv.Itoa(wrapped.statusCode)
			path := r.URL.Path
			method := r.Method

			m.HTTPRequestsTotal.WithLabelValues(method, path, status).Inc()
			m.HTTPRequestDuration.WithLabelValues(method, path).Observe(duration)
			m.HTTPRequestSize.WithLabelValues(method, path).Observe(float64(requestSize))
			m.HTTPResponseSize.WithLabelValues(method, path).Observe(float64(wrapped.bytesWritten))
		})
	}
}

// computeApproximateRequestSize calculates approximate request size
func computeApproximateRequestSize(r *http.Request) int {
	size := 0

	// Request line (method + URI + proto)
	if r.URL != nil {
		size += len(r.Method) + len(r.URL.String()) + len(r.Proto) + 2 // +2 for spaces
	}

	// Headers
	for name, values := range r.Header {
		size += len(name) + len(": \r\n")
		for _, value := range values {
			size += len(value)
		}
	}

	// Body (if Content-Length is set and valid)
	// Note: r.ContentLength is -1 when not set (e.g., GET requests)
	if r.ContentLength > 0 {
		size += int(r.ContentLength)
	}

	return size
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

// metricsResponseWriter extends responseWriter to track response size for metrics
type metricsResponseWriter struct {
	http.ResponseWriter
	statusCode   int
	bytesWritten int
	written      bool
}

func (mrw *metricsResponseWriter) WriteHeader(code int) {
	if !mrw.written {
		mrw.statusCode = code
		mrw.ResponseWriter.WriteHeader(code)
		mrw.written = true
	}
}

func (mrw *metricsResponseWriter) Write(b []byte) (int, error) {
	if !mrw.written {
		mrw.WriteHeader(http.StatusOK)
	}
	n, err := mrw.ResponseWriter.Write(b)
	mrw.bytesWritten += n
	return n, err
}

// contextKey is a custom type for context keys to avoid collisions
type contextKey string

const (
	// ClaimsContextKey is the key for storing auth claims in the request context
	ClaimsContextKey contextKey = "auth_claims"
)

// AuthConfig holds configuration for the auth middleware
type AuthConfig struct {
	Enabled       bool
	Authenticator *auth.Authenticator
	PublicPaths   []string // Paths that don't require authentication
	Logger        *slog.Logger
	Metrics       *metrics.Metrics
}

// Auth creates an authentication middleware that validates JWT tokens
func Auth(cfg *AuthConfig) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// If auth is disabled, pass through
			if !cfg.Enabled {
				next.ServeHTTP(w, r)
				return
			}

			start := time.Now()
			path := r.URL.Path

			// Check if this is a public path
			if isPublicPath(path, cfg.PublicPaths) {
				cfg.Logger.Debug("public path, skipping auth", "path", path)
				next.ServeHTTP(w, r)
				return
			}

			// Extract token from Authorization header
			authHeader := r.Header.Get("Authorization")
			token, err := auth.ExtractBearerToken(authHeader)
			if err != nil {
				cfg.Logger.Debug("failed to extract bearer token",
					"error", err,
					"path", path,
					"ip", r.RemoteAddr,
				)
				if cfg.Metrics != nil {
					cfg.Metrics.AuthAttemptsTotal.WithLabelValues(path, "missing_token").Inc()
				}
				http.Error(w, `{"error": "missing or invalid authorization header"}`, http.StatusUnauthorized)
				return
			}

			// Verify token and check authorization
			claims, err := cfg.Authenticator.VerifyAndAuthorize(r.Context(), token)
			if err != nil {
				cfg.Logger.Warn("authentication failed",
					"error", err,
					"path", path,
					"ip", r.RemoteAddr,
				)

				// Determine error type for metrics
				status := "invalid_token"
				if strings.Contains(err.Error(), "missing required") {
					status = "forbidden"
				}

				if cfg.Metrics != nil {
					cfg.Metrics.AuthAttemptsTotal.WithLabelValues(path, status).Inc()
					cfg.Metrics.AuthTokensVerified.WithLabelValues(status).Inc()
				}

				// Return appropriate error
				if status == "forbidden" {
					http.Error(w, `{"error": "insufficient permissions"}`, http.StatusForbidden)
				} else {
					http.Error(w, `{"error": "invalid or expired token"}`, http.StatusUnauthorized)
				}
				return
			}

			// Record successful auth
			duration := time.Since(start).Seconds()
			if cfg.Metrics != nil {
				cfg.Metrics.AuthAttemptsTotal.WithLabelValues(path, "success").Inc()
				cfg.Metrics.AuthDuration.WithLabelValues(path).Observe(duration)
				cfg.Metrics.AuthTokensVerified.WithLabelValues("success").Inc()
			}

			cfg.Logger.Debug("authentication successful",
				"subject", claims.Subject,
				"email", claims.Email,
				"path", path,
				"duration_ms", time.Since(start).Milliseconds(),
			)

			// Add claims to request context for handlers to access
			ctx := context.WithValue(r.Context(), ClaimsContextKey, claims)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// isPublicPath checks if the given path matches any public path pattern
func isPublicPath(path string, publicPaths []string) bool {
	for _, publicPath := range publicPaths {
		// Exact match
		if path == publicPath {
			return true
		}
		// Wildcard match (e.g., "/api/*")
		if strings.HasSuffix(publicPath, "/*") {
			prefix := strings.TrimSuffix(publicPath, "/*")
			if strings.HasPrefix(path, prefix+"/") || path == prefix {
				return true
			}
		}
	}
	return false
}

// GetClaims extracts the auth claims from the request context
func GetClaims(r *http.Request) (*auth.Claims, bool) {
	claims, ok := r.Context().Value(ClaimsContextKey).(*auth.Claims)
	return claims, ok
}
