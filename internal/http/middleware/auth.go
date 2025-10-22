package middleware

import (
	"log/slog"
	"net/http"
	"os"
)

// AdminAuth is a simple admin authentication middleware using API key
type AdminAuth struct {
	adminAPIKey string
	logger      *slog.Logger
}

// NewAdminAuth creates a new admin authentication middleware
func NewAdminAuth(logger *slog.Logger) *AdminAuth {
	apiKey := os.Getenv("ADMIN_API_KEY")
	if apiKey == "" {
		logger.Warn("ADMIN_API_KEY not set - admin endpoints will be unprotected!")
		logger.Warn("Set ADMIN_API_KEY environment variable to enable authentication")
	}

	return &AdminAuth{
		adminAPIKey: apiKey,
		logger:      logger,
	}
}

// Middleware returns the authentication middleware handler
func (a *AdminAuth) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// If no API key is configured, allow all requests (development mode)
		if a.adminAPIKey == "" {
			a.logger.Debug("Admin auth bypassed - no API key configured")
			next.ServeHTTP(w, r)
			return
		}

		// Check Authorization header
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			a.logger.Warn("Admin request rejected - no authorization header",
				"path", r.URL.Path,
				"remote_addr", r.RemoteAddr,
			)
			http.Error(w, "Unauthorized - missing Authorization header", http.StatusUnauthorized)
			return
		}

		// Expect format: "Bearer <api_key>"
		expectedAuth := "Bearer " + a.adminAPIKey
		if authHeader != expectedAuth {
			a.logger.Warn("Admin request rejected - invalid API key",
				"path", r.URL.Path,
				"remote_addr", r.RemoteAddr,
			)
			http.Error(w, "Unauthorized - invalid API key", http.StatusUnauthorized)
			return
		}

		// Authentication successful
		a.logger.Debug("Admin request authenticated",
			"path", r.URL.Path,
			"method", r.Method,
		)

		next.ServeHTTP(w, r)
	})
}
