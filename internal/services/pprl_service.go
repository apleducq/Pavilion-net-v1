package services

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
	"time"
)

// PPRLService provides privacy-preserving record linkage functionality
type PPRLService struct {
	bloomFilter *BloomFilter
	config      *PPRLConfig
}

// PPRLConfig holds configuration for PPRL operations
type PPRLConfig struct {
	BloomFilterSize              int
	BloomFilterHashCount         int
	BloomFilterFalsePositiveRate float64
	HashAlgorithm                string
	Salt                         string
}

// NewPPRLConfig creates a new PPRL configuration
func NewPPRLConfig(size, hashCount int, falsePositiveRate float64, salt string) *PPRLConfig {
	return &PPRLConfig{
		BloomFilterSize:              size,
		BloomFilterHashCount:         hashCount,
		BloomFilterFalsePositiveRate: falsePositiveRate,
		HashAlgorithm:                "SHA-256",
		Salt:                         salt,
	}
}

// NewPPRLService creates a new PPRL service
func NewPPRLService(config *PPRLConfig) *PPRLService {
	bloomFilter := NewBloomFilter(config.BloomFilterSize, config.BloomFilterHashCount)

	return &PPRLService{
		bloomFilter: bloomFilter,
		config:      config,
	}
}

// SensitiveField represents a sensitive field that needs to be hashed
type SensitiveField struct {
	Name  string `json:"name"`
	Value string `json:"value"`
	Type  string `json:"type"` // "exact", "fuzzy", "phonetic"
}

// DataProviderRecord represents a record from a data provider
type DataProviderRecord struct {
	ID       string                    `json:"id"`
	Provider string                    `json:"provider"`
	Fields   map[string]SensitiveField `json:"fields"`
	Created  time.Time                 `json:"created"`
}

// PPRLRequest represents a request for privacy-preserving record linkage
type PPRLRequest struct {
	QueryFields map[string]SensitiveField `json:"query_fields"`
	ProviderID  string                    `json:"provider_id"`
	Threshold   float64                   `json:"threshold,omitempty"`
}

// PPRLResponse represents the response from PPRL operations
type PPRLResponse struct {
	MatchFound    bool                   `json:"match_found"`
	Confidence    float64                `json:"confidence"`
	ProviderID    string                 `json:"provider_id"`
	MatchedFields []string               `json:"matched_fields,omitempty"`
	BloomFilter   string                 `json:"bloom_filter,omitempty"`
	Metadata      map[string]interface{} `json:"metadata"`
}

// HashSensitiveField hashes a sensitive field using SHA-256
func (p *PPRLService) HashSensitiveField(field SensitiveField) (string, error) {
	// Normalize the field value
	normalizedValue := strings.ToLower(strings.TrimSpace(field.Value))

	// Apply field-specific processing
	switch field.Type {
	case "exact":
		// Use exact value
		break
	case "fuzzy":
		// Apply fuzzy matching preprocessing
		normalizedValue = p.preprocessForFuzzy(normalizedValue)
	case "phonetic":
		// Apply phonetic encoding
		encoder := NewPhoneticEncoder()
		normalizedValue = encoder.Encode(normalizedValue)
	default:
		// Default to exact matching
		break
	}

	// Create data to hash: field name + value + salt
	dataToHash := fmt.Sprintf("%s:%s:%s", field.Name, normalizedValue, p.config.Salt)

	// Hash using SHA-256
	hash := sha256.Sum256([]byte(dataToHash))

	return hex.EncodeToString(hash[:]), nil
}

// preprocessForFuzzy applies preprocessing for fuzzy matching
func (p *PPRLService) preprocessForFuzzy(value string) string {
	// Convert to lowercase first
	value = strings.ToLower(value)

	// Remove common prefixes/suffixes
	value = strings.TrimPrefix(value, "mr ")
	value = strings.TrimPrefix(value, "mrs ")
	value = strings.TrimPrefix(value, "ms ")
	value = strings.TrimPrefix(value, "dr ")

	// Remove punctuation and extra spaces
	value = strings.Map(func(r rune) rune {
		if r >= 'a' && r <= 'z' || r >= '0' && r <= '9' {
			return r
		}
		if r == ' ' || r == '-' {
			return ' '
		}
		return -1
	}, value)

	// Normalize spaces
	value = strings.Join(strings.Fields(value), " ")

	return value
}

