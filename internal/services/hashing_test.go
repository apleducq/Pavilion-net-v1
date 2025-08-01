package services

import (
	"context"
	"encoding/hex"
	"strings"
	"testing"
	"time"

	"github.com/pavilion-trust/core-broker/internal/config"
)

func TestNewHashService(t *testing.T) {
	cfg := &config.Config{}
	service := NewHashService(cfg)
	
	if service.config != cfg {
		t.Error("Hash service should have the correct config")
	}
}

func TestHashService_HashIdentifier(t *testing.T) {
	cfg := &config.Config{}
	service := NewHashService(cfg)
	
	testCases := []struct {
		identifier string
		expectError bool
	}{
		{"test123", false},
		{"user@example.com", false},
		{"12345", false},
		{"", true}, // Empty identifier should fail
	}
	
	for _, tc := range testCases {
		result, err := service.HashIdentifier(tc.identifier)
		
		if tc.expectError {
			if err == nil {
				t.Errorf("Expected error for identifier '%s', but got none", tc.identifier)
			}
			continue
		}
		
		if err != nil {
			t.Errorf("Unexpected error for identifier '%s': %v", tc.identifier, err)
			continue
		}
		
		// Validate result
		if result == nil {
			t.Errorf("Result should not be nil for identifier '%s'", tc.identifier)
			continue
		}
		
		if result.HashedValue == "" {
			t.Errorf("Hashed value should not be empty for identifier '%s'", tc.identifier)
		}
		
		if len(result.HashedValue) != 64 {
			t.Errorf("Hash should be 64 characters, got %d for identifier '%s'", len(result.HashedValue), tc.identifier)
		}
		
		if result.Salt == "" {
			t.Errorf("Salt should not be empty for identifier '%s'", tc.identifier)
		}
		
		if result.HashType != "sha256" {
			t.Errorf("Expected hash type 'sha256', got '%s' for identifier '%s'", result.HashType, tc.identifier)
		}
		
		if result.Timestamp == "" {
			t.Errorf("Timestamp should not be empty for identifier '%s'", tc.identifier)
		}
		
		// Verify hash can be recreated
		expectedHash := service.hashWithSalt(tc.identifier, result.Salt)
		if result.HashedValue != expectedHash {
			t.Errorf("Hash verification failed for identifier '%s'", tc.identifier)
		}
	}
}

func TestHashService_HashIdentifierDeterministic(t *testing.T) {
	cfg := &config.Config{}
	service := NewHashService(cfg)
	
	identifier := "test123"
	
	// Hash the same identifier multiple times
	result1, err := service.HashIdentifierDeterministic(identifier)
	if err != nil {
		t.Fatalf("Failed to hash identifier: %v", err)
	}
	
	result2, err := service.HashIdentifierDeterministic(identifier)
	if err != nil {
		t.Fatalf("Failed to hash identifier: %v", err)
	}
	
	// Deterministic hashes should be identical
	if result1.HashedValue != result2.HashedValue {
		t.Errorf("Deterministic hashes should be identical, got %s and %s", result1.HashedValue, result2.HashedValue)
	}
	
	if result1.Salt != result2.Salt {
		t.Errorf("Deterministic salts should be identical, got %s and %s", result1.Salt, result2.Salt)
	}
	
	if result1.HashType != "sha256_deterministic" {
		t.Errorf("Expected hash type 'sha256_deterministic', got '%s'", result1.HashType)
	}
	
	// Verify deterministic salt
	expectedSalt := service.getDeterministicSalt()
	if result1.Salt != expectedSalt {
		t.Errorf("Expected deterministic salt '%s', got '%s'", expectedSalt, result1.Salt)
	}
}

func TestHashService_ValidateHash(t *testing.T) {
	cfg := &config.Config{}
	service := NewHashService(cfg)
	
	// Test valid hash result
	validResult := &HashResult{
		HashedValue: "a" + strings.Repeat("0", 63), // 64 characters
		HashType:    "sha256",
		Timestamp:   time.Now().Format(time.RFC3339),
	}
	
	if err := service.ValidateHash(validResult); err != nil {
		t.Errorf("Valid hash result should not fail validation: %v", err)
	}
	
	// Test nil result
	if err := service.ValidateHash(nil); err == nil {
		t.Error("Nil hash result should fail validation")
	}
	
	// Test empty hashed value
	invalidResult := &HashResult{
		HashedValue: "",
		HashType:    "sha256",
		Timestamp:   time.Now().Format(time.RFC3339),
	}
	
	if err := service.ValidateHash(invalidResult); err == nil {
		t.Error("Empty hashed value should fail validation")
	}
	
	// Test invalid hash length
	invalidLengthResult := &HashResult{
		HashedValue: "short",
		HashType:    "sha256",
		Timestamp:   time.Now().Format(time.RFC3339),
	}
	
	if err := service.ValidateHash(invalidLengthResult); err == nil {
		t.Error("Invalid hash length should fail validation")
	}
	
	// Test invalid hex format
	invalidHexResult := &HashResult{
		HashedValue: strings.Repeat("g", 64), // Invalid hex
		HashType:    "sha256",
		Timestamp:   time.Now().Format(time.RFC3339),
	}
	
	if err := service.ValidateHash(invalidHexResult); err == nil {
		t.Error("Invalid hex format should fail validation")
	}
}

