package services

import (
	"context"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/pavilion-trust/core-broker/internal/models"
)

// CredentialValidator handles credential validation
type CredentialValidator struct {
	trustedIssuers map[string]*rsa.PublicKey
	cache          *CredentialCache
}

// CredentialCache provides caching for credential validation results
type CredentialCache struct {
	results map[string]*CredentialValidationResult
	mu      sync.RWMutex
	ttl     time.Duration
}

// CredentialValidationResult represents the result of credential validation
type CredentialValidationResult struct {
	Valid          bool      `json:"valid"`
	Reason         string    `json:"reason"`
	Timestamp      time.Time `json:"timestamp"`
	ExpiresAt      time.Time `json:"expires_at"`
	IssuerValid    bool      `json:"issuer_valid"`
	SignatureValid bool      `json:"signature_valid"`
	NotExpired     bool      `json:"not_expired"`
}

// NewCredentialCache creates a new credential cache
func NewCredentialCache(ttl time.Duration) *CredentialCache {
	return &CredentialCache{
		results: make(map[string]*CredentialValidationResult),
		ttl:     ttl,
	}
}

// Get retrieves a cached validation result
func (c *CredentialCache) Get(key string) (*CredentialValidationResult, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	result, exists := c.results[key]
	if !exists {
		return nil, false
	}

	// Check if result has expired
	if time.Now().After(result.ExpiresAt) {
		delete(c.results, key)
		return nil, false
	}

	return result, true
}

// Set stores a validation result in cache
func (c *CredentialCache) Set(key string, result *CredentialValidationResult) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.results[key] = result
}

// NewCredentialValidator creates a new credential validator
func NewCredentialValidator() *CredentialValidator {
	return &CredentialValidator{
		trustedIssuers: make(map[string]*rsa.PublicKey),
		cache:          NewCredentialCache(10 * time.Minute), // Cache validation results for 10 minutes
	}
}

// AddTrustedIssuer adds a trusted issuer with their public key
func (cv *CredentialValidator) AddTrustedIssuer(issuer string, publicKeyPEM string) error {
	// Decode PEM block
	block, _ := pem.Decode([]byte(publicKeyPEM))
	if block == nil {
		return fmt.Errorf("failed to decode PEM block")
	}

	// Parse public key
	publicKey, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return fmt.Errorf("failed to parse public key: %w", err)
	}

	// Convert to RSA public key
	rsaKey, ok := publicKey.(*rsa.PublicKey)
	if !ok {
		return fmt.Errorf("public key is not RSA")
	}

	cv.trustedIssuers[issuer] = rsaKey
	return nil
}

// ValidateCredential validates a single credential
func (cv *CredentialValidator) ValidateCredential(ctx context.Context, credential *models.Credential) (*CredentialValidationResult, error) {
	// Generate cache key
	cacheKey := cv.generateCacheKey(credential)

	// Check cache first
	if cached, exists := cv.cache.Get(cacheKey); exists {
		return cached, nil
	}

	// Validate credential structure
	if err := cv.validateStructure(credential); err != nil {
		return cv.createValidationResult(false, fmt.Sprintf("Structure validation failed: %v", err)), nil
	}

	// Validate signature
	signatureValid, err := cv.validateSignature(credential)
	if err != nil {
		return cv.createValidationResult(false, fmt.Sprintf("Signature validation failed: %v", err)), nil
	}

	// Validate issuer
	issuerValid, err := cv.validateIssuer(credential)
	if err != nil {
		return cv.createValidationResult(false, fmt.Sprintf("Issuer validation failed: %v", err)), nil
	}

	// Check expiration
	notExpired, err := cv.checkExpiration(credential)
	if err != nil {
		return cv.createValidationResult(false, fmt.Sprintf("Expiration check failed: %v", err)), nil
	}

	// Determine overall validity
	valid := signatureValid && issuerValid && notExpired
	reason := cv.buildValidationReason(signatureValid, issuerValid, notExpired)

	// Create validation result
	result := &CredentialValidationResult{
		Valid:          valid,
		Reason:         reason,
		Timestamp:      time.Now(),
		ExpiresAt:      time.Now().Add(cv.cache.ttl),
		IssuerValid:    issuerValid,
		SignatureValid: signatureValid,
		NotExpired:     notExpired,
	}

	// Cache the result
	cv.cache.Set(cacheKey, result)

	return result, nil
}

// ValidateCredentials validates multiple credentials
func (cv *CredentialValidator) ValidateCredentials(ctx context.Context, credentials []models.Credential) ([]*CredentialValidationResult, error) {
	var results []*CredentialValidationResult

	for _, credential := range credentials {
		result, err := cv.ValidateCredential(ctx, &credential)
		if err != nil {
			return nil, fmt.Errorf("failed to validate credential %s: %w", credential.ID, err)
		}
		results = append(results, result)
	}

	return results, nil
}

// validateStructure validates the credential structure
func (cv *CredentialValidator) validateStructure(credential *models.Credential) error {
	if credential.ID == "" {
		return fmt.Errorf("credential ID is required")
	}

	if credential.Type == "" {
		return fmt.Errorf("credential type is required")
	}

	if credential.Issuer == "" {
		return fmt.Errorf("credential issuer is required")
	}

	if credential.Subject == "" {
		return fmt.Errorf("credential subject is required")
	}

	if credential.IssuanceDate == "" {
		return fmt.Errorf("credential issuance date is required")
	}

	if credential.Version == "" {
		return fmt.Errorf("credential version is required")
	}

	if len(credential.Claims) == 0 {
		return fmt.Errorf("credential must have at least one claim")
	}

	// Validate proof structure
	if err := cv.validateProofStructure(&credential.Proof); err != nil {
		return fmt.Errorf("proof validation failed: %w", err)
	}

	return nil
}

