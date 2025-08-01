package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/pavilion-trust/core-broker/internal/config"
)

func TestHTTPSRedirect(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	middleware := HTTPSRedirect(handler)

	tests := []struct {
		name           string
		requestURL     string
		headers        map[string]string
		expectedStatus int
		expectedRedirect string
	}{
		{
			name:           "HTTP request should redirect to HTTPS",
			requestURL:     "http://example.com/api/v1/verify",
			expectedStatus: http.StatusMovedPermanently,
			expectedRedirect: "https://example.com/api/v1/verify",
		},
		{
			name:           "HTTPS request should not redirect",
			requestURL:     "https://example.com/api/v1/verify",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "HTTP request with X-Forwarded-Proto should not redirect",
			requestURL:     "http://example.com/api/v1/verify",
			headers:        map[string]string{"X-Forwarded-Proto": "https"},
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", tt.requestURL, nil)
			for key, value := range tt.headers {
				req.Header.Set(key, value)
			}

			w := httptest.NewRecorder()
			middleware.ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			if tt.expectedRedirect != "" {
				location := w.Header().Get("Location")
				if location != tt.expectedRedirect {
					t.Errorf("Expected redirect to %s, got %s", tt.expectedRedirect, location)
				}
			}
		})
	}
}

func TestSecurityHeaders(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	middleware := SecurityHeaders(handler)

	req := httptest.NewRequest("GET", "/api/v1/verify", nil)
	w := httptest.NewRecorder()

	middleware.ServeHTTP(w, req)

	// Check security headers
	expectedHeaders := map[string]string{
		"X-Content-Type-Options": "nosniff",
		"X-Frame-Options":        "DENY",
		"X-XSS-Protection":       "1; mode=block",
		"Content-Security-Policy": "default-src 'self'",
		"Strict-Transport-Security": "max-age=31536000; includeSubDomains",
	}

	for header, expectedValue := range expectedHeaders {
		actualValue := w.Header().Get(header)
		if actualValue != expectedValue {
			t.Errorf("Expected header %s to be %s, got %s", header, expectedValue, actualValue)
		}
	}
}

func TestRateLimiting(t *testing.T) {
	cfg := &config.Config{}

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	middleware := RateLimiting(cfg)(handler)

	req := httptest.NewRequest("GET", "/api/v1/verify", nil)
	w := httptest.NewRecorder()

	middleware.ServeHTTP(w, req)

	// Should not be rate limited in this simplified implementation
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

func TestRateLimiting_WithAPIKey(t *testing.T) {
	cfg := &config.Config{}

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	middleware := RateLimiting(cfg)(handler)

	req := httptest.NewRequest("GET", "/api/v1/verify", nil)
	req.Header.Set("X-API-Key", "test-api-key")
	w := httptest.NewRecorder()

	middleware.ServeHTTP(w, req)

	// Should not be rate limited in this simplified implementation
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

func TestGetClientID(t *testing.T) {
	tests := []struct {
		name     string
		headers  map[string]string
		remoteAddr string
		expected string
	}{
		{
			name:     "API key should be used when present",
			headers:  map[string]string{"X-API-Key": "test-key"},
			remoteAddr: "192.168.1.1:1234",
			expected: "test-key",
		},
		{
			name:     "IP address should be used when no API key",
			headers:  map[string]string{},
			remoteAddr: "192.168.1.1:1234",
			expected: "192.168.1.1:1234",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/api/v1/verify", nil)
			for key, value := range tt.headers {
				req.Header.Set(key, value)
			}
			req.RemoteAddr = tt.remoteAddr

			result := getClientID(req)
			if result != tt.expected {
				t.Errorf("Expected client ID %s, got %s", tt.expected, result)
			}
		})
	}
}

func TestIsRateLimited(t *testing.T) {
	// This is a simplified implementation that always returns false
	// In production, this would check against Redis or another storage
	result := isRateLimited("test-client")
	if result {
		t.Error("Expected rate limiting to be disabled in test implementation")
	}
} 