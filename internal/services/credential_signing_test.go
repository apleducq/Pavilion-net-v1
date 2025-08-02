package services

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"fmt"
	"testing"
	"time"

	"github.com/pavilion-trust/core-broker/internal/models"
)

func TestCredentialSigningService(t *testing.T) {
	// Generate test keys
	rsaKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("Failed to generate RSA key: %v", err)
	}

	ecdsaKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("Failed to generate ECDSA key: %v", err)
	}

	// Create signing service
	service := NewCredentialSigningService(rsaKey, ecdsaKey, "test-key-1", "test-issuer")

	// Create test credential
	credential := &models.Credential{
		ID:           "cred-123",
		Type:         "StudentCredential",
		Issuer:       "university.edu",
		Subject:      "student-456",
		IssuanceDate: time.Now().Format(time.RFC3339),
		Version:      "1.0",
		Claims: map[string]interface{}{
			"program": "Computer Science",
			"status":  "enrolled",
		},
		Status: "valid",
	}

	t.Run("GetSupportedMethods", func(t *testing.T) {
		methods := service.GetSupportedMethods()
		expectedMethods := []SigningMethod{SigningMethodJWT, SigningMethodLDProof, SigningMethodECDSA, SigningMethodRSA}

		if len(methods) != len(expectedMethods) {
			t.Errorf("Expected %d methods, got %d", len(expectedMethods), len(methods))
		}

		// Check that all expected methods are present
		for _, expected := range expectedMethods {
			found := false
			for _, method := range methods {
				if method == expected {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("Expected method %s not found", expected)
			}
		}
	})

	t.Run("SignJWT", func(t *testing.T) {
		result, err := service.SignCredential(credential, SigningMethodJWT)
		if err != nil {
			t.Fatalf("Failed to sign credential with JWT: %v", err)
		}

		if result.Method != SigningMethodJWT {
			t.Errorf("Expected method JWT, got %s", result.Method)
		}

		if result.Signature == "" {
			t.Error("Expected signature to be non-empty")
		}

		if result.KeyID != "test-key-1" {
			t.Errorf("Expected key ID test-key-1, got %s", result.KeyID)
		}

		if result.Algorithm != "RS256" {
			t.Errorf("Expected algorithm RS256, got %s", result.Algorithm)
		}

		// Verify the signature
		valid, err := service.VerifySignature(credential, result)
		if err != nil {
			t.Fatalf("Failed to verify JWT signature: %v", err)
		}

		if !valid {
			t.Error("Expected JWT signature to be valid")
		}
	})

	t.Run("SignLDProof", func(t *testing.T) {
		result, err := service.SignCredential(credential, SigningMethodLDProof)
		if err != nil {
			t.Fatalf("Failed to sign credential with LD-Proof: %v", err)
		}

		if result.Method != SigningMethodLDProof {
			t.Errorf("Expected method LD-PROOF, got %s", result.Method)
		}

		if result.Signature == "" {
			t.Error("Expected signature to be non-empty")
		}

		if result.KeyID != "test-key-1" {
			t.Errorf("Expected key ID test-key-1, got %s", result.KeyID)
		}

		if result.Algorithm != "Ed25519" {
			t.Errorf("Expected algorithm Ed25519, got %s", result.Algorithm)
		}

		// Verify the signature
		valid, err := service.VerifySignature(credential, result)
		if err != nil {
			t.Fatalf("Failed to verify LD-Proof signature: %v", err)
		}

		if !valid {
			t.Error("Expected LD-Proof signature to be valid")
		}
	})

	t.Run("SignECDSA", func(t *testing.T) {
		result, err := service.SignCredential(credential, SigningMethodECDSA)
		if err != nil {
			t.Fatalf("Failed to sign credential with ECDSA: %v", err)
		}

		if result.Method != SigningMethodECDSA {
			t.Errorf("Expected method ECDSA, got %s", result.Method)
		}

		if result.Signature == "" {
			t.Error("Expected signature to be non-empty")
		}

		if result.KeyID != "test-key-1" {
			t.Errorf("Expected key ID test-key-1, got %s", result.KeyID)
		}

		if result.Algorithm != "ECDSA-SHA256" {
			t.Errorf("Expected algorithm ECDSA-SHA256, got %s", result.Algorithm)
		}

		// Verify the signature
		valid, err := service.VerifySignature(credential, result)
		if err != nil {
			t.Fatalf("Failed to verify ECDSA signature: %v", err)
		}

		if !valid {
			t.Error("Expected ECDSA signature to be valid")
		}
	})

	t.Run("SignRSA", func(t *testing.T) {
		result, err := service.SignCredential(credential, SigningMethodRSA)
		if err != nil {
			t.Fatalf("Failed to sign credential with RSA: %v", err)
		}

		if result.Method != SigningMethodRSA {
			t.Errorf("Expected method RSA, got %s", result.Method)
		}

		if result.Signature == "" {
			t.Error("Expected signature to be non-empty")
		}

		if result.KeyID != "test-key-1" {
			t.Errorf("Expected key ID test-key-1, got %s", result.KeyID)
		}

		if result.Algorithm != "RSA-SHA256" {
			t.Errorf("Expected algorithm RSA-SHA256, got %s", result.Algorithm)
		}

		// Verify the signature
		valid, err := service.VerifySignature(credential, result)
		if err != nil {
			t.Fatalf("Failed to verify RSA signature: %v", err)
		}

		if !valid {
			t.Error("Expected RSA signature to be valid")
		}
	})

	t.Run("UnsupportedMethod", func(t *testing.T) {
		_, err := service.SignCredential(credential, "UNSUPPORTED")
		if err == nil {
			t.Error("Expected error for unsupported signing method")
		}
	})

	t.Run("ServiceWithoutKeys", func(t *testing.T) {
		// Create service without keys
		emptyService := NewCredentialSigningService(nil, nil, "test-key-2", "test-issuer")

		// Test JWT signing without RSA key
		_, err := emptyService.SignCredential(credential, SigningMethodJWT)
		if err == nil {
			t.Error("Expected error when signing JWT without RSA key")
		}

		// Test LD-Proof signing without ECDSA key
		_, err = emptyService.SignCredential(credential, SigningMethodLDProof)
		if err == nil {
			t.Error("Expected error when signing LD-Proof without ECDSA key")
		}

		// Test ECDSA signing without ECDSA key
		_, err = emptyService.SignCredential(credential, SigningMethodECDSA)
		if err == nil {
			t.Error("Expected error when signing ECDSA without ECDSA key")
		}

		// Test RSA signing without RSA key
		_, err = emptyService.SignCredential(credential, SigningMethodRSA)
		if err == nil {
			t.Error("Expected error when signing RSA without RSA key")
		}
	})

	t.Run("VerifyInvalidSignature", func(t *testing.T) {
		// Create an invalid signature result
		invalidResult := &SigningResult{
			Method:    SigningMethodJWT,
			Signature: "invalid.signature.here",
			KeyID:     "test-key-1",
			Algorithm: "RS256",
			Created:   time.Now(),
		}

		// Try to verify with invalid signature
		valid, err := service.VerifySignature(credential, invalidResult)
		if err == nil {
			t.Error("Expected error when verifying invalid signature")
		}

		if valid {
			t.Error("Expected invalid signature to be rejected")
		}
	})

	t.Run("SigningResultMetadata", func(t *testing.T) {
		result, err := service.SignCredential(credential, SigningMethodJWT)
		if err != nil {
			t.Fatalf("Failed to sign credential: %v", err)
		}

		// Check that metadata is populated
		if result.Metadata == nil {
			t.Error("Expected metadata to be populated")
		}

		// Check JWT-specific metadata
		if jwtHeader, exists := result.Metadata["jwt_header"]; !exists {
			t.Error("Expected JWT header in metadata")
		} else {
			header, ok := jwtHeader.(map[string]interface{})
			if !ok {
				t.Error("Expected JWT header to be a map")
			}
			if header["kid"] != "test-key-1" {
				t.Errorf("Expected key ID in header, got %v", header["kid"])
			}
		}
	})
}

