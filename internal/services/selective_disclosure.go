package services

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// SelectiveDisclosureService provides selective disclosure functionality
type SelectiveDisclosureService struct {
	config *SelectiveDisclosureConfig
}

// SelectiveDisclosureConfig holds configuration for selective disclosure
type SelectiveDisclosureConfig struct {
	MinimalDisclosureEnabled bool
	AuditLoggingEnabled      bool
	HashAlgorithm            string
	Salt                     string
}

// NewSelectiveDisclosureConfig creates a new selective disclosure configuration
func NewSelectiveDisclosureConfig(minimalDisclosure, auditLogging bool, salt string) *SelectiveDisclosureConfig {
	return &SelectiveDisclosureConfig{
		MinimalDisclosureEnabled: minimalDisclosure,
		AuditLoggingEnabled:      auditLogging,
		HashAlgorithm:            "SHA-256",
		Salt:                     salt,
	}
}

// NewSelectiveDisclosureService creates a new selective disclosure service
func NewSelectiveDisclosureService(config *SelectiveDisclosureConfig) *SelectiveDisclosureService {
	return &SelectiveDisclosureService{
		config: config,
	}
}

// Claim represents a claim that can be selectively disclosed
type Claim struct {
	Name       string                 `json:"name"`
	Value      interface{}            `json:"value"`
	Type       string                 `json:"type"` // "exact", "range", "hash", "proof"
	Required   bool                   `json:"required"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
	Disclosure DisclosureLevel        `json:"disclosure"`
}

// DisclosureLevel represents the level of disclosure for a claim
type DisclosureLevel string

const (
	DisclosureLevelFull  DisclosureLevel = "full"
	DisclosureLevelHash  DisclosureLevel = "hash"
	DisclosureLevelRange DisclosureLevel = "range"
	DisclosureLevelProof DisclosureLevel = "proof"
	DisclosureLevelNone  DisclosureLevel = "none"
)

// SelectiveDisclosureRequest represents a request for selective disclosure
type SelectiveDisclosureRequest struct {
	CredentialID string                 `json:"credential_id"`
	Claims       map[string]Claim       `json:"claims"`
	Purpose      string                 `json:"purpose"`
	RequesterID  string                 `json:"requester_id"`
	Expiration   time.Time              `json:"expiration,omitempty"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
}

// SelectiveDisclosureResponse represents the response from selective disclosure
type SelectiveDisclosureResponse struct {
	CredentialID    string                 `json:"credential_id"`
	DisclosedClaims map[string]interface{} `json:"disclosed_claims"`
	HiddenClaims    []string               `json:"hidden_claims"`
	Proofs          map[string]interface{} `json:"proofs,omitempty"`
	AuditLog        *DisclosureAuditLog    `json:"audit_log,omitempty"`
	Metadata        map[string]interface{} `json:"metadata"`
}

// DisclosureAuditLog represents an audit log entry for disclosure
type DisclosureAuditLog struct {
	Timestamp      time.Time              `json:"timestamp"`
	CredentialID   string                 `json:"credential_id"`
	RequesterID    string                 `json:"requester_id"`
	Purpose        string                 `json:"purpose"`
	DisclosedCount int                    `json:"disclosed_count"`
	HiddenCount    int                    `json:"hidden_count"`
	PrivacyHash    string                 `json:"privacy_hash"`
	Metadata       map[string]interface{} `json:"metadata"`
}

// ExtractClaims extracts claims from a credential based on disclosure requirements
func (s *SelectiveDisclosureService) ExtractClaims(credential map[string]interface{}, request SelectiveDisclosureRequest) (*SelectiveDisclosureResponse, error) {
	// Validate request
	if err := s.validateDisclosureRequest(request); err != nil {
		return nil, fmt.Errorf("invalid disclosure request: %w", err)
	}

	disclosedClaims := make(map[string]interface{})
	hiddenClaims := make([]string, 0)
	proofs := make(map[string]interface{})

	// Process each requested claim
	for claimName, claim := range request.Claims {
		if value, exists := credential[claimName]; exists {
			disclosedValue, proof, err := s.processClaim(claimName, value, claim)
			if err != nil {
				return nil, fmt.Errorf("failed to process claim %s: %w", claimName, err)
			}

			if disclosedValue != nil {
				disclosedClaims[claimName] = disclosedValue
			} else {
				hiddenClaims = append(hiddenClaims, claimName)
			}

			// Add proof if it exists (regardless of whether value is disclosed)
			if proof != nil {
				proofs[claimName] = proof
			}
		} else {
			hiddenClaims = append(hiddenClaims, claimName)
		}
	}

	// Create audit log if enabled
	var auditLog *DisclosureAuditLog
	if s.config.AuditLoggingEnabled {
		auditLog = s.createAuditLog(request, disclosedClaims, hiddenClaims)
	}

	// Create privacy hash
	privacyHash := s.generatePrivacyHash(request, disclosedClaims)

	response := &SelectiveDisclosureResponse{
		CredentialID:    request.CredentialID,
		DisclosedClaims: disclosedClaims,
		HiddenClaims:    hiddenClaims,
		Proofs:          proofs,
		AuditLog:        auditLog,
		Metadata: map[string]interface{}{
			"privacy_hash": privacyHash,
			"timestamp":    time.Now().Format(time.RFC3339),
			"purpose":      request.Purpose,
			"requester_id": request.RequesterID,
		},
	}

	return response, nil
}

