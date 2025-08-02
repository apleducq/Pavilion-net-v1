package services

import (
	"testing"
	"time"

	"github.com/pavilion-trust/core-broker/internal/models"
)

func TestNewCredentialFromTemplate(t *testing.T) {
	template := CredentialTemplate{
		Type:    "StudentCredential",
		Issuer:  "university.edu",
		Version: "1.0",
		Claims: map[string]interface{}{
			"status": "enrolled",
		},
		Metadata: map[string]interface{}{
			"template": "default-student",
		},
	}

	claims := map[string]interface{}{
		"program": "Computer Science",
	}
	metadata := map[string]interface{}{
		"issued_by": "Registrar",
	}
	proof := models.CredentialProof{
		Type:               "JWT",
		Created:            time.Now().Format(time.RFC3339),
		VerificationMethod: "https://university.edu/keys/1",
		ProofPurpose:       "assertionMethod",
		JWS:                "header.payload.signature",
	}

	cred := NewCredentialFromTemplate(template, "student-123", "valid", proof, claims, metadata)

	if cred.Type != "StudentCredential" {
		t.Errorf("Expected type StudentCredential, got %s", cred.Type)
	}
	if cred.Issuer != "university.edu" {
		t.Errorf("Expected issuer university.edu, got %s", cred.Issuer)
	}
	if cred.Version != "1.0" {
		t.Errorf("Expected version 1.0, got %s", cred.Version)
	}
	if cred.Claims["status"] != "enrolled" {
		t.Errorf("Expected claim status=enrolled, got %v", cred.Claims["status"])
	}
	if cred.Claims["program"] != "Computer Science" {
		t.Errorf("Expected claim program=Computer Science, got %v", cred.Claims["program"])
	}
	if cred.Metadata["template"] != "default-student" {
		t.Errorf("Expected metadata template=default-student, got %v", cred.Metadata["template"])
	}
	if cred.Metadata["issued_by"] != "Registrar" {
		t.Errorf("Expected metadata issued_by=Registrar, got %v", cred.Metadata["issued_by"])
	}
	if cred.Status != "valid" {
		t.Errorf("Expected status valid, got %s", cred.Status)
	}
	if cred.Subject != "student-123" {
		t.Errorf("Expected subject student-123, got %s", cred.Subject)
	}
	if cred.Proof.Type != "JWT" {
		t.Errorf("Expected proof type JWT, got %s", cred.Proof.Type)
	}
}