// CreateBloomFilterForRecord creates a Bloom filter for a data provider record
func (p *PPRLService) CreateBloomFilterForRecord(record DataProviderRecord) (*BloomFilter, error) {
	// Create a new Bloom filter for this record
	bloomFilter := NewBloomFilter(p.config.BloomFilterSize, p.config.BloomFilterHashCount)

	// Hash and add each sensitive field
	for fieldName, field := range record.Fields {
		hashedValue, err := p.HashSensitiveField(field)
		if err != nil {
			return nil, fmt.Errorf("failed to hash field %s: %w", fieldName, err)
		}

		// Add the hashed value to the Bloom filter
		bloomFilter.Add(hashedValue)

		// Also add field name + hashed value for better specificity
		fieldIdentifier := fmt.Sprintf("%s:%s", fieldName, hashedValue)
		bloomFilter.Add(fieldIdentifier)
	}

	return bloomFilter, nil
}

// PerformPPRL performs privacy-preserving record linkage
func (p *PPRLService) PerformPPRL(request PPRLRequest, providerRecords []DataProviderRecord) (*PPRLResponse, error) {
	// Create Bloom filter for query fields
	queryBloomFilter := NewBloomFilter(p.config.BloomFilterSize, p.config.BloomFilterHashCount)

	// Hash and add query fields to Bloom filter
	for fieldName, field := range request.QueryFields {
		hashedValue, err := p.HashSensitiveField(field)
		if err != nil {
			return nil, fmt.Errorf("failed to hash query field %s: %w", fieldName, err)
		}

		queryBloomFilter.Add(hashedValue)

		// Also add field name + hashed value
		fieldIdentifier := fmt.Sprintf("%s:%s", fieldName, hashedValue)
		queryBloomFilter.Add(fieldIdentifier)
	}

	// Compare with provider records
	bestMatch := p.findBestMatch(queryBloomFilter, providerRecords, request.Threshold)

	response := &PPRLResponse{
		MatchFound:  bestMatch != nil,
		ProviderID:  request.ProviderID,
		BloomFilter: queryBloomFilter.ToHexString(),
		Metadata: map[string]interface{}{
			"query_fields_count":     len(request.QueryFields),
			"provider_records_count": len(providerRecords),
			"false_positive_rate":    queryBloomFilter.GetFalsePositiveRate(),
			"timestamp":              time.Now().Format(time.RFC3339),
		},
	}

	if bestMatch != nil {
		response.Confidence = bestMatch.confidence
		response.MatchedFields = bestMatch.matchedFields
		response.Metadata["matched_record_id"] = bestMatch.record.ID
		response.Metadata["matched_provider"] = bestMatch.record.Provider
	}

	return response, nil
}

// matchResult represents a match result with confidence and details
type matchResult struct {
	record        DataProviderRecord
	confidence    float64
	matchedFields []string
}

// findBestMatch finds the best matching record
func (p *PPRLService) findBestMatch(queryBloomFilter *BloomFilter, providerRecords []DataProviderRecord, threshold float64) *matchResult {
	var bestMatch *matchResult

	for _, record := range providerRecords {
		// Create Bloom filter for this record
		recordBloomFilter, err := p.CreateBloomFilterForRecord(record)
		if err != nil {
			continue // Skip this record if we can't process it
		}

		// Calculate similarity between Bloom filters
		similarity := p.calculateBloomFilterSimilarity(queryBloomFilter, recordBloomFilter)

		// Check if this is a better match
		if similarity >= threshold && (bestMatch == nil || similarity > bestMatch.confidence) {
			matchedFields := p.findMatchedFields(queryBloomFilter, recordBloomFilter, record)

			bestMatch = &matchResult{
				record:        record,
				confidence:    similarity,
				matchedFields: matchedFields,
			}
		}
	}

	return bestMatch
}

