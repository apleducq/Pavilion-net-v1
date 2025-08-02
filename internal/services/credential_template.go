package services

import (
	"time"

	"github.com/pavilion-trust/core-broker/internal/models"
)

// CredentialTemplate defines a reusable template for credentials
// Fields can be pre-filled; others are set at instantiation time.
type CredentialTemplate struct {
	Type     string
	Issuer   string
	Version  string
	Claims   map[string]interface{}
	Metadata map[string]interface{}
}

// NewCredentialFromTemplate instantiates a Credential from a template
func NewCredentialFromTemplate(template CredentialTemplate, subject, status string, proof models.CredentialProof, claims map[string]interface{}, metadata map[string]interface{}) *models.Credential {
	return &models.Credential{
		ID:           generateCredentialID(),
		Type:         template.Type,
		Issuer:       template.Issuer,
		Subject:      subject,
		IssuanceDate: time.Now().Format(time.RFC3339),
		Version:      template.Version,
		Claims:       mergeClaims(template.Claims, claims),
		Proof:        proof,
		Status:       status,
		Metadata:     mergeMetadata(template.Metadata, metadata),
	}
}

// mergeClaims merges template and instance claims
func mergeClaims(template, instance map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{})
	for k, v := range template {
		result[k] = v
	}
	for k, v := range instance {
		result[k] = v
	}
	return result
}

// mergeMetadata merges template and instance metadata
func mergeMetadata(template, instance map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{})
	for k, v := range template {
		result[k] = v
	}
	for k, v := range instance {
		result[k] = v
	}
	return result
}

// generateCredentialID generates a unique credential ID (for MVP, use timestamp)
func generateCredentialID() string {
	return "cred-" + time.Now().Format("20060102T150405.000")
}
