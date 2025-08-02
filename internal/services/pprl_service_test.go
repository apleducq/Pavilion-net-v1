package services

import (
	"testing"
	"time"
)

func TestPPRLService(t *testing.T) {
	// Create PPRL configuration
	config := NewPPRLConfig(1000, 5, 0.01, "test-salt-123")
	
	// Create PPRL service
	service := NewPPRLService(config)
	
	t.Run("HashSensitiveField", func(t *testing.T) {
		field := SensitiveField{
			Name:  "email",
			Value: "john.doe@example.com",
			Type:  "exact",
		}
		
		hash, err := service.HashSensitiveField(field)
		if err != nil {
			t.Fatalf("Failed to hash sensitive field: %v", err)
		}
		
		if len(hash) != 64 { // SHA-256 produces 64 hex characters
			t.Errorf("Expected hash length 64, got %d", len(hash))
		}
		
		// Test that same field produces same hash
		hash2, err := service.HashSensitiveField(field)
		if err != nil {
			t.Fatalf("Failed to hash sensitive field second time: %v", err)
		}
		
		if hash != hash2 {
			t.Error("Same field should produce same hash")
		}
		
		// Test different field types
		t.Run("FuzzyType", func(t *testing.T) {
			fuzzyField := SensitiveField{
				Name:  "name",
				Value: "John Doe",
				Type:  "fuzzy",
			}
			
			hash, err := service.HashSensitiveField(fuzzyField)
			if err != nil {
				t.Fatalf("Failed to hash fuzzy field: %v", err)
			}
			
			if len(hash) != 64 {
				t.Errorf("Expected hash length 64, got %d", len(hash))
			}
		})
		
		t.Run("PhoneticType", func(t *testing.T) {
			phoneticField := SensitiveField{
				Name:  "name",
				Value: "John Doe",
				Type:  "phonetic",
			}
			
			hash, err := service.HashSensitiveField(phoneticField)
			if err != nil {
				t.Fatalf("Failed to hash phonetic field: %v", err)
			}
			
			if len(hash) != 64 {
				t.Errorf("Expected hash length 64, got %d", len(hash))
			}
		})
	})
	
	t.Run("CreateBloomFilterForRecord", func(t *testing.T) {
		record := DataProviderRecord{
			ID:       "record-123",
			Provider: "test-provider",
			Fields: map[string]SensitiveField{
				"email": {
					Name:  "email",
					Value: "john.doe@example.com",
					Type:  "exact",
				},
				"name": {
					Name:  "name",
					Value: "John Doe",
					Type:  "fuzzy",
				},
				"phone": {
					Name:  "phone",
					Value: "+1234567890",
					Type:  "exact",
				},
			},
			Created: time.Now(),
		}
		
		bloomFilter, err := service.CreateBloomFilterForRecord(record)
		if err != nil {
			t.Fatalf("Failed to create Bloom filter for record: %v", err)
		}
		
		if bloomFilter == nil {
			t.Error("Expected non-nil Bloom filter")
		}
		
		// Test that the Bloom filter contains the hashed values
		for fieldName, field := range record.Fields {
			hashedValue, err := service.HashSensitiveField(field)
			if err != nil {
				t.Fatalf("Failed to hash field %s: %v", fieldName, err)
			}
			
			if !bloomFilter.Contains(hashedValue) {
				t.Errorf("Bloom filter should contain hash for field %s", fieldName)
			}
		}
	})
	
	t.Run("PerformPPRL", func(t *testing.T) {
		// Create test provider records
		providerRecords := []DataProviderRecord{
			{
				ID:       "record-1",
				Provider: "provider-1",
				Fields: map[string]SensitiveField{
					"email": {
						Name:  "email",
						Value: "john.doe@example.com",
						Type:  "exact",
					},
					"name": {
						Name:  "name",
						Value: "John Doe",
						Type:  "fuzzy",
					},
				},
				Created: time.Now(),
			},
			{
				ID:       "record-2",
				Provider: "provider-1",
				Fields: map[string]SensitiveField{
					"email": {
						Name:  "jane.smith@example.com",
						Value: "jane.smith@example.com",
						Type:  "exact",
					},
					"name": {
						Name:  "name",
						Value: "Jane Smith",
						Type:  "fuzzy",
					},
				},
				Created: time.Now(),
			},
		}
		
		// Create PPRL request
		request := PPRLRequest{
			QueryFields: map[string]SensitiveField{
				"email": {
					Name:  "email",
					Value: "john.doe@example.com",
					Type:  "exact",
				},
				"name": {
					Name:  "name",
					Value: "John Doe",
					Type:  "fuzzy",
				},
			},
			ProviderID: "provider-1",
			Threshold:  0.5,
		}
		
		response, err := service.PerformPPRL(request, providerRecords)
		if err != nil {
			t.Fatalf("Failed to perform PPRL: %v", err)
		}
		
		if response == nil {
			t.Fatal("Expected non-nil response")
		}
		
		if !response.MatchFound {
			t.Error("Expected match to be found")
		}
		
		if response.Confidence < 0.5 {
			t.Errorf("Expected confidence >= 0.5, got %f", response.Confidence)
		}
		
		if response.ProviderID != "provider-1" {
			t.Errorf("Expected provider ID provider-1, got %s", response.ProviderID)
		}
		
		if len(response.MatchedFields) == 0 {
			t.Error("Expected matched fields to be non-empty")
		}
		
		if response.BloomFilter == "" {
			t.Error("Expected Bloom filter to be non-empty")
		}
		
		// Test metadata
		if response.Metadata == nil {
			t.Error("Expected metadata to be non-nil")
		}
		
		queryFieldsCount, ok := response.Metadata["query_fields_count"].(int)
		if !ok || queryFieldsCount != 2 {
			t.Errorf("Expected query_fields_count to be 2, got %v", queryFieldsCount)
		}
		
		providerRecordsCount, ok := response.Metadata["provider_records_count"].(int)
		if !ok || providerRecordsCount != 2 {
			t.Errorf("Expected provider_records_count to be 2, got %v", providerRecordsCount)
		}
	})
	
	t.Run("PerformPPRL_NoMatch", func(t *testing.T) {
		providerRecords := []DataProviderRecord{
			{
				ID:       "record-1",
				Provider: "provider-1",
				Fields: map[string]SensitiveField{
					"email": {
						Name:  "jane.smith@example.com",
						Value: "jane.smith@example.com",
						Type:  "exact",
					},
				},
				Created: time.Now(),
			},
		}
		
		request := PPRLRequest{
			QueryFields: map[string]SensitiveField{
				"email": {
					Name:  "email",
					Value: "john.doe@example.com",
					Type:  "exact",
				},
			},
			ProviderID: "provider-1",
			Threshold:  0.5,
		}
		
		response, err := service.PerformPPRL(request, providerRecords)
		if err != nil {
			t.Fatalf("Failed to perform PPRL: %v", err)
		}
		
		if response.MatchFound {
			t.Error("Expected no match to be found")
		}
		
		if response.Confidence > 0 {
			t.Errorf("Expected confidence to be 0, got %f", response.Confidence)
		}
	})
	
	t.Run("ValidatePPRLRequest", func(t *testing.T) {
		t.Run("ValidRequest", func(t *testing.T) {
			request := PPRLRequest{
				QueryFields: map[string]SensitiveField{
					"email": {
						Name:  "email",
						Value: "john.doe@example.com",
						Type:  "exact",
					},
				},
				ProviderID: "provider-1",
				Threshold:  0.5,
			}
			
			err := service.ValidatePPRLRequest(request)
			if err != nil {
				t.Errorf("Expected valid request, got error: %v", err)
			}
		})
		
		t.Run("EmptyQueryFields", func(t *testing.T) {
			request := PPRLRequest{
				QueryFields: map[string]SensitiveField{},
				ProviderID:  "provider-1",
				Threshold:   0.5,
			}
			
			err := service.ValidatePPRLRequest(request)
			if err == nil {
				t.Error("Expected error for empty query fields")
			}
		})
		
		t.Run("EmptyProviderID", func(t *testing.T) {
			request := PPRLRequest{
				QueryFields: map[string]SensitiveField{
					"email": {
						Name:  "email",
						Value: "john.doe@example.com",
						Type:  "exact",
					},
				},
				ProviderID: "",
				Threshold:  0.5,
			}
			
			err := service.ValidatePPRLRequest(request)
			if err == nil {
				t.Error("Expected error for empty provider ID")
			}
		})
		
		t.Run("InvalidThreshold", func(t *testing.T) {
			request := PPRLRequest{
				QueryFields: map[string]SensitiveField{
					"email": {
						Name:  "email",
						Value: "john.doe@example.com",
						Type:  "exact",
					},
				},
				ProviderID: "provider-1",
				Threshold:  1.5, // Invalid threshold
			}
			
			err := service.ValidatePPRLRequest(request)
			if err == nil {
				t.Error("Expected error for invalid threshold")
			}
		})
		
		t.Run("InvalidFieldType", func(t *testing.T) {
			request := PPRLRequest{
				QueryFields: map[string]SensitiveField{
					"email": {
						Name:  "email",
						Value: "john.doe@example.com",
						Type:  "invalid",
					},
				},
				ProviderID: "provider-1",
				Threshold:  0.5,
			}
			
			err := service.ValidatePPRLRequest(request)
			if err == nil {
				t.Error("Expected error for invalid field type")
			}
		})
	})
	
	t.Run("GetPPRLStats", func(t *testing.T) {
		stats := service.GetPPRLStats()
		
		if stats == nil {
			t.Error("Expected non-nil stats")
		}
		
		// Check required fields
		requiredFields := []string{
			"bloom_filter_size",
			"bloom_filter_hash_count",
			"bloom_filter_false_positive_rate",
			"hash_algorithm",
			"current_false_positive_rate",
		}
		
		for _, field := range requiredFields {
			if _, exists := stats[field]; !exists {
				t.Errorf("Expected field %s in stats", field)
			}
		}
		
		// Check specific values
		if stats["bloom_filter_size"] != 1000 {
			t.Errorf("Expected bloom_filter_size to be 1000, got %v", stats["bloom_filter_size"])
		}
		
		if stats["bloom_filter_hash_count"] != 5 {
			t.Errorf("Expected bloom_filter_hash_count to be 5, got %v", stats["bloom_filter_hash_count"])
		}
		
		if stats["hash_algorithm"] != "SHA-256" {
			t.Errorf("Expected hash_algorithm to be SHA-256, got %v", stats["hash_algorithm"])
		}
	})
	
	t.Run("ExportImportBloomFilter", func(t *testing.T) {
		// Create a test Bloom filter
		bloomFilter := NewBloomFilter(100, 3)
		bloomFilter.Add("test-value-1")
		bloomFilter.Add("test-value-2")
		
		// Export the Bloom filter
		exported, err := service.ExportBloomFilter(bloomFilter)
		if err != nil {
			t.Fatalf("Failed to export Bloom filter: %v", err)
		}
		
		if exported == nil {
			t.Error("Expected non-nil exported data")
		}
		
		// Import the Bloom filter
		imported, err := service.ImportBloomFilter(exported)
		if err != nil {
			t.Fatalf("Failed to import Bloom filter: %v", err)
		}
		
		if imported == nil {
			t.Error("Expected non-nil imported Bloom filter")
		}
		
		// Verify that the imported Bloom filter contains the same values
		if !imported.Contains("test-value-1") {
			t.Error("Imported Bloom filter should contain test-value-1")
		}
		
		if !imported.Contains("test-value-2") {
			t.Error("Imported Bloom filter should contain test-value-2")
		}
		
		if imported.Contains("test-value-3") {
			t.Error("Imported Bloom filter should not contain test-value-3")
		}
	})
	
	t.Run("PreprocessForFuzzy", func(t *testing.T) {
		testCases := []struct {
			input    string
			expected string
		}{
			{"Mr John Doe", "john doe"},
			{"Mrs Jane Smith", "jane smith"},
			{"Dr Robert Johnson", "robert johnson"},
			{"John-Doe", "john doe"},
			{"John   Doe", "john doe"},
			{"JOHN DOE", "john doe"},
		}
		
		for _, tc := range testCases {
			result := service.preprocessForFuzzy(tc.input)
			if result != tc.expected {
				t.Errorf("For input '%s', expected '%s', got '%s'", tc.input, tc.expected, result)
			}
		}
	})
} 