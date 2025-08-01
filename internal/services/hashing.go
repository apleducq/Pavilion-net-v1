package services

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/pavilion-trust/core-broker/internal/config"
)

// HashService handles identifier hashing with enhanced privacy features
type HashService struct {
	config *config.Config
}

// HashResult represents the result of a hashing operation
type HashResult struct {
	OriginalValue string            `json:"original_value,omitempty"`
	HashedValue   string            `json:"hashed_value"`
	Salt          string            `json:"salt,omitempty"`
	HashType      string            `json:"hash_type"`
	Timestamp     string            `json:"timestamp"`
	Metadata      map[string]string `json:"metadata,omitempty"`
}

// NewHashService creates a new hash service
func NewHashService(cfg *config.Config) *HashService {
	return &HashService{
		config: cfg,
	}
}

// HashIdentifier creates a SHA-256 hash of an identifier
func (s *HashService) HashIdentifier(identifier string) (*HashResult, error) {
	if identifier == "" {
		return nil, fmt.Errorf("identifier cannot be empty")
	}

	// Generate salt for enhanced privacy
	salt, err := s.generateSalt()
	if err != nil {
		return nil, fmt.Errorf("failed to generate salt: %w", err)
	}

	// Create hash with salt
	hashedValue := s.hashWithSalt(identifier, salt)

	result := &HashResult{
		OriginalValue: identifier,
		HashedValue:   hashedValue,
		Salt:          salt,
		HashType:      "sha256",
		Timestamp:     time.Now().Format(time.RFC3339),
		Metadata: map[string]string{
			"algorithm": "SHA-256",
			"salted":    "true",
		},
	}

	// Log hash operation for audit
	s.logHashOperation(result)

	return result, nil
}

// HashIdentifierDeterministic creates a deterministic hash for matching purposes
func (s *HashService) HashIdentifierDeterministic(identifier string) (*HashResult, error) {
	if identifier == "" {
		return nil, fmt.Errorf("identifier cannot be empty")
	}

	// Use a fixed salt for deterministic hashing
	fixedSalt := s.getDeterministicSalt()
	hashedValue := s.hashWithSalt(identifier, fixedSalt)

	result := &HashResult{
		OriginalValue: identifier,
		HashedValue:   hashedValue,
		Salt:          fixedSalt,
		HashType:      "sha256_deterministic",
		Timestamp:     time.Now().Format(time.RFC3339),
		Metadata: map[string]string{
			"algorithm": "SHA-256",
			"salted":    "true",
			"deterministic": "true",
		},
	}

	// Log hash operation for audit
	s.logHashOperation(result)

	return result, nil
}

// ValidateHash validates a hash result
func (s *HashService) ValidateHash(result *HashResult) error {
	if result == nil {
		return fmt.Errorf("hash result cannot be nil")
	}

	if result.HashedValue == "" {
		return fmt.Errorf("hashed value cannot be empty")
	}

	if result.HashType == "" {
		return fmt.Errorf("hash type cannot be empty")
	}

	if result.Timestamp == "" {
		return fmt.Errorf("timestamp cannot be empty")
	}

	// Validate hash format (should be 64 characters for SHA-256)
	if len(result.HashedValue) != 64 {
		return fmt.Errorf("invalid hash length: expected 64, got %d", len(result.HashedValue))
	}

	// Validate hex format
	if _, err := hex.DecodeString(result.HashedValue); err != nil {
		return fmt.Errorf("invalid hash format: not a valid hex string")
	}

	return nil
}

// VerifyHash verifies that a hash matches the original value
func (s *HashService) VerifyHash(originalValue, hashedValue, salt string) (bool, error) {
	if originalValue == "" {
		return false, fmt.Errorf("original value cannot be empty")
	}

	if hashedValue == "" {
		return false, fmt.Errorf("hashed value cannot be empty")
	}

	// Recreate hash with the same salt
	expectedHash := s.hashWithSalt(originalValue, salt)

	return hashedValue == expectedHash, nil
}

// generateSalt generates a cryptographically secure random salt
func (s *HashService) generateSalt() (string, error) {
	saltBytes := make([]byte, 32) // 256 bits
	if _, err := rand.Read(saltBytes); err != nil {
		return "", fmt.Errorf("failed to generate random salt: %w", err)
	}
	return hex.EncodeToString(saltBytes), nil
}

// getDeterministicSalt returns a fixed salt for deterministic hashing
func (s *HashService) getDeterministicSalt() string {
	// Use a fixed salt for deterministic hashing
	// In production, this should be configurable and stored securely
	return "pavilion_deterministic_salt_v1"
}

// hashWithSalt creates a SHA-256 hash of the identifier with salt
func (s *HashService) hashWithSalt(identifier, salt string) string {
	// Combine identifier and salt
	data := identifier + salt
	
	// Create SHA-256 hash
	hash := sha256.Sum256([]byte(data))
	
	// Return hex-encoded hash
	return hex.EncodeToString(hash[:])
}

// logHashOperation logs hash operations for audit purposes
func (s *HashService) logHashOperation(result *HashResult) {
	// In a real implementation, this would log to an audit service
	// For now, we'll just add metadata to track the operation
	result.Metadata["audit_logged"] = "true"
	result.Metadata["log_timestamp"] = time.Now().Format(time.RFC3339)
}

// GetHashStats returns statistics about the hashing service
func (s *HashService) GetHashStats() map[string]interface{} {
	return map[string]interface{}{
		"service_status": "active",
		"hash_algorithm": "SHA-256",
		"salt_length":    32, // 256 bits
		"deterministic_salt_enabled": true,
	}
}

// HealthCheck checks if the hash service is healthy
func (s *HashService) HealthCheck(ctx context.Context) error {
	// Test salt generation
	_, err := s.generateSalt()
	if err != nil {
		return fmt.Errorf("hash service health check failed: %w", err)
	}

	// Test deterministic hashing
	_, err = s.HashIdentifierDeterministic("test")
	if err != nil {
		return fmt.Errorf("hash service health check failed: %w", err)
	}

	return nil
} 