// processClaim processes a single claim based on its disclosure level
func (s *SelectiveDisclosureService) processClaim(claimName string, value interface{}, claim Claim) (interface{}, interface{}, error) {
	switch claim.Disclosure {
	case DisclosureLevelFull:
		return value, nil, nil

	case DisclosureLevelHash:
		hash, err := s.hashValue(claimName, value)
		if err != nil {
			return nil, nil, err
		}
		return hash, nil, nil

	case DisclosureLevelRange:
		rangeValue, err := s.createRangeValue(claimName, value, claim)
		if err != nil {
			return nil, nil, err
		}
		return rangeValue, nil, nil

	case DisclosureLevelProof:
		proof, err := s.createProof(claimName, value, claim)
		if err != nil {
			return nil, nil, err
		}
		// For proof disclosure, we return nil for the value (hidden) but return the proof
		return nil, proof, nil

	case DisclosureLevelNone:
		return nil, nil, nil

	default:
		return nil, nil, fmt.Errorf("unknown disclosure level: %s", claim.Disclosure)
	}
}

// hashValue creates a hash of a value
func (s *SelectiveDisclosureService) hashValue(claimName string, value interface{}) (string, error) {
	// Convert value to string
	valueStr := fmt.Sprintf("%v", value)

	// Create data to hash: claim name + value + salt
	dataToHash := fmt.Sprintf("%s:%s:%s", claimName, valueStr, s.config.Salt)

	// Hash using SHA-256
	hash := sha256.Sum256([]byte(dataToHash))

	return hex.EncodeToString(hash[:]), nil
}

// createRangeValue creates a range value for numeric claims
func (s *SelectiveDisclosureService) createRangeValue(claimName string, value interface{}, claim Claim) (interface{}, error) {
	// For MVP, we'll create simple ranges
	// In production, you would implement more sophisticated range logic

	switch v := value.(type) {
	case int:
		// Create age range (e.g., 25 -> "18-30")
		if strings.Contains(strings.ToLower(claimName), "age") {
			return s.createAgeRange(v), nil
		}
		// Create general range
		return s.createNumericRange(v), nil

	case float64:
		// Create general range for float values
		return s.createFloatRange(v), nil

	default:
		// For non-numeric values, return a hash
		return s.hashValue(claimName, value)
	}
}

// createAgeRange creates an age range
func (s *SelectiveDisclosureService) createAgeRange(age int) string {
	switch {
	case age < 18:
		return "under-18"
	case age >= 18 && age < 30:
		return "18-30"
	case age >= 30 && age < 50:
		return "30-50"
	case age >= 50 && age < 65:
		return "50-65"
	default:
		return "65-plus"
	}
}

// createNumericRange creates a numeric range
func (s *SelectiveDisclosureService) createNumericRange(value int) string {
	// Create ranges of 10
	rangeStart := (value / 10) * 10
	rangeEnd := rangeStart + 9
	return fmt.Sprintf("%d-%d", rangeStart, rangeEnd)
}

// createFloatRange creates a float range
func (s *SelectiveDisclosureService) createFloatRange(value float64) string {
	// Create ranges of 1.0
	rangeStart := int(value)
	rangeEnd := rangeStart + 1
	return fmt.Sprintf("%d-%d", rangeStart, rangeEnd)
}

// createProof creates a zero-knowledge proof for a claim
func (s *SelectiveDisclosureService) createProof(claimName string, value interface{}, claim Claim) (interface{}, error) {
	// For MVP, we'll create a simple proof structure
	// In production, you would implement actual zero-knowledge proofs

	proof := map[string]interface{}{
		"type":       "simple_proof",
		"claim_name": claimName,
		"proof_hash": s.generateProofHash(claimName, value),
		"timestamp":  time.Now().Format(time.RFC3339),
		"algorithm":  "SHA-256",
		"metadata":   claim.Metadata,
	}

	return proof, nil
}

