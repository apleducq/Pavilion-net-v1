package middleware

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/pavilion-trust/core-broker/internal/config"
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
		
		// Add request ID to context
		ctx := context.WithValue(r.Context(), RequestIDKey{}, requestID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
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
			
			// TODO: Implement JWT validation with Keycloak
			// For now, we'll just check if token exists
			if token == "" {
				writeError(w, "AUTHENTICATION_FAILED", "Invalid JWT token", http.StatusUnauthorized)
				return
			}
			
			// Add user info to context (placeholder for now)
			ctx := context.WithValue(r.Context(), "user", map[string]interface{}{
				"sub": "user123",
				"rp_id": "rp123",
			})
			
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