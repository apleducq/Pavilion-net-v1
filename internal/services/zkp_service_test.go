package services

import (
	"testing"
	"time"
)

func TestZKPService(t *testing.T) {
	// Create ZKP service
	config := NewZKPConfig(30*time.Second, 1024, "test-salt", true)
	service := NewZKPService(config)

	t.Run("GenerateAgeProof", func(t *testing.T) {
		request := ZKPRequest{
			ProofType: "age_verification",
			Statement: "User is at least 18 years old",
			Witness: map[string]interface{}{
				"age": 25.0,
			},
			PublicInputs: map[string]interface{}{
				"minimum_age": 18.0,
			},
			Metadata: map[string]interface{}{
				"purpose": "age_verification",
			},
		}

		response, err := service.GenerateProof(request)
		if err != nil {
			t.Fatalf("Failed to generate age proof: %v", err)
		}

		if response.ProofID == "" {
			t.Error("Proof ID should not be empty")
		}

		if response.ProofType != "age_verification" {
			t.Errorf("Expected proof type 'age_verification', got '%s'", response.ProofType)
		}

		if response.Proof == "" {
			t.Error("Proof should not be empty")
		}

		if response.VerificationKey == "" {
			t.Error("Verification key should not be empty")
		}
	})

	t.Run("GenerateRangeProof", func(t *testing.T) {
		request := ZKPRequest{
			ProofType: "range_proof",
			Statement: "Salary is between 50000 and 100000",
			Witness: map[string]interface{}{
				"value": 75000.0,
			},
			PublicInputs: map[string]interface{}{
				"min_value": 50000.0,
				"max_value": 100000.0,
			},
			Metadata: map[string]interface{}{
				"purpose": "salary_verification",
			},
		}

		response, err := service.GenerateProof(request)
		if err != nil {
			t.Fatalf("Failed to generate range proof: %v", err)
		}

		if response.ProofType != "range_proof" {
			t.Errorf("Expected proof type 'range_proof', got '%s'", response.ProofType)
		}

		if response.Proof == "" {
			t.Error("Proof should not be empty")
		}
	})

	t.Run("GenerateMembershipProof", func(t *testing.T) {
		request := ZKPRequest{
			ProofType: "membership_proof",
			Statement: "User is in approved list",
			Witness: map[string]interface{}{
				"element": "user123",
			},
			PublicInputs: map[string]interface{}{
				"set": []interface{}{"user123", "user456", "user789"},
			},
			Metadata: map[string]interface{}{
				"purpose": "access_control",
			},
		}

		response, err := service.GenerateProof(request)
		if err != nil {
			t.Fatalf("Failed to generate membership proof: %v", err)
		}

		if response.ProofType != "membership_proof" {
			t.Errorf("Expected proof type 'membership_proof', got '%s'", response.ProofType)
		}

		if response.Proof == "" {
			t.Error("Proof should not be empty")
		}
	})

	t.Run("GenerateEqualityProof", func(t *testing.T) {
		request := ZKPRequest{
			ProofType: "equality_proof",
			Statement: "Two values are equal",
			Witness: map[string]interface{}{
				"value1": "secret123",
				"value2": "secret123",
			},
			Metadata: map[string]interface{}{
				"purpose": "equality_verification",
			},
		}

		response, err := service.GenerateProof(request)
		if err != nil {
			t.Fatalf("Failed to generate equality proof: %v", err)
		}

		if response.ProofType != "equality_proof" {
			t.Errorf("Expected proof type 'equality_proof', got '%s'", response.ProofType)
		}

		if response.Proof == "" {
			t.Error("Proof should not be empty")
		}
	})

	t.Run("VerifyAgeProof", func(t *testing.T) {
		// First generate a proof
		generateRequest := ZKPRequest{
			ProofType: "age_verification",
			Statement: "User is at least 18 years old",
			Witness: map[string]interface{}{
				"age": 25.0,
			},
			PublicInputs: map[string]interface{}{
				"minimum_age": 18.0,
			},
		}

		generateResponse, err := service.GenerateProof(generateRequest)
		if err != nil {
			t.Fatalf("Failed to generate proof for verification: %v", err)
		}

		// Now verify the proof
		verifyRequest := ZKPVerificationRequest{
			ProofID:   generateResponse.ProofID,
			Proof:     generateResponse.Proof,
			Statement: generateResponse.Statement,
		}

		verifyResponse, err := service.VerifyProof(verifyRequest)
		if err != nil {
			t.Fatalf("Failed to verify proof: %v", err)
		}

		if !verifyResponse.Valid {
			t.Error("Proof verification should be valid")
		}

		if verifyResponse.ProofID != generateResponse.ProofID {
			t.Error("Proof ID should match")
		}
	})

	t.Run("VerifyRangeProof", func(t *testing.T) {
		// Generate a range proof
		generateRequest := ZKPRequest{
			ProofType: "range_proof",
			Statement: "Value is in range",
			Witness: map[string]interface{}{
				"value": 75.0,
			},
			PublicInputs: map[string]interface{}{
				"min_value": 50.0,
				"max_value": 100.0,
			},
		}

		generateResponse, err := service.GenerateProof(generateRequest)
		if err != nil {
			t.Fatalf("Failed to generate range proof: %v", err)
		}

		// Verify the proof
		verifyRequest := ZKPVerificationRequest{
			ProofID:   generateResponse.ProofID,
			Proof:     generateResponse.Proof,
			Statement: generateResponse.Statement,
		}

		verifyResponse, err := service.VerifyProof(verifyRequest)
		if err != nil {
			t.Fatalf("Failed to verify range proof: %v", err)
		}

		if !verifyResponse.Valid {
			t.Error("Range proof verification should be valid")
		}
	})

	t.Run("ValidateZKPRequest", func(t *testing.T) {
		// Test valid request
		validRequest := ZKPRequest{
			ProofType: "age_verification",
			Statement: "Test statement",
			Witness: map[string]interface{}{
				"age": 25.0,
			},
		}

		err := service.validateZKPRequest(validRequest)
		if err != nil {
			t.Errorf("Valid request should not return error: %v", err)
		}

		// Test invalid proof type
		invalidRequest := ZKPRequest{
			ProofType: "invalid_type",
			Statement: "Test statement",
			Witness: map[string]interface{}{
				"age": 25.0,
			},
		}

		err = service.validateZKPRequest(invalidRequest)
		if err == nil {
			t.Error("Invalid proof type should return error")
		}

		// Test missing proof type
		missingTypeRequest := ZKPRequest{
			Statement: "Test statement",
			Witness: map[string]interface{}{
				"age": 25.0,
			},
		}

		err = service.validateZKPRequest(missingTypeRequest)
		if err == nil {
			t.Error("Missing proof type should return error")
		}

		// Test missing statement
		missingStatementRequest := ZKPRequest{
			ProofType: "age_verification",
			Witness: map[string]interface{}{
				"age": 25.0,
			},
		}

		err = service.validateZKPRequest(missingStatementRequest)
		if err == nil {
			t.Error("Missing statement should return error")
		}

		// Test missing witness
		missingWitnessRequest := ZKPRequest{
			ProofType: "age_verification",
			Statement: "Test statement",
		}

		err = service.validateZKPRequest(missingWitnessRequest)
		if err == nil {
			t.Error("Missing witness should return error")
		}
	})

	t.Run("ValidateVerificationRequest", func(t *testing.T) {
		// Test valid verification request
		validRequest := ZKPVerificationRequest{
			Proof:     "test_proof",
			Statement: "Test statement",
		}

		err := service.validateVerificationRequest(validRequest)
		if err != nil {
			t.Errorf("Valid verification request should not return error: %v", err)
		}

		// Test missing proof
		missingProofRequest := ZKPVerificationRequest{
			Statement: "Test statement",
		}

		err = service.validateVerificationRequest(missingProofRequest)
		if err == nil {
			t.Error("Missing proof should return error")
		}

		// Test missing statement
		missingStatementRequest := ZKPVerificationRequest{
			Proof: "test_proof",
		}

		err = service.validateVerificationRequest(missingStatementRequest)
		if err == nil {
			t.Error("Missing statement should return error")
		}
	})

	t.Run("GetZKPStats", func(t *testing.T) {
		stats := service.GetZKPStats()

		if stats["proof_timeout"] == nil {
			t.Error("Stats should include proof timeout")
		}

		if stats["max_proof_size"] == nil {
			t.Error("Stats should include max proof size")
		}

		if stats["hash_algorithm"] == nil {
			t.Error("Stats should include hash algorithm")
		}

		if stats["audit_log_enabled"] == nil {
			t.Error("Stats should include audit log enabled")
		}

		supportedTypes, ok := stats["supported_proof_types"].([]string)
		if !ok {
			t.Error("Stats should include supported proof types")
		}

		expectedTypes := []string{"age_verification", "range_proof", "membership_proof", "equality_proof"}
		if len(supportedTypes) != len(expectedTypes) {
			t.Errorf("Expected %d supported types, got %d", len(expectedTypes), len(supportedTypes))
		}
	})

	t.Run("GetSupportedCircuits", func(t *testing.T) {
		circuits := service.GetSupportedCircuits()

		if len(circuits) != 4 {
			t.Errorf("Expected 4 circuits, got %d", len(circuits))
		}

		// Check for specific circuits
		foundAge := false
		foundRange := false
		foundMembership := false
		foundEquality := false

		for _, circuit := range circuits {
			switch circuit.Name {
			case "age_verification":
				foundAge = true
			case "range_proof":
				foundRange = true
			case "membership_proof":
				foundMembership = true
			case "equality_proof":
				foundEquality = true
			}
		}

		if !foundAge {
			t.Error("Should include age_verification circuit")
		}

		if !foundRange {
			t.Error("Should include range_proof circuit")
		}

		if !foundMembership {
			t.Error("Should include membership_proof circuit")
		}

		if !foundEquality {
			t.Error("Should include equality_proof circuit")
		}
	})

	t.Run("ErrorHandling", func(t *testing.T) {
		// Test unsupported proof type
		unsupportedRequest := ZKPRequest{
			ProofType: "unsupported_type",
			Statement: "Test statement",
			Witness: map[string]interface{}{
				"test": "value",
			},
		}

		_, err := service.GenerateProof(unsupportedRequest)
		if err == nil {
			t.Error("Unsupported proof type should return error")
		}

		// Test missing required fields in age proof
		invalidAgeRequest := ZKPRequest{
			ProofType: "age_verification",
			Statement: "Test statement",
			Witness: map[string]interface{}{
				"wrong_field": 25.0,
			},
		}

		_, err = service.GenerateProof(invalidAgeRequest)
		if err == nil {
			t.Error("Missing age field should return error")
		}

		// Test missing public inputs in age proof
		missingPublicInputsRequest := ZKPRequest{
			ProofType: "age_verification",
			Statement: "Test statement",
			Witness: map[string]interface{}{
				"age": 25.0,
			},
		}

		_, err = service.GenerateProof(missingPublicInputsRequest)
		if err == nil {
			t.Error("Missing public inputs should return error")
		}
	})

	t.Run("ProofIDGeneration", func(t *testing.T) {
		request1 := ZKPRequest{
			ProofType: "age_verification",
			Statement: "Test statement 1",
			Witness: map[string]interface{}{
				"age": 25.0,
			},
		}

		request2 := ZKPRequest{
			ProofType: "age_verification",
			Statement: "Test statement 2",
			Witness: map[string]interface{}{
				"age": 25.0,
			},
		}

		id1 := service.generateProofID(request1)
		id2 := service.generateProofID(request2)

		if id1 == id2 {
			t.Error("Different requests should generate different proof IDs")
		}

		if len(id1) == 0 {
			t.Error("Proof ID should not be empty")
		}
	})
}

