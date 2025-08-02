package services

import (
	"testing"
)

func TestSelectiveDisclosureService(t *testing.T) {
	// Create selective disclosure configuration
	config := NewSelectiveDisclosureConfig(true, true, "test-salt-123")

	// Create selective disclosure service
	service := NewSelectiveDisclosureService(config)

	t.Run("ExtractClaims", func(t *testing.T) {
		// Create test credential
		credential := map[string]interface{}{
			"name":       "John Doe",
			"email":      "john.doe@example.com",
			"age":        25,
			"salary":     75000.0,
			"ssn":        "123-45-6789",
			"phone":      "+1234567890",
			"address":    "123 Main St, City, State",
			"department": "Engineering",
			"start_date": "2020-01-15",
		}

		// Create disclosure request
		request := SelectiveDisclosureRequest{
			CredentialID: "cred-123",
			Claims: map[string]Claim{
				"name": {
					Name:       "name",
					Type:       "string",
					Required:   true,
					Disclosure: DisclosureLevelFull,
				},
				"email": {
					Name:       "email",
					Type:       "string",
					Required:   false,
					Disclosure: DisclosureLevelHash,
				},
				"age": {
					Name:       "age",
					Type:       "integer",
					Required:   false,
					Disclosure: DisclosureLevelRange,
				},
				"salary": {
					Name:       "salary",
					Type:       "float",
					Required:   false,
					Disclosure: DisclosureLevelRange,
				},
				"ssn": {
					Name:       "ssn",
					Type:       "string",
					Required:   false,
					Disclosure: DisclosureLevelNone,
				},
				"phone": {
					Name:       "phone",
					Type:       "string",
					Required:   false,
					Disclosure: DisclosureLevelProof,
				},
			},
			Purpose:     "employment_verification",
			RequesterID: "employer-456",
			Metadata: map[string]interface{}{
				"request_type": "background_check",
			},
		}

		response, err := service.ExtractClaims(credential, request)
		if err != nil {
			t.Fatalf("Failed to extract claims: %v", err)
		}

		if response == nil {
			t.Fatal("Expected non-nil response")
		}

		if response.CredentialID != "cred-123" {
			t.Errorf("Expected credential ID cred-123, got %s", response.CredentialID)
		}

		// Check disclosed claims
		if len(response.DisclosedClaims) == 0 {
			t.Error("Expected disclosed claims to be non-empty")
		}

		// Check that name is fully disclosed
		if name, exists := response.DisclosedClaims["name"]; !exists {
			t.Error("Expected name to be disclosed")
		} else if name != "John Doe" {
			t.Errorf("Expected name to be 'John Doe', got %v", name)
		}

		// Check that email is hashed
		if emailHash, exists := response.DisclosedClaims["email"]; !exists {
			t.Error("Expected email hash to be disclosed")
		} else {
			hashStr, ok := emailHash.(string)
			if !ok {
				t.Error("Expected email hash to be a string")
			} else if len(hashStr) != 64 {
				t.Errorf("Expected email hash to be 64 characters, got %d", len(hashStr))
			}
		}

		// Check that age is in range format
		if ageRange, exists := response.DisclosedClaims["age"]; !exists {
			t.Error("Expected age range to be disclosed")
		} else {
			rangeStr, ok := ageRange.(string)
			if !ok {
				t.Error("Expected age range to be a string")
			} else if rangeStr != "18-30" {
				t.Errorf("Expected age range to be '18-30', got %s", rangeStr)
			}
		}

		// Check that salary is in range format
		if salaryRange, exists := response.DisclosedClaims["salary"]; !exists {
			t.Error("Expected salary range to be disclosed")
		} else {
			rangeStr, ok := salaryRange.(string)
			if !ok {
				t.Error("Expected salary range to be a string")
			} else if rangeStr != "75000-75001" {
				t.Errorf("Expected salary range to be '75000-75001', got %s", rangeStr)
			}
		}

		// Check that SSN is hidden
		if _, exists := response.DisclosedClaims["ssn"]; exists {
			t.Error("Expected SSN to be hidden")
		}

		// Check that phone has a proof
		if _, exists := response.DisclosedClaims["phone"]; exists {
			t.Error("Expected phone to be hidden when using proof disclosure")
		}

		if len(response.Proofs) == 0 {
			t.Error("Expected proofs to be non-empty")
		}

		if phoneProof, exists := response.Proofs["phone"]; !exists {
			t.Error("Expected phone proof to be present")
		} else {
			proof, ok := phoneProof.(map[string]interface{})
			if !ok {
				t.Error("Expected phone proof to be a map")
			} else {
				if proof["type"] != "simple_proof" {
					t.Errorf("Expected proof type to be 'simple_proof', got %v", proof["type"])
				}
				if proof["claim_name"] != "phone" {
					t.Errorf("Expected proof claim name to be 'phone', got %v", proof["claim_name"])
				}
			}
		}

		// Check hidden claims
		expectedHidden := []string{"ssn", "phone"}
		for _, hidden := range expectedHidden {
			found := false
			for _, responseHidden := range response.HiddenClaims {
				if responseHidden == hidden {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("Expected %s to be in hidden claims", hidden)
			}
		}

		// Check audit log
		if response.AuditLog == nil {
			t.Error("Expected audit log to be present")
		} else {
			if response.AuditLog.CredentialID != "cred-123" {
				t.Errorf("Expected audit log credential ID to be cred-123, got %s", response.AuditLog.CredentialID)
			}
			if response.AuditLog.RequesterID != "employer-456" {
				t.Errorf("Expected audit log requester ID to be employer-456, got %s", response.AuditLog.RequesterID)
			}
			if response.AuditLog.Purpose != "employment_verification" {
				t.Errorf("Expected audit log purpose to be employment_verification, got %s", response.AuditLog.Purpose)
			}
			if response.AuditLog.DisclosedCount != 4 {
				t.Errorf("Expected disclosed count to be 4, got %d", response.AuditLog.DisclosedCount)
			}
			if response.AuditLog.HiddenCount != 2 {
				t.Errorf("Expected hidden count to be 2, got %d", response.AuditLog.HiddenCount)
			}
			if response.AuditLog.PrivacyHash == "" {
				t.Error("Expected privacy hash to be non-empty")
			}
		}

		// Check metadata
		if response.Metadata == nil {
			t.Error("Expected metadata to be non-nil")
		} else {
			if response.Metadata["privacy_hash"] == "" {
				t.Error("Expected privacy hash in metadata")
			}
			if response.Metadata["purpose"] != "employment_verification" {
				t.Errorf("Expected purpose in metadata to be employment_verification, got %v", response.Metadata["purpose"])
			}
			if response.Metadata["requester_id"] != "employer-456" {
				t.Errorf("Expected requester ID in metadata to be employer-456, got %v", response.Metadata["requester_id"])
			}
		}
	})

	t.Run("ValidateDisclosureRequest", func(t *testing.T) {
		t.Run("ValidRequest", func(t *testing.T) {
			request := SelectiveDisclosureRequest{
				CredentialID: "cred-123",
				Claims: map[string]Claim{
					"name": {
						Name:       "name",
						Type:       "string",
						Required:   true,
						Disclosure: DisclosureLevelFull,
					},
				},
				Purpose:     "verification",
				RequesterID: "requester-123",
			}

			err := service.validateDisclosureRequest(request)
			if err != nil {
				t.Errorf("Expected valid request, got error: %v", err)
			}
		})

		t.Run("EmptyCredentialID", func(t *testing.T) {
			request := SelectiveDisclosureRequest{
				CredentialID: "",
				Claims: map[string]Claim{
					"name": {
						Name:       "name",
						Type:       "string",
						Required:   true,
						Disclosure: DisclosureLevelFull,
					},
				},
				Purpose:     "verification",
				RequesterID: "requester-123",
			}

			err := service.validateDisclosureRequest(request)
			if err == nil {
				t.Error("Expected error for empty credential ID")
			}
		})

		t.Run("EmptyClaims", func(t *testing.T) {
			request := SelectiveDisclosureRequest{
				CredentialID: "cred-123",
				Claims:       map[string]Claim{},
				Purpose:      "verification",
				RequesterID:  "requester-123",
			}

			err := service.validateDisclosureRequest(request)
			if err == nil {
				t.Error("Expected error for empty claims")
			}
		})

		t.Run("EmptyPurpose", func(t *testing.T) {
			request := SelectiveDisclosureRequest{
				CredentialID: "cred-123",
				Claims: map[string]Claim{
					"name": {
						Name:       "name",
						Type:       "string",
						Required:   true,
						Disclosure: DisclosureLevelFull,
					},
				},
				Purpose:     "",
				RequesterID: "requester-123",
			}

			err := service.validateDisclosureRequest(request)
			if err == nil {
				t.Error("Expected error for empty purpose")
			}
		})

		t.Run("EmptyRequesterID", func(t *testing.T) {
			request := SelectiveDisclosureRequest{
				CredentialID: "cred-123",
				Claims: map[string]Claim{
					"name": {
						Name:       "name",
						Type:       "string",
						Required:   true,
						Disclosure: DisclosureLevelFull,
					},
				},
				Purpose:     "verification",
				RequesterID: "",
			}

			err := service.validateDisclosureRequest(request)
			if err == nil {
				t.Error("Expected error for empty requester ID")
			}
		})

		t.Run("InvalidDisclosureLevel", func(t *testing.T) {
			request := SelectiveDisclosureRequest{
				CredentialID: "cred-123",
				Claims: map[string]Claim{
					"name": {
						Name:       "name",
						Type:       "string",
						Required:   true,
						Disclosure: "invalid",
					},
				},
				Purpose:     "verification",
				RequesterID: "requester-123",
			}

			err := service.validateDisclosureRequest(request)
			if err == nil {
				t.Error("Expected error for invalid disclosure level")
			}
		})
	})

	t.Run("ValidateDisclosurePrivacy", func(t *testing.T) {
		response := &SelectiveDisclosureResponse{
			CredentialID: "cred-123",
			DisclosedClaims: map[string]interface{}{
				"name":  "John Doe",
				"email": "hash123",
			},
			HiddenClaims: []string{"ssn", "phone"},
			Metadata: map[string]interface{}{
				"privacy_hash": "hash456",
			},
		}

		err := service.ValidateDisclosurePrivacy(response)
		if err != nil {
			t.Errorf("Expected valid privacy, got error: %v", err)
		}

		// Test with hidden claim that is also disclosed
		response.DisclosedClaims["ssn"] = "123-45-6789"
		err = service.ValidateDisclosurePrivacy(response)
		if err == nil {
			t.Error("Expected error for hidden claim that is disclosed")
		}

		// Test with missing privacy hash
		response.DisclosedClaims = map[string]interface{}{
			"name": "John Doe",
		}
		response.HiddenClaims = []string{"ssn"}
		response.Metadata = map[string]interface{}{}
		err = service.ValidateDisclosurePrivacy(response)
		if err == nil {
			t.Error("Expected error for missing privacy hash")
		}
	})

	t.Run("GetDisclosureStats", func(t *testing.T) {
		stats := service.GetDisclosureStats()

		if stats == nil {
			t.Error("Expected non-nil stats")
		}

		// Check required fields
		requiredFields := []string{
			"minimal_disclosure_enabled",
			"audit_logging_enabled",
			"hash_algorithm",
			"supported_disclosure_levels",
		}

		for _, field := range requiredFields {
			if _, exists := stats[field]; !exists {
				t.Errorf("Expected field %s in stats", field)
			}
		}

		// Check specific values
		if stats["minimal_disclosure_enabled"] != true {
			t.Errorf("Expected minimal_disclosure_enabled to be true, got %v", stats["minimal_disclosure_enabled"])
		}

		if stats["audit_logging_enabled"] != true {
			t.Errorf("Expected audit_logging_enabled to be true, got %v", stats["audit_logging_enabled"])
		}

		if stats["hash_algorithm"] != "SHA-256" {
			t.Errorf("Expected hash_algorithm to be SHA-256, got %v", stats["hash_algorithm"])
		}

		// Check supported disclosure levels
		supportedLevels, ok := stats["supported_disclosure_levels"].([]string)
		if !ok {
			t.Error("Expected supported_disclosure_levels to be a slice")
		} else {
			expectedLevels := []string{"full", "hash", "range", "proof", "none"}
			if len(supportedLevels) != len(expectedLevels) {
				t.Errorf("Expected %d supported levels, got %d", len(expectedLevels), len(supportedLevels))
			}
		}
	})

	t.Run("RangeValueCreation", func(t *testing.T) {
		t.Run("AgeRange", func(t *testing.T) {
			testCases := []struct {
				age    int
				expect string
			}{
				{17, "under-18"},
				{25, "18-30"},
				{35, "30-50"},
				{55, "50-65"},
				{70, "65-plus"},
			}

			for _, tc := range testCases {
				result := service.createAgeRange(tc.age)
				if result != tc.expect {
					t.Errorf("For age %d, expected %s, got %s", tc.age, tc.expect, result)
				}
			}
		})

		t.Run("NumericRange", func(t *testing.T) {
			testCases := []struct {
				value  int
				expect string
			}{
				{25, "20-29"},
				{30, "30-39"},
				{45, "40-49"},
			}

			for _, tc := range testCases {
				result := service.createNumericRange(tc.value)
				if result != tc.expect {
					t.Errorf("For value %d, expected %s, got %s", tc.value, tc.expect, result)
				}
			}
		})

		t.Run("FloatRange", func(t *testing.T) {
			testCases := []struct {
				value  float64
				expect string
			}{
				{25.5, "25-26"},
				{30.0, "30-31"},
				{45.7, "45-46"},
			}

			for _, tc := range testCases {
				result := service.createFloatRange(tc.value)
				if result != tc.expect {
					t.Errorf("For value %f, expected %s, got %s", tc.value, tc.expect, result)
				}
			}
		})
	})

	t.Run("HashValue", func(t *testing.T) {
		hash, err := service.hashValue("test_claim", "test_value")
		if err != nil {
			t.Fatalf("Failed to hash value: %v", err)
		}

		if len(hash) != 64 {
			t.Errorf("Expected hash length 64, got %d", len(hash))
		}

		// Test that same input produces same hash
		hash2, err := service.hashValue("test_claim", "test_value")
		if err != nil {
			t.Fatalf("Failed to hash value second time: %v", err)
		}

		if hash != hash2 {
			t.Error("Same input should produce same hash")
		}

		// Test that different input produces different hash
		hash3, err := service.hashValue("test_claim", "different_value")
		if err != nil {
			t.Fatalf("Failed to hash different value: %v", err)
		}

		if hash == hash3 {
			t.Error("Different input should produce different hash")
		}
	})

	t.Run("CreateProof", func(t *testing.T) {
		claim := Claim{
			Name:       "test_claim",
			Type:       "string",
			Required:   true,
			Disclosure: DisclosureLevelProof,
			Metadata: map[string]interface{}{
				"proof_type": "age_verification",
			},
		}

		proof, err := service.createProof("test_claim", "test_value", claim)
		if err != nil {
			t.Fatalf("Failed to create proof: %v", err)
		}

		proofMap, ok := proof.(map[string]interface{})
		if !ok {
			t.Fatal("Expected proof to be a map")
		}

		if proofMap["type"] != "simple_proof" {
			t.Errorf("Expected proof type to be 'simple_proof', got %v", proofMap["type"])
		}

		if proofMap["claim_name"] != "test_claim" {
			t.Errorf("Expected claim name to be 'test_claim', got %v", proofMap["claim_name"])
		}

		if proofMap["algorithm"] != "SHA-256" {
			t.Errorf("Expected algorithm to be 'SHA-256', got %v", proofMap["algorithm"])
		}

		if proofMap["proof_hash"] == "" {
			t.Error("Expected proof hash to be non-empty")
		}

		if proofMap["metadata"] == nil {
			t.Error("Expected metadata to be present")
		}
	})
}