func TestCredentialSigningService_Integration(t *testing.T) {
	// Generate test keys
	rsaKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("Failed to generate RSA key: %v", err)
	}

	ecdsaKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("Failed to generate ECDSA key: %v", err)
	}

	// Create signing service
	service := NewCredentialSigningService(rsaKey, ecdsaKey, "integration-key", "integration-issuer")

	// Create multiple test credentials
	credentials := []*models.Credential{
		{
			ID:           "cred-1",
			Type:         "StudentCredential",
			Issuer:       "university.edu",
			Subject:      "student-1",
			IssuanceDate: time.Now().Format(time.RFC3339),
			Version:      "1.0",
			Claims: map[string]interface{}{
				"program": "Computer Science",
			},
			Status: "valid",
		},
		{
			ID:           "cred-2",
			Type:         "EmployeeCredential",
			Issuer:       "company.com",
			Subject:      "employee-2",
			IssuanceDate: time.Now().Format(time.RFC3339),
			Version:      "1.0",
			Claims: map[string]interface{}{
				"department": "Engineering",
			},
			Status: "valid",
		},
	}

	methods := []SigningMethod{SigningMethodJWT, SigningMethodLDProof, SigningMethodECDSA, SigningMethodRSA}

	t.Run("MultipleCredentialsMultipleMethods", func(t *testing.T) {
		for i, credential := range credentials {
			for _, method := range methods {
				t.Run(fmt.Sprintf("Credential_%d_Method_%s", i+1, method), func(t *testing.T) {
					// Sign the credential
					result, err := service.SignCredential(credential, method)
					if err != nil {
						t.Fatalf("Failed to sign credential %d with method %s: %v", i+1, method, err)
					}

					// Verify the signature
					valid, err := service.VerifySignature(credential, result)
					if err != nil {
						t.Fatalf("Failed to verify signature for credential %d with method %s: %v", i+1, method, err)
					}

					if !valid {
						t.Errorf("Signature verification failed for credential %d with method %s", i+1, method)
					}

					// Verify result properties
					if result.Method != method {
						t.Errorf("Expected method %s, got %s", method, result.Method)
					}

					if result.Signature == "" {
						t.Error("Expected signature to be non-empty")
					}

					if result.KeyID != "integration-key" {
						t.Errorf("Expected key ID integration-key, got %s", result.KeyID)
					}
				})
			}
		}
	})
}
