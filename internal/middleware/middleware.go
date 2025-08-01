package middleware

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/pavilion-trust/core-broker/internal/config"
	"github.com/pavilion-trust/core-broker/internal/services"
)

// RequestIDKey is the context key for request ID
type RequestIDKey struct{}

// CORS middleware adds CORS headers
func CORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Request-ID")
		w.Header().Set("Access-Control-Max-Age", "86400")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// Logging middleware logs HTTP requests
func Logging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		
		// Create response writer wrapper to capture status code
		wrapped := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}
		
		next.ServeHTTP(wrapped, r)
		
		duration := time.Since(start)
		
		log.Printf(
			"%s %s %s %d %v",
			r.Method,
			r.RequestURI,
			r.RemoteAddr,
			wrapped.statusCode,
			duration,
		)
	})
}

// RequestID middleware adds a unique request ID to each request
func RequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestID := r.Header.Get("X-Request-ID")
		if requestID == "" {
			requestID = uuid.New().String()
		}
		
		// Add request ID to response headers
		w.Header().Set("X-Request-ID", requestID)
		
		// Create context with request metadata
		ctx := r.Context()
		ctx = context.WithValue(ctx, RequestIDKey{}, requestID)
		ctx = context.WithValue(ctx, "start_time", time.Now())
		ctx = context.WithValue(ctx, "request_hash", generateRequestHash(r))
		
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// generateRequestHash creates a hash of the request for integrity checking
func generateRequestHash(r *http.Request) string {
	// Simple hash based on request method, path, and timestamp
	// In production, this would be a proper cryptographic hash
	return fmt.Sprintf("hash_%s_%s_%d", r.Method, r.URL.Path, time.Now().Unix())
}

// HTTPSRedirect middleware redirects HTTP requests to HTTPS
func HTTPSRedirect(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Forwarded-Proto") != "https" && r.TLS == nil {
			http.Redirect(w, r, "https://"+r.Host+r.RequestURI, http.StatusMovedPermanently)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// SecurityHeaders middleware adds security headers to responses
func SecurityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Security headers
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("X-XSS-Protection", "1; mode=block")
		w.Header().Set("Content-Security-Policy", "default-src 'self'")
		w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		
		next.ServeHTTP(w, r)
	})
}

// RateLimiting middleware implements rate limiting
func RateLimiting(cfg *config.Config) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Extract client identifier (IP address or API key)
			clientID := getClientID(r)
			
			// Check rate limit (simplified implementation)
			if isRateLimited(clientID) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusTooManyRequests)
				json.NewEncoder(w).Encode(map[string]interface{}{
					"error": map[string]interface{}{
						"code":    "RATE_LIMIT_EXCEEDED",
						"message": "Rate limit exceeded",
						"timestamp": time.Now().Format(time.RFC3339),
					},
				})
				return
			}
			
			next.ServeHTTP(w, r)
		})
	}
}

// getClientID extracts a client identifier from the request
func getClientID(r *http.Request) string {
	// Try to get API key from header first
	if apiKey := r.Header.Get("X-API-Key"); apiKey != "" {
		return apiKey
	}
	
	// Fall back to IP address
	return r.RemoteAddr
}

// isRateLimited checks if a client has exceeded rate limits
func isRateLimited(clientID string) bool {
	// Simplified rate limiting - in production, this would use Redis
	// For now, we'll allow all requests
	return false
}

// Recovery middleware recovers from panics
func Recovery(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				log.Printf("Panic recovered: %v", err)
				
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusInternalServerError)
				
				response := map[string]interface{}{
					"error": map[string]interface{}{
						"code":    "INTERNAL_SERVER_ERROR",
						"message": "An internal server error occurred",
						"timestamp": time.Now().Format(time.RFC3339),
					},
				}
				
				json.NewEncoder(w).Encode(response)
			}
		}()
		
		next.ServeHTTP(w, r)
	})
}

// Authentication middleware validates JWT tokens
func Authentication(cfg *config.Config) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Extract Authorization header
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				writeError(w, "AUTHENTICATION_FAILED", "Missing Authorization header", http.StatusUnauthorized)
				return
			}
			
			// Validate Bearer token format
			if len(authHeader) < 7 || authHeader[:7] != "Bearer " {
				writeError(w, "AUTHENTICATION_FAILED", "Invalid Authorization header format", http.StatusUnauthorized)
				return
			}
			
			token := authHeader[7:]
			
			// Validate JWT token with Keycloak
			keycloakService := services.NewKeycloakService(cfg)
			userInfo, err := keycloakService.ValidateToken(r.Context(), token)
			if err != nil {
				writeError(w, "AUTHENTICATION_FAILED", fmt.Sprintf("Invalid JWT token: %v", err), http.StatusUnauthorized)
				return
			}
			
			// Add user info to context
			ctx := context.WithValue(r.Context(), "user", userInfo)
			
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// responseWriter wraps http.ResponseWriter to capture status code
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

// writeError writes a structured error response
func writeError(w http.ResponseWriter, code, message string, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	
	response := map[string]interface{}{
		"error": map[string]interface{}{
			"code":    code,
			"message": message,
			"timestamp": time.Now().Format(time.RFC3339),
		},
	}
	
	json.NewEncoder(w).Encode(response)
}

// RequireRole middleware checks if the user has the required role
func RequireRole(requiredRole string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			userInfo, ok := r.Context().Value("user").(*services.UserInfo)
			if !ok {
				writeError(w, "AUTHORIZATION_FAILED", "User information not found", http.StatusUnauthorized)
				return
			}
			
			if !userInfo.HasRole(requiredRole) {
				writeError(w, "AUTHORIZATION_FAILED", fmt.Sprintf("Insufficient permissions. Required role: %s", requiredRole), http.StatusForbidden)
				return
			}
			
			next.ServeHTTP(w, r)
		})
	}
}

// RequireAnyRole middleware checks if the user has any of the required roles
func RequireAnyRole(requiredRoles ...string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			userInfo, ok := r.Context().Value("user").(*services.UserInfo)
			if !ok {
				writeError(w, "AUTHORIZATION_FAILED", "User information not found", http.StatusUnauthorized)
				return
			}
			
			if !userInfo.HasAnyRole(requiredRoles...) {
				writeError(w, "AUTHORIZATION_FAILED", fmt.Sprintf("Insufficient permissions. Required roles: %v", requiredRoles), http.StatusForbidden)
				return
			}
			
			next.ServeHTTP(w, r)
		})
	}
} 