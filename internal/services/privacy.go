package services

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"hash/fnv"

	"github.com/pavilion-trust/core-broker/internal/config"
	"github.com/pavilion-trust/core-broker/internal/models"
)

// PrivacyService handles privacy-preserving transformations
type PrivacyService struct {
	config *config.Config
}

// NewPrivacyService creates a new privacy service
func NewPrivacyService(cfg *config.Config) *PrivacyService {
	return &PrivacyService{
		config: cfg,
	}
}

// TransformRequest applies privacy-preserving transformations to a verification request
func (s *PrivacyService) TransformRequest(ctx context.Context, req models.VerificationRequest) (*models.PrivacyRequest, error) {
	// Hash user ID
	userHash := s.hashIdentifier(req.UserID)
	
	// Hash all identifiers
	hashedIdentifiers := make(map[string]string)
	for key, value := range req.Identifiers {
		hashedIdentifiers[key] = s.hashIdentifier(value)
	}
	
	// Generate Bloom filters for fuzzy matching
	bloomFilters := make(map[string]string)
	for key, value := range req.Identifiers {
		bloomFilters[key] = s.generateBloomFilter(value)
	}
	
	return &models.PrivacyRequest{
		RPID:             req.RPID,
		UserHash:         userHash,
		ClaimType:        req.ClaimType,
		HashedIdentifiers: hashedIdentifiers,
		BloomFilters:     bloomFilters,
		Metadata:         req.Metadata,
	}, nil
}

// hashIdentifier creates a SHA-256 hash of an identifier
func (s *PrivacyService) hashIdentifier(identifier string) string {
	hash := sha256.Sum256([]byte(identifier))
	return hex.EncodeToString(hash[:])
}

// generateBloomFilter creates a simple Bloom filter for fuzzy matching
func (s *PrivacyService) generateBloomFilter(value string) string {
	// Simple Bloom filter implementation using FNV hash
	// In production, this would use a more sophisticated Bloom filter library
	
	h := fnv.New32a()
	h.Write([]byte(value))
	hash := h.Sum32()
	
	// Convert to hex string for transmission
	return fmt.Sprintf("%08x", hash)
}

// HealthCheck checks if the privacy service is healthy
func (s *PrivacyService) HealthCheck(ctx context.Context) error {
	// Privacy service is stateless, so always healthy
	return nil
} 