// calculateBloomFilterSimilarity calculates similarity between two Bloom filters
func (p *PPRLService) calculateBloomFilterSimilarity(bf1, bf2 *BloomFilter) float64 {
	if bf1.size != bf2.size {
		return 0.0
	}

	// Calculate Jaccard similarity
	intersection := 0
	union := 0

	for i := 0; i < bf1.size; i++ {
		if bf1.bitArray[i] && bf2.bitArray[i] {
			intersection++
		}
		if bf1.bitArray[i] || bf2.bitArray[i] {
			union++
		}
	}

	if union == 0 {
		return 0.0
	}

	return float64(intersection) / float64(union)
}

// findMatchedFields identifies which fields matched
func (p *PPRLService) findMatchedFields(queryBF, recordBF *BloomFilter, record DataProviderRecord) []string {
	var matchedFields []string

	for fieldName, field := range record.Fields {
		hashedValue, err := p.HashSensitiveField(field)
		if err != nil {
			continue
		}

		// Check if this field's hash is in the query Bloom filter
		if queryBF.Contains(hashedValue) {
			matchedFields = append(matchedFields, fieldName)
		}
	}

	return matchedFields
}

// ValidatePPRLRequest validates a PPRL request
func (p *PPRLService) ValidatePPRLRequest(request PPRLRequest) error {
	if len(request.QueryFields) == 0 {
		return fmt.Errorf("query fields cannot be empty")
	}

	if request.ProviderID == "" {
		return fmt.Errorf("provider ID is required")
	}

	if request.Threshold <= 0 || request.Threshold > 1 {
		return fmt.Errorf("threshold must be between 0 and 1")
	}

	// Validate each field
	for fieldName, field := range request.QueryFields {
		if field.Name == "" {
			return fmt.Errorf("field name cannot be empty for field %s", fieldName)
		}

		if field.Value == "" {
			return fmt.Errorf("field value cannot be empty for field %s", fieldName)
		}

		if field.Type != "exact" && field.Type != "fuzzy" && field.Type != "phonetic" {
			return fmt.Errorf("invalid field type %s for field %s", field.Type, fieldName)
		}
	}

	return nil
}

// GetPPRLStats returns statistics about PPRL operations
func (p *PPRLService) GetPPRLStats() map[string]interface{} {
	return map[string]interface{}{
		"bloom_filter_size":                p.config.BloomFilterSize,
		"bloom_filter_hash_count":          p.config.BloomFilterHashCount,
		"bloom_filter_false_positive_rate": p.config.BloomFilterFalsePositiveRate,
		"hash_algorithm":                   p.config.HashAlgorithm,
		"current_false_positive_rate":      p.bloomFilter.GetFalsePositiveRate(),
	}
}

// ExportBloomFilter exports a Bloom filter to a portable format
func (p *PPRLService) ExportBloomFilter(bloomFilter *BloomFilter) (map[string]interface{}, error) {
	return map[string]interface{}{
		"bloom_filter": bloomFilter.ToHexString(),
		"size":         bloomFilter.size,
		"hash_count":   bloomFilter.hashCount,
		"metadata": map[string]interface{}{
			"exported_at":         time.Now().Format(time.RFC3339),
			"false_positive_rate": bloomFilter.GetFalsePositiveRate(),
		},
	}, nil
}

// ImportBloomFilter imports a Bloom filter from a portable format
func (p *PPRLService) ImportBloomFilter(data map[string]interface{}) (*BloomFilter, error) {
	bloomFilterHex, ok := data["bloom_filter"].(string)
	if !ok {
		return nil, fmt.Errorf("invalid bloom filter data")
	}

	size, ok := data["size"].(int)
	if !ok {
		return nil, fmt.Errorf("invalid size data")
	}

	hashCount, ok := data["hash_count"].(int)
	if !ok {
		return nil, fmt.Errorf("invalid hash count data")
	}

	return FromHexString(bloomFilterHex, size, hashCount)
}