// generateProofHash generates a hash for a proof
func (s *SelectiveDisclosureService) generateProofHash(claimName string, value interface{}) string {
	valueStr := fmt.Sprintf("%v", value)
	dataToHash := fmt.Sprintf("proof:%s:%s:%s", claimName, valueStr, s.config.Salt)
	hash := sha256.Sum256([]byte(dataToHash))
	return hex.EncodeToString(hash[:])
}

// generatePrivacyHash generates a privacy hash for the disclosure
func (s *SelectiveDisclosureService) generatePrivacyHash(request SelectiveDisclosureRequest, disclosedClaims map[string]interface{}) string {
	// Create a hash of the disclosure request and results
	data := map[string]interface{}{
		"credential_id":    request.CredentialID,
		"purpose":          request.Purpose,
		"requester_id":     request.RequesterID,
		"disclosed_claims": disclosedClaims,
		"timestamp":        time.Now().Format(time.RFC3339),
	}

	dataBytes, _ := json.Marshal(data)
	dataToHash := fmt.Sprintf("%s:%s", string(dataBytes), s.config.Salt)
	hash := sha256.Sum256([]byte(dataToHash))
	return hex.EncodeToString(hash[:])
}

// createAuditLog creates an audit log entry
func (s *SelectiveDisclosureService) createAuditLog(request SelectiveDisclosureRequest, disclosedClaims map[string]interface{}, hiddenClaims []string) *DisclosureAuditLog {
	return &DisclosureAuditLog{
		Timestamp:      time.Now(),
		CredentialID:   request.CredentialID,
		RequesterID:    request.RequesterID,
		Purpose:        request.Purpose,
		DisclosedCount: len(disclosedClaims),
		HiddenCount:    len(hiddenClaims),
		PrivacyHash:    s.generatePrivacyHash(request, disclosedClaims),
		Metadata:       request.Metadata,
	}
}

// validateDisclosureRequest validates a disclosure request
func (s *SelectiveDisclosureService) validateDisclosureRequest(request SelectiveDisclosureRequest) error {
	if request.CredentialID == "" {
		return fmt.Errorf("credential ID is required")
	}

	if len(request.Claims) == 0 {
		return fmt.Errorf("at least one claim is required")
	}

	if request.Purpose == "" {
		return fmt.Errorf("purpose is required")
	}

	if request.RequesterID == "" {
		return fmt.Errorf("requester ID is required")
	}

	// Validate each claim
	for claimName, claim := range request.Claims {
		if claimName == "" {
			return fmt.Errorf("claim name cannot be empty")
		}

		if claim.Disclosure == "" {
			return fmt.Errorf("disclosure level is required for claim %s", claimName)
		}

		// Validate disclosure level
		validLevels := []DisclosureLevel{
			DisclosureLevelFull,
			DisclosureLevelHash,
			DisclosureLevelRange,
			DisclosureLevelProof,
			DisclosureLevelNone,
		}

		valid := false
		for _, level := range validLevels {
			if claim.Disclosure == level {
				valid = true
				break
			}
		}

		if !valid {
			return fmt.Errorf("invalid disclosure level %s for claim %s", claim.Disclosure, claimName)
		}
	}

	return nil
}

// ValidateDisclosurePrivacy validates privacy guarantees of a disclosure
func (s *SelectiveDisclosureService) ValidateDisclosurePrivacy(response *SelectiveDisclosureResponse) error {
	// Check if sensitive information is properly hidden
	for _, hiddenClaim := range response.HiddenClaims {
		if _, exists := response.DisclosedClaims[hiddenClaim]; exists {
			return fmt.Errorf("claim %s is marked as hidden but is disclosed", hiddenClaim)
		}
	}

	// Check if required claims are disclosed
	// This would be based on the original request requirements

	// Validate privacy hash
	if privacyHash, exists := response.Metadata["privacy_hash"]; !exists || privacyHash == "" {
		return fmt.Errorf("privacy hash is missing")
	}

	return nil
}

// GetDisclosureStats returns statistics about disclosure operations
func (s *SelectiveDisclosureService) GetDisclosureStats() map[string]interface{} {
	return map[string]interface{}{
		"minimal_disclosure_enabled": s.config.MinimalDisclosureEnabled,
		"audit_logging_enabled":      s.config.AuditLoggingEnabled,
		"hash_algorithm":             s.config.HashAlgorithm,
		"supported_disclosure_levels": []string{
			string(DisclosureLevelFull),
			string(DisclosureLevelHash),
			string(DisclosureLevelRange),
			string(DisclosureLevelProof),
			string(DisclosureLevelNone),
		},
	}
}
