package services

import (
	"context"
	"math"
	"testing"

	"github.com/pavilion-trust/core-broker/internal/config"
	"github.com/pavilion-trust/core-broker/internal/models"
)

func TestNewPrivacyService(t *testing.T) {
	cfg := &config.Config{
		BloomFilterSize:     1000,
		BloomFilterHashCount: 5,
		PhoneticEncodingEnabled: true,
	}
	
	service := NewPrivacyService(cfg)
	
	if service.config != cfg {
		t.Error("Privacy service should have the correct config")
	}
	
	if service.bloomFilter == nil {
		t.Error("Privacy service should have a Bloom filter")
	}
	
	if service.fuzzyMatcher == nil {
		t.Error("Privacy service should have a fuzzy matcher")
	}
}

func TestPrivacyService_TransformRequest(t *testing.T) {
	cfg := &config.Config{
		BloomFilterSize:     1000,
		BloomFilterHashCount: 5,
		PhoneticEncodingEnabled: true,
	}
	
	service := NewPrivacyService(cfg)
	
	req := models.VerificationRequest{
		RPID:      "test-rp",
		UserID:    "user123",
		ClaimType: "student_verification",
		Identifiers: map[string]string{
			"first_name": "John",
			"last_name":  "Smith",
			"student_id": "12345",
		},
		Metadata: map[string]interface{}{
			"source": "test",
		},
	}
	
	ctx := context.Background()
	privacyReq, err := service.TransformRequest(ctx, req)
	
	if err != nil {
		t.Fatalf("TransformRequest failed: %v", err)
	}
	
	if privacyReq.RPID != req.RPID {
		t.Errorf("Expected RPID %s, got %s", req.RPID, privacyReq.RPID)
	}
	
	if privacyReq.ClaimType != req.ClaimType {
		t.Errorf("Expected ClaimType %s, got %s", req.ClaimType, privacyReq.ClaimType)
	}
	
	// Check that user ID is hashed
	if privacyReq.UserHash == req.UserID {
		t.Error("User ID should be hashed")
	}
	
	// Check that identifiers are hashed
	for key, value := range req.Identifiers {
		if privacyReq.HashedIdentifiers[key] == value {
			t.Errorf("Identifier %s should be hashed", key)
		}
	}
	
	// Check that Bloom filters are generated
	for key := range req.Identifiers {
		if privacyReq.BloomFilters[key] == "" {
			t.Errorf("Bloom filter should be generated for %s", key)
		}
	}
	
	// Check that phonetic encodings are generated for name fields
	if privacyReq.BloomFilters["first_name_phonetic"] == "" {
		t.Error("Phonetic encoding should be generated for first_name")
	}
	
	if privacyReq.BloomFilters["last_name_phonetic"] == "" {
		t.Error("Phonetic encoding should be generated for last_name")
	}
	
	// Check that non-name fields don't have phonetic encodings
	if privacyReq.BloomFilters["student_id_phonetic"] != "" {
		t.Error("Non-name field should not have phonetic encoding")
	}
}

func TestPrivacyService_TransformRequest_NoPhoneticEncoding(t *testing.T) {
	cfg := &config.Config{
		BloomFilterSize:     1000,
		BloomFilterHashCount: 5,
		PhoneticEncodingEnabled: false,
	}
	
	service := NewPrivacyService(cfg)
	
	req := models.VerificationRequest{
		RPID:      "test-rp",
		UserID:    "user123",
		ClaimType: "student_verification",
		Identifiers: map[string]string{
			"first_name": "John",
			"last_name":  "Smith",
		},
	}
	
	ctx := context.Background()
	privacyReq, err := service.TransformRequest(ctx, req)
	
	if err != nil {
		t.Fatalf("TransformRequest failed: %v", err)
	}
	
	// Check that phonetic encodings are not generated when disabled
	if privacyReq.BloomFilters["first_name_phonetic"] != "" {
		t.Error("Phonetic encoding should not be generated when disabled")
	}
	
	if privacyReq.BloomFilters["last_name_phonetic"] != "" {
		t.Error("Phonetic encoding should not be generated when disabled")
	}
}

func TestPrivacyService_HashIdentifier(t *testing.T) {
	cfg := &config.Config{}
	service := NewPrivacyService(cfg)
	
	testCases := []struct {
		input    string
		expected string
	}{
		{"test", "9f86d081884c7d659a2feaa0c55ad015a3bf4f1b2b0b822cd15d6c15b0f00a08"},
		{"hello", "2cf24dba5fb0a30e26e83b2ac5b9e29e1b161e5c1fa7425e73043362938b9824"},
		{"", "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"},
	}
	
	for _, tc := range testCases {
		result := service.hashIdentifier(tc.input)
		if result != tc.expected {
			t.Errorf("For input '%s', expected '%s', got '%s'", tc.input, tc.expected, result)
		}
	}
}