func TestHashService_VerifyHash(t *testing.T) {
	cfg := &config.Config{}
	service := NewHashService(cfg)
	
	identifier := "test123"
	salt := "testsalt"
	hashedValue := service.hashWithSalt(identifier, salt)
	
	// Test valid verification
	valid, err := service.VerifyHash(identifier, hashedValue, salt)
	if err != nil {
		t.Errorf("Hash verification should not fail: %v", err)
	}
	if !valid {
		t.Error("Hash verification should be valid")
	}
	
	// Test invalid verification
	valid, err = service.VerifyHash("wrong", hashedValue, salt)
	if err != nil {
		t.Errorf("Hash verification should not fail: %v", err)
	}
	if valid {
		t.Error("Hash verification should be invalid for wrong identifier")
	}
	
	// Test empty original value
	_, err = service.VerifyHash("", hashedValue, salt)
	if err == nil {
		t.Error("Empty original value should fail verification")
	}
	
	// Test empty hashed value
	_, err = service.VerifyHash(identifier, "", salt)
	if err == nil {
		t.Error("Empty hashed value should fail verification")
	}
}

func TestHashService_GenerateSalt(t *testing.T) {
	cfg := &config.Config{}
	service := NewHashService(cfg)
	
	// Generate multiple salts
	salt1, err := service.generateSalt()
	if err != nil {
		t.Fatalf("Failed to generate salt: %v", err)
	}
	
	salt2, err := service.generateSalt()
	if err != nil {
		t.Fatalf("Failed to generate salt: %v", err)
	}
	
	// Salts should be different (random)
	if salt1 == salt2 {
		t.Error("Generated salts should be different")
	}
	
	// Salts should be 64 characters (32 bytes = 64 hex chars)
	if len(salt1) != 64 {
		t.Errorf("Salt should be 64 characters, got %d", len(salt1))
	}
	
	if len(salt2) != 64 {
		t.Errorf("Salt should be 64 characters, got %d", len(salt2))
	}
	
	// Validate hex format
	if _, err := hex.DecodeString(salt1); err != nil {
		t.Errorf("Salt should be valid hex: %v", err)
	}
	
	if _, err := hex.DecodeString(salt2); err != nil {
		t.Errorf("Salt should be valid hex: %v", err)
	}
}

func TestHashService_GetDeterministicSalt(t *testing.T) {
	cfg := &config.Config{}
	service := NewHashService(cfg)
	
	salt1 := service.getDeterministicSalt()
	salt2 := service.getDeterministicSalt()
	
	// Deterministic salt should be the same
	if salt1 != salt2 {
		t.Error("Deterministic salt should be the same")
	}
	
	// Should not be empty
	if salt1 == "" {
		t.Error("Deterministic salt should not be empty")
	}
}

func TestHashService_HashWithSalt(t *testing.T) {
	cfg := &config.Config{}
	service := NewHashService(cfg)
	
	identifier := "test123"
	salt := "testsalt"
	
	hash1 := service.hashWithSalt(identifier, salt)
	hash2 := service.hashWithSalt(identifier, salt)
	
	// Same input should produce same hash
	if hash1 != hash2 {
		t.Error("Same input should produce same hash")
	}
	
	// Hash should be 64 characters
	if len(hash1) != 64 {
		t.Errorf("Hash should be 64 characters, got %d", len(hash1))
	}
	
	// Different salt should produce different hash
	hash3 := service.hashWithSalt(identifier, "differentsalt")
	if hash1 == hash3 {
		t.Error("Different salt should produce different hash")
	}
	
	// Different identifier should produce different hash
	hash4 := service.hashWithSalt("different", salt)
	if hash1 == hash4 {
		t.Error("Different identifier should produce different hash")
	}
}

func TestHashService_GetHashStats(t *testing.T) {
	cfg := &config.Config{}
	service := NewHashService(cfg)
	
	stats := service.GetHashStats()
	
	expectedKeys := []string{"service_status", "hash_algorithm", "salt_length", "deterministic_salt_enabled"}
	for _, key := range expectedKeys {
		if _, exists := stats[key]; !exists {
			t.Errorf("Stats should contain key: %s", key)
		}
	}
	
	if stats["service_status"] != "active" {
		t.Errorf("Expected service_status 'active', got %v", stats["service_status"])
	}
	
	if stats["hash_algorithm"] != "SHA-256" {
		t.Errorf("Expected hash_algorithm 'SHA-256', got %v", stats["hash_algorithm"])
	}
	
	if stats["salt_length"] != 32 {
		t.Errorf("Expected salt_length 32, got %v", stats["salt_length"])
	}
	
	if stats["deterministic_salt_enabled"] != true {
		t.Errorf("Expected deterministic_salt_enabled true, got %v", stats["deterministic_salt_enabled"])
	}
}

func TestHashService_HealthCheck(t *testing.T) {
	cfg := &config.Config{}
	service := NewHashService(cfg)
	
	ctx := context.Background()
	err := service.HealthCheck(ctx)
	
	if err != nil {
		t.Errorf("Health check should not fail: %v", err)
	}
} 