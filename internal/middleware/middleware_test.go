package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/pavilion-trust/core-broker/internal/config"
	"github.com/pavilion-trust/core-broker/internal/services"
)

func TestAuthentication_MissingHeader(t *testing.T) {
	cfg := &config.Config{
		KeycloakURL:  "http://keycloak:8080",
		KeycloakRealm: "pavilion",
	}
	
	handler := Authentication(cfg)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("Handler should not be called when authentication fails")
	}))
	
	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	
	handler.ServeHTTP(w, req)
	
	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", w.Code)
	}
}

func TestAuthentication_InvalidFormat(t *testing.T) {
	cfg := &config.Config{
		KeycloakURL:  "http://keycloak:8080",
		KeycloakRealm: "pavilion",
	}
	
	handler := Authentication(cfg)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("Handler should not be called when authentication fails")
	}))
	
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "InvalidFormat token")
	w := httptest.NewRecorder()
	
	handler.ServeHTTP(w, req)
	
	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", w.Code)
	}
}

func TestAuthentication_EmptyToken(t *testing.T) {
	cfg := &config.Config{
		KeycloakURL:  "http://keycloak:8080",
		KeycloakRealm: "pavilion",
	}
	
	handler := Authentication(cfg)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("Handler should not be called when authentication fails")
	}))
	
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer ")
	w := httptest.NewRecorder()
	
	handler.ServeHTTP(w, req)
	
	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", w.Code)
	}
}

func TestRequireRole_NoUserInfo(t *testing.T) {
	handler := RequireRole("rp")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("Handler should not be called when authorization fails")
	}))
	
	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	
	handler.ServeHTTP(w, req)
	
	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", w.Code)
	}
}

func TestRequireRole_InsufficientPermissions(t *testing.T) {
	handler := RequireRole("admin")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("Handler should not be called when authorization fails")
	}))
	
	req := httptest.NewRequest("GET", "/test", nil)
	// Add user info to context (without required role)
	req = req.WithContext(context.WithValue(req.Context(), "user", &services.UserInfo{
		Subject: "user123",
		Roles:   []string{"rp"}, // Missing 'admin' role
	}))
	w := httptest.NewRecorder()
	
	handler.ServeHTTP(w, req)
	
	if w.Code != http.StatusForbidden {
		t.Errorf("Expected status 403, got %d", w.Code)
	}
}

func TestRequireAnyRole_Success(t *testing.T) {
	handler := RequireAnyRole("rp", "admin")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("success"))
	}))
	
	req := httptest.NewRequest("GET", "/test", nil)
	// Add user info to context (with one of the required roles)
	req = req.WithContext(context.WithValue(req.Context(), "user", &services.UserInfo{
		Subject: "user123",
		Roles:   []string{"rp"}, // Has 'rp' role
	}))
	w := httptest.NewRecorder()
	
	handler.ServeHTTP(w, req)
	
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
} 