func TestPrivacyService_GenerateBloomFilter(t *testing.T) {
	cfg := &config.Config{
		BloomFilterSize:     100,
		BloomFilterHashCount: 3,
	}
	
	service := NewPrivacyService(cfg)
	
	value := "test value"
	bloomFilter := service.generateBloomFilter(value)
	
	if bloomFilter == "" {
		t.Error("Bloom filter should not be empty")
	}
	
	// Test that the hex string is valid
	if len(bloomFilter)%2 != 0 {
		t.Error("Bloom filter hex string should have even length")
	}
}

func TestPrivacyService_IsNameField(t *testing.T) {
	cfg := &config.Config{}
	service := NewPrivacyService(cfg)
	
	testCases := []struct {
		fieldName string
		expected  bool
	}{
		{"name", true},
		{"first_name", true},
		{"last_name", true},
		{"full_name", true},
		{"given_name", true},
		{"family_name", true},
		{"student_id", false},
		{"email", false},
		{"phone", false},
		{"address", false},
		{"Name", true},
		{"FIRST_NAME", true},
		{"LastName", true},
	}
	
	for _, tc := range testCases {
		result := service.isNameField(tc.fieldName)
		if result != tc.expected {
			t.Errorf("For field '%s', expected %t, got %t", tc.fieldName, tc.expected, result)
		}
	}
}

func TestPrivacyService_CalculateFuzzySimilarity(t *testing.T) {
	cfg := &config.Config{}
	service := NewPrivacyService(cfg)
	
	testCases := []struct {
		value1   string
		value2   string
		expected float64
	}{
		{"hello", "hello", 1.0},
		{"hello", "helo", 0.8},
		{"john", "jon", 0.75},
		{"smith", "smyth", 0.8},
		{"", "", 0.0}, // Empty strings should return 0.0
		{"hello", "", 0.0},
		{"", "world", 0.0},
	}
	
	for _, tc := range testCases {
		result := service.CalculateFuzzySimilarity(tc.value1, tc.value2)
		// Use approximate comparison for floating point values
		if math.Abs(result-tc.expected) > 0.001 {
			t.Errorf("For '%s' vs '%s', expected %f, got %f", tc.value1, tc.value2, tc.expected, result)
		}
	}
}

func TestPrivacyService_IsPhoneticallySimilar(t *testing.T) {
	cfg := &config.Config{}
	service := NewPrivacyService(cfg)
	
	testCases := []struct {
		name1    string
		name2    string
		expected bool
	}{
		{"John", "Jon", true},
		{"Smith", "Smyth", true},
		{"Johnson", "Jonson", true},
		{"John", "Jane", true}, // Both encode to J500 (no consonants)
		{"Smith", "Brown", false},
	}
	
	for _, tc := range testCases {
		result := service.IsPhoneticallySimilar(tc.name1, tc.name2)
		if result != tc.expected {
			t.Errorf("For '%s' vs '%s', expected %t, got %t", tc.name1, tc.name2, tc.expected, result)
		}
	}
}

func TestPrivacyService_GetPhoneticCode(t *testing.T) {
	cfg := &config.Config{}
	service := NewPrivacyService(cfg)
	
	testCases := []struct {
		input    string
		expected string
	}{
		{"John", "J500"},
		{"Smith", "S530"},
		{"Johnson", "J525"},
		{"Williams", "W452"},
		{"Brown", "B650"},
	}
	
	for _, tc := range testCases {
		result := service.GetPhoneticCode(tc.input)
		if result != tc.expected {
			t.Errorf("For input '%s', expected '%s', got '%s'", tc.input, tc.expected, result)
		}
	}
}

func TestPrivacyService_GetBloomFilterStats(t *testing.T) {
	cfg := &config.Config{
		BloomFilterSize:     1000,
		BloomFilterHashCount: 5,
		BloomFilterFalsePositiveRate: 0.01,
		PhoneticEncodingEnabled: true,
	}
	
	service := NewPrivacyService(cfg)
	stats := service.GetBloomFilterStats()
	
	expectedKeys := []string{"size", "hash_count", "false_positive_rate", "phonetic_encoding_enabled"}
	for _, key := range expectedKeys {
		if _, exists := stats[key]; !exists {
			t.Errorf("Stats should contain key: %s", key)
		}
	}
	
	if stats["size"] != 1000 {
		t.Errorf("Expected size 1000, got %v", stats["size"])
	}
	
	if stats["hash_count"] != 5 {
		t.Errorf("Expected hash_count 5, got %v", stats["hash_count"])
	}
	
	if stats["false_positive_rate"] != 0.01 {
		t.Errorf("Expected false_positive_rate 0.01, got %v", stats["false_positive_rate"])
	}
	
	if stats["phonetic_encoding_enabled"] != true {
		t.Errorf("Expected phonetic_encoding_enabled true, got %v", stats["phonetic_encoding_enabled"])
	}
}

func TestPrivacyService_HealthCheck(t *testing.T) {
	cfg := &config.Config{}
	service := NewPrivacyService(cfg)
	
	ctx := context.Background()
	err := service.HealthCheck(ctx)
	
	if err != nil {
		t.Errorf("Health check should not fail: %v", err)
	}
} 