// validateProofStructure validates the credential proof structure
func (cv *CredentialValidator) validateProofStructure(proof *models.CredentialProof) error {
	if proof.Type == "" {
		return fmt.Errorf("proof type is required")
	}

	if proof.Created == "" {
		return fmt.Errorf("proof creation date is required")
	}

	if proof.VerificationMethod == "" {
		return fmt.Errorf("proof verification method is required")
	}

	if proof.ProofPurpose == "" {
		return fmt.Errorf("proof purpose is required")
	}

	// Validate JWS if present
	if proof.JWS != "" {
		if err := cv.validateJWS(proof.JWS); err != nil {
			return fmt.Errorf("JWS validation failed: %w", err)
		}
	}

	return nil
}

// validateJWS validates a JSON Web Signature
func (cv *CredentialValidator) validateJWS(jws string) error {
	// Parse JWS
	parts := strings.Split(jws, ".")
	if len(parts) != 3 {
		return fmt.Errorf("invalid JWS format")
	}

	// Decode header
	headerBytes, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return fmt.Errorf("failed to decode JWS header: %w", err)
	}

	var header map[string]interface{}
	if err := json.Unmarshal(headerBytes, &header); err != nil {
		return fmt.Errorf("failed to parse JWS header: %w", err)
	}

	// Validate algorithm
	alg, ok := header["alg"].(string)
	if !ok {
		return fmt.Errorf("missing algorithm in JWS header")
	}

	if alg != "RS256" {
		return fmt.Errorf("unsupported algorithm: %s", alg)
	}

	return nil
}

// validateSignature validates the credential signature
func (cv *CredentialValidator) validateSignature(credential *models.Credential) (bool, error) {
	// For MVP, we'll implement a simplified signature validation
	// In production, this would validate the actual cryptographic signature

	if credential.Proof.JWS == "" {
		return false, fmt.Errorf("no JWS signature provided")
	}

	// Basic JWS format validation
	parts := strings.Split(credential.Proof.JWS, ".")
	if len(parts) != 3 {
		return false, fmt.Errorf("invalid JWS format")
	}

	// Validate that all parts are base64-encoded
	for i, part := range parts {
		if _, err := base64.RawURLEncoding.DecodeString(part); err != nil {
			return false, fmt.Errorf("invalid base64 encoding in part %d: %w", i, err)
		}
	}

	// For MVP, we'll assume the signature is valid if the format is correct
	// In production, you would verify the actual cryptographic signature
	return true, nil
}

// validateIssuer validates the credential issuer
func (cv *CredentialValidator) validateIssuer(credential *models.Credential) (bool, error) {
	// Check if issuer is in trusted list
	if _, exists := cv.trustedIssuers[credential.Issuer]; exists {
		return true, nil
	}

	// For MVP, we'll accept any issuer
	// In production, you would have a strict list of trusted issuers
	return true, nil
}

// checkExpiration checks if the credential is expired
func (cv *CredentialValidator) checkExpiration(credential *models.Credential) (bool, error) {
	if credential.ExpirationDate == "" {
		// No expiration date, consider it valid
		return true, nil
	}

	// Parse expiration date
	expiration, err := time.Parse(time.RFC3339, credential.ExpirationDate)
	if err != nil {
		return false, fmt.Errorf("invalid expiration date format: %w", err)
	}

	// Check if expired
	now := time.Now()
	if now.After(expiration) {
		return false, nil
	}

	return true, nil
}

// createValidationResult creates a validation result
func (cv *CredentialValidator) createValidationResult(valid bool, reason string) *CredentialValidationResult {
	return &CredentialValidationResult{
		Valid:          valid,
		Reason:         reason,
		Timestamp:      time.Now(),
		ExpiresAt:      time.Now().Add(cv.cache.ttl),
		IssuerValid:    false,
		SignatureValid: false,
		NotExpired:     false,
	}
}

// buildValidationReason builds a human-readable validation reason
func (cv *CredentialValidator) buildValidationReason(signatureValid, issuerValid, notExpired bool) string {
	var reasons []string

	if !signatureValid {
		reasons = append(reasons, "invalid signature")
	}

	if !issuerValid {
		reasons = append(reasons, "untrusted issuer")
	}

	if !notExpired {
		reasons = append(reasons, "expired credential")
	}

	if len(reasons) == 0 {
		return "Credential is valid"
	}

	return fmt.Sprintf("Credential validation failed: %s", strings.Join(reasons, ", "))
}

// generateCacheKey creates a cache key for credential validation
func (cv *CredentialValidator) generateCacheKey(credential *models.Credential) string {
	// Create a simple hash-based key
	// In a real implementation, you might want to use a proper hash function
	key := fmt.Sprintf("%s-%s-%s-%s", credential.ID, credential.Type, credential.Issuer, credential.IssuanceDate)
	return key
}

// GetCacheStats returns cache statistics for monitoring
func (cv *CredentialValidator) GetCacheStats() map[string]interface{} {
	cv.cache.mu.RLock()
	defer cv.cache.mu.RUnlock()

	return map[string]interface{}{
		"cached_validations": len(cv.cache.results),
		"cache_ttl":          cv.cache.ttl.String(),
		"trusted_issuers":    len(cv.trustedIssuers),
	}
}