func TestZKPService_Integration(t *testing.T) {
	config := NewZKPConfig(30*time.Second, 1024, "integration-test-salt", true)
	service := NewZKPService(config)

	t.Run("FullWorkflow", func(t *testing.T) {
		// Generate proof
		generateRequest := ZKPRequest{
			ProofType: "age_verification",
			Statement: "User is at least 21 years old",
			Witness: map[string]interface{}{
				"age": 25.0,
			},
			PublicInputs: map[string]interface{}{
				"minimum_age": 21.0,
			},
			Metadata: map[string]interface{}{
				"purpose":   "alcohol_verification",
				"requester": "bar_owner",
			},
		}

		generateResponse, err := service.GenerateProof(generateRequest)
		if err != nil {
			t.Fatalf("Failed to generate proof: %v", err)
		}

		// Verify the generated proof
		verifyRequest := ZKPVerificationRequest{
			ProofID:   generateResponse.ProofID,
			Proof:     generateResponse.Proof,
			Statement: generateResponse.Statement,
		}

		verifyResponse, err := service.VerifyProof(verifyRequest)
		if err != nil {
			t.Fatalf("Failed to verify proof: %v", err)
		}

		if !verifyResponse.Valid {
			t.Error("Generated proof should be valid")
		}

		// Verify metadata is preserved
		if generateResponse.Metadata["purpose"] != "alcohol_verification" {
			t.Error("Metadata should be preserved in response")
		}
	})

	t.Run("MultipleProofTypes", func(t *testing.T) {
		proofTypes := []string{"age_verification", "range_proof", "membership_proof", "equality_proof"}

		for _, proofType := range proofTypes {
			t.Run(proofType, func(t *testing.T) {
				var request ZKPRequest

				switch proofType {
				case "age_verification":
					request = ZKPRequest{
						ProofType: proofType,
						Statement: "Age verification test",
						Witness: map[string]interface{}{
							"age": 30.0,
						},
						PublicInputs: map[string]interface{}{
							"minimum_age": 18.0,
						},
					}
				case "range_proof":
					request = ZKPRequest{
						ProofType: proofType,
						Statement: "Range proof test",
						Witness: map[string]interface{}{
							"value": 50.0,
						},
						PublicInputs: map[string]interface{}{
							"min_value": 0.0,
							"max_value": 100.0,
						},
					}
				case "membership_proof":
					request = ZKPRequest{
						ProofType: proofType,
						Statement: "Membership proof test",
						Witness: map[string]interface{}{
							"element": "test_element",
						},
						PublicInputs: map[string]interface{}{
							"set": []interface{}{"test_element", "other_element"},
						},
					}
				case "equality_proof":
					request = ZKPRequest{
						ProofType: proofType,
						Statement: "Equality proof test",
						Witness: map[string]interface{}{
							"value1": "test_value",
							"value2": "test_value",
						},
					}
				}

				response, err := service.GenerateProof(request)
				if err != nil {
					t.Fatalf("Failed to generate %s proof: %v", proofType, err)
				}

				if response.ProofType != proofType {
					t.Errorf("Expected proof type %s, got %s", proofType, response.ProofType)
				}

				if response.Proof == "" {
					t.Errorf("Proof should not be empty for %s", proofType)
				}
			})
		}
	})
}
