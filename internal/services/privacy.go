package services

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"strings"

	"github.com/pavilion-trust/core-broker/internal/config"
	"github.com/pavilion-trust/core-broker/internal/models"
)

// PrivacyService handles privacy-preserving transformations
type PrivacyService struct {
	config       *config.Config
	bloomFilter  *BloomFilter
	fuzzyMatcher *FuzzyMatcher
}

// NewPrivacyService creates a new privacy service
func NewPrivacyService(cfg *config.Config) *PrivacyService {
	// Create Bloom filter with configurable parameters
	bloomFilter := NewBloomFilter(cfg.BloomFilterSize, cfg.BloomFilterHashCount)
	
	return &PrivacyService{
		config:       cfg,
		bloomFilter:  bloomFilter,
		fuzzyMatcher: NewFuzzyMatcher(),
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
		bloomFilter := s.generateBloomFilter(value)
		bloomFilters[key] = bloomFilter
	}
	
	// Add phonetic encodings if enabled
	if s.config.PhoneticEncodingEnabled {
		for key, value := range req.Identifiers {
			// Only apply phonetic encoding to name-like fields
			if s.isNameField(key) {
				phoneticCode := s.fuzzyMatcher.GetPhoneticCode(value)
				bloomFilters[key+"_phonetic"] = phoneticCode
			}
		}
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

// generateBloomFilter creates a Bloom filter for fuzzy matching
func (s *PrivacyService) generateBloomFilter(value string) string {
	// Create a new Bloom filter for this value
	bf := NewBloomFilter(s.config.BloomFilterSize, s.config.BloomFilterHashCount)
	
	// Add the value to the Bloom filter
	bf.Add(value)
	
	// Convert to hex string for transmission
	return bf.ToHexString()
}

// isNameField checks if a field is likely to be a name field
func (s *PrivacyService) isNameField(fieldName string) bool {
	nameFields := []string{"name", "first_name", "last_name", "full_name", "given_name", "family_name"}
	fieldLower := strings.ToLower(fieldName)
	
	for _, nameField := range nameFields {
		if strings.Contains(fieldLower, nameField) {
			return true
		}
	}
	return false
}

// CalculateFuzzySimilarity calculates similarity between two values
func (s *PrivacyService) CalculateFuzzySimilarity(value1, value2 string) float64 {
	return s.fuzzyMatcher.CalculateSimilarity(value1, value2)
}

// IsPhoneticallySimilar checks if two names are phonetically similar
func (s *PrivacyService) IsPhoneticallySimilar(name1, name2 string) bool {
	return s.fuzzyMatcher.IsPhoneticallySimilar(name1, name2)
}

// GetPhoneticCode returns the phonetic encoding of a name
func (s *PrivacyService) GetPhoneticCode(name string) string {
	return s.fuzzyMatcher.GetPhoneticCode(name)
}

// GetBloomFilterStats returns statistics about the Bloom filter
func (s *PrivacyService) GetBloomFilterStats() map[string]interface{} {
	return map[string]interface{}{
		"size":                    s.config.BloomFilterSize,
		"hash_count":              s.config.BloomFilterHashCount,
		"false_positive_rate":     s.config.BloomFilterFalsePositiveRate,
		"phonetic_encoding_enabled": s.config.PhoneticEncodingEnabled,
	}
}

// HealthCheck checks if the privacy service is healthy
func (s *PrivacyService) HealthCheck(ctx context.Context) error {
	// Privacy service is stateless, so always healthy
	return nil
} 