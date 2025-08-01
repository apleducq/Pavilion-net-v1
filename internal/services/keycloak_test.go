package services

import (
	"context"
	"testing"

	"github.com/pavilion-trust/core-broker/internal/config"
)

func TestNewKeycloakService(t *testing.T) {
	cfg := &config.Config{
		KeycloakURL:  "http://keycloak:8080",
		KeycloakRealm: "pavilion",
	}
	
	service := NewKeycloakService(cfg)
	
	if service == nil {
		t.Error("Expected KeycloakService to be created")
	}
	
	if service.config != cfg {
		t.Error("Expected config to be set")
	}
	
	if service.publicKeys == nil {
		t.Error("Expected publicKeys map to be initialized")
	}
}

func TestUserInfo_HasRole(t *testing.T) {
	userInfo := &UserInfo{
		Subject: "user123",
		Roles:   []string{"rp", "admin"},
	}
	
	if !userInfo.HasRole("rp") {
		t.Error("Expected user to have 'rp' role")
	}
	
	if !userInfo.HasRole("admin") {
		t.Error("Expected user to have 'admin' role")
	}
	
	if userInfo.HasRole("nonexistent") {
		t.Error("Expected user to not have 'nonexistent' role")
	}
}

func TestUserInfo_HasAnyRole(t *testing.T) {
	userInfo := &UserInfo{
		Subject: "user123",
		Roles:   []string{"rp", "admin"},
	}
	
	if !userInfo.HasAnyRole("rp", "user") {
		t.Error("Expected user to have at least one of the required roles")
	}
	
	if !userInfo.HasAnyRole("admin", "superuser") {
		t.Error("Expected user to have at least one of the required roles")
	}
	
	if userInfo.HasAnyRole("nonexistent", "another") {
		t.Error("Expected user to not have any of the required roles")
	}
}

func TestKeycloakService_ValidateToken_InvalidToken(t *testing.T) {
	cfg := &config.Config{
		KeycloakURL:  "http://keycloak:8080",
		KeycloakRealm: "pavilion",
	}
	
	service := NewKeycloakService(cfg)
	ctx := context.Background()
	
	// Test with invalid token
	_, err := service.ValidateToken(ctx, "invalid.token.here")
	if err == nil {
		t.Error("Expected error for invalid token")
	}
}

func TestKeycloakService_HealthCheck(t *testing.T) {
	cfg := &config.Config{
		KeycloakURL:  "http://keycloak:8080",
		KeycloakRealm: "pavilion",
	}
	
	service := NewKeycloakService(cfg)
	ctx := context.Background()
	
	// Health check should fail in test environment (no Keycloak running)
	err := service.HealthCheck(ctx)
	if err == nil {
		t.Log("Health check passed (Keycloak might be running)")
	} else {
		t.Logf("Health check failed as expected: %v", err)
	}
} 