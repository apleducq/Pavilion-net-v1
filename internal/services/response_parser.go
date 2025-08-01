package services

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/pavilion-trust/core-broker/internal/config"
	"github.com/pavilion-trust/core-broker/internal/models"
)

// ResponseParserService handles parsing and validation of DP verification responses
type ResponseParserService struct {
	config *config.Config
	// Response validation rules
	validationRules map[string]ValidationRule
	// Response integrity checker
	integrityChecker *ResponseIntegrityChecker
}

// ValidationRule defines validation rules for responses
type ValidationRule struct {
	FieldName     string   `json:"field_name"`
	Required      bool     `json:"required"`
	Type          string   `json:"type"`
	MinValue      float64  `json:"min_value,omitempty"`
	MaxValue      float64  `json:"max_value,omitempty"`
	AllowedValues []string `json:"allowed_values,omitempty"`
	Pattern       string   `json:"pattern,omitempty"`
}

// ResponseIntegrityChecker validates response integrity
type ResponseIntegrityChecker struct {
	// Checksums for response validation
	checksums map[string]string
}

// ParsedResponse represents a parsed and validated response
type ParsedResponse struct {
	JobID           string                 `json:"job_id"`
	Status          string                 `json:"status"`
	Verified        bool                   `json:"verified"`
	Confidence      float64                `json:"confidence"`
	Reason          string                 `json:"reason,omitempty"`
	Evidence        []string               `json:"evidence,omitempty"`
	DPID            string                 `json:"dp_id"`
	Timestamp       string                 `json:"timestamp"`
	ExpirationTime  string                 `json:"expiration_time,omitempty"`
	IntegrityHash   string                 `json:"integrity_hash,omitempty"`
	Metadata        map[string]interface{} `json:"metadata,omitempty"`
	ValidationErrors []string              `json:"validation_errors,omitempty"`
}

// NewResponseParserService creates a new response parser service
func NewResponseParserService(cfg *config.Config) *ResponseParserService {
	service := &ResponseParserService{
		config: cfg,
		validationRules: make(map[string]ValidationRule),
		integrityChecker: &ResponseIntegrityChecker{
			checksums: make(map[string]string),
		},
	}

	// Initialize validation rules
	service.initializeValidationRules()

	return service
}

// initializeValidationRules sets up validation rules for responses
func (s *ResponseParserService) initializeValidationRules() {
	s.validationRules = map[string]ValidationRule{
		"job_id": {
			FieldName: "job_id",
			Required:  true,
			Type:      "string",
			Pattern:   `^job_[a-zA-Z0-9_-]+$`,
		},
		"status": {
			FieldName:     "status",
			Required:      true,
			Type:          "string",
			AllowedValues: []string{"verified", "not_verified", "pending", "failed", "timeout"},
		},
		"verified": {
			FieldName: "verified",
			Required:  true,
			Type:      "boolean",
		},
		"confidence": {
			FieldName: "confidence",
			Required:  true,
			Type:      "float",
			MinValue:  0.0,
			MaxValue:  1.0,
		},
		"dp_id": {
			FieldName: "dp_id",
			Required:  true,
			Type:      "string",
			Pattern:   `^dp_[a-zA-Z0-9_-]+$`,
		},
		"timestamp": {
			FieldName: "timestamp",
			Required:  true,
			Type:      "string",
			Pattern:   `^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}Z$`,
		},
	}
}

// ParseDPResponse parses a DP verification response
func (s *ResponseParserService) ParseDPResponse(dpResp *DPResponse) (*ParsedResponse, error) {
	if dpResp == nil {
		return nil, fmt.Errorf("response cannot be nil")
	}

	// Create parsed response
	parsed := &ParsedResponse{
		JobID:      dpResp.JobID,
		Status:     dpResp.Status,
		Timestamp:  dpResp.Timestamp,
		Metadata:   dpResp.Metadata,
	}

	// Extract verification result
	if dpResp.VerificationResult != nil {
		parsed.Verified = dpResp.VerificationResult.Verified
		parsed.Confidence = dpResp.VerificationResult.Confidence
		parsed.Reason = dpResp.VerificationResult.Reason
		parsed.Evidence = dpResp.VerificationResult.Evidence
	}

	// Set DP ID
	parsed.DPID = "dp-connector"

	// Set expiration time (24 hours from timestamp)
	if parsed.Timestamp != "" {
		if timestamp, err := time.Parse(time.RFC3339, parsed.Timestamp); err == nil {
			expiration := timestamp.Add(24 * time.Hour)
			parsed.ExpirationTime = expiration.Format(time.RFC3339)
		}
	}

	// Validate the parsed response
	validationErrors := s.validateParsedResponse(parsed)
	if len(validationErrors) > 0 {
		parsed.ValidationErrors = validationErrors
	}

	// Generate integrity hash
	parsed.IntegrityHash = s.generateIntegrityHash(parsed)

	return parsed, nil
}

// validateParsedResponse validates a parsed response against rules
func (s *ResponseParserService) validateParsedResponse(parsed *ParsedResponse) []string {
	var errors []string

	// Validate job ID
	if err := s.validateField("job_id", parsed.JobID); err != nil {
		errors = append(errors, fmt.Sprintf("job_id: %s", err))
	}

	// Validate status
	if err := s.validateField("status", parsed.Status); err != nil {
		errors = append(errors, fmt.Sprintf("status: %s", err))
	}

	// Validate confidence
	if err := s.validateField("confidence", parsed.Confidence); err != nil {
		errors = append(errors, fmt.Sprintf("confidence: %s", err))
	}

	// Validate DP ID
	if err := s.validateField("dp_id", parsed.DPID); err != nil {
		errors = append(errors, fmt.Sprintf("dp_id: %s", err))
	}

	// Validate timestamp
	if err := s.validateField("timestamp", parsed.Timestamp); err != nil {
		errors = append(errors, fmt.Sprintf("timestamp: %s", err))
	}

	// Validate verification status consistency
	if parsed.Status == "verified" && !parsed.Verified {
		errors = append(errors, "status inconsistency: status is 'verified' but verified is false")
	}

	if parsed.Status == "not_verified" && parsed.Verified {
		errors = append(errors, "status inconsistency: status is 'not_verified' but verified is true")
	}

	return errors
}

// validateField validates a field against its validation rule
func (s *ResponseParserService) validateField(fieldName string, value interface{}) error {
	rule, exists := s.validationRules[fieldName]
	if !exists {
		return nil // No validation rule for this field
	}

	// Check required fields
	if rule.Required {
		if value == nil || (rule.Type == "string" && value.(string) == "") {
			return fmt.Errorf("field is required")
		}
	}

	// Type-specific validation
	switch rule.Type {
	case "string":
		strValue, ok := value.(string)
		if !ok {
			return fmt.Errorf("expected string, got %T", value)
		}

		// Check pattern if specified
		if rule.Pattern != "" {
			// Simple pattern validation (in production, use regex)
			if !s.matchesPattern(strValue, rule.Pattern) {
				return fmt.Errorf("does not match pattern %s", rule.Pattern)
			}
		}

		// Check allowed values if specified
		if len(rule.AllowedValues) > 0 {
			found := false
			for _, allowed := range rule.AllowedValues {
				if strValue == allowed {
					found = true
					break
				}
			}
			if !found {
				return fmt.Errorf("value '%s' not in allowed values: %v", strValue, rule.AllowedValues)
			}
		}

	case "boolean":
		_, ok := value.(bool)
		if !ok {
			return fmt.Errorf("expected boolean, got %T", value)
		}

	case "float":
		floatValue, ok := value.(float64)
		if !ok {
			return fmt.Errorf("expected float, got %T", value)
		}

		// Check min/max values
		if rule.MinValue != 0 && floatValue < rule.MinValue {
			return fmt.Errorf("value %f is below minimum %f", floatValue, rule.MinValue)
		}

		if rule.MaxValue != 0 && floatValue > rule.MaxValue {
			return fmt.Errorf("value %f is above maximum %f", floatValue, rule.MaxValue)
		}
	}

	return nil
}

// matchesPattern checks if a string matches a pattern
func (s *ResponseParserService) matchesPattern(value, pattern string) bool {
	// Simple pattern validation - in production, use proper regex
	switch pattern {
	case `^job_[a-zA-Z0-9_-]+$`:
		return strings.HasPrefix(value, "job_") && len(value) > 4
	case `^dp_[a-zA-Z0-9_-]+$`:
		return strings.HasPrefix(value, "dp_") && len(value) > 3
	case `^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}Z$`:
		// Simple timestamp validation
		return strings.Contains(value, "T") && strings.HasSuffix(value, "Z")
	default:
		return true
	}
}

// generateIntegrityHash generates a hash for response integrity
func (s *ResponseParserService) generateIntegrityHash(parsed *ParsedResponse) string {
	// Create a hash of the critical response fields
	criticalData := fmt.Sprintf("%s|%s|%t|%.3f|%s|%s",
		parsed.JobID,
		parsed.Status,
		parsed.Verified,
		parsed.Confidence,
		parsed.DPID,
		parsed.Timestamp,
	)

	// In production, use a proper cryptographic hash
	// For now, use a simple hash
	hash := fmt.Sprintf("hash_%d", len(criticalData))
	return hash
}

// ValidateResponseIntegrity validates the integrity of a response
func (s *ResponseParserService) ValidateResponseIntegrity(parsed *ParsedResponse) error {
	if parsed.IntegrityHash == "" {
		return fmt.Errorf("missing integrity hash")
	}

	// Regenerate hash and compare
	expectedHash := s.generateIntegrityHash(parsed)
	if parsed.IntegrityHash != expectedHash {
		return fmt.Errorf("integrity hash mismatch")
	}

	return nil
}

// HandleMalformedResponse handles malformed responses gracefully
func (s *ResponseParserService) HandleMalformedResponse(rawResponse []byte, err error) (*ParsedResponse, error) {
	// Create a minimal parsed response for malformed data
	parsed := &ParsedResponse{
		JobID:      "unknown",
		Status:     "failed",
		Verified:   false,
		Confidence: 0.0,
		DPID:       "unknown",
		Timestamp:  time.Now().Format(time.RFC3339),
		ValidationErrors: []string{
			fmt.Sprintf("malformed response: %v", err),
		},
		Metadata: map[string]interface{}{
			"raw_response_length": len(rawResponse),
			"parse_error":         err.Error(),
		},
	}

	return parsed, nil
}

// ParseAndValidateResponse parses and validates a response in one step
func (s *ResponseParserService) ParseAndValidateResponse(dpResp *DPResponse) (*ParsedResponse, error) {
	// Parse the response
	parsed, err := s.ParseDPResponse(dpResp)
	if err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	// Validate integrity
	if err := s.ValidateResponseIntegrity(parsed); err != nil {
		parsed.ValidationErrors = append(parsed.ValidationErrors, fmt.Sprintf("integrity validation failed: %v", err))
	}

	// Check if response has validation errors
	if len(parsed.ValidationErrors) > 0 {
		return parsed, fmt.Errorf("response validation failed: %v", parsed.ValidationErrors)
	}

	return parsed, nil
}

// ConvertToDPResponse converts a parsed response back to DPResponse format
func (s *ResponseParserService) ConvertToDPResponse(parsed *ParsedResponse) *models.DPResponse {
	return &models.DPResponse{
		Status:         parsed.Status,
		Verified:       parsed.Verified,
		ConfidenceScore: parsed.Confidence,
		Reason:         parsed.Reason,
		DPID:           parsed.DPID,
		Timestamp:      parsed.Timestamp,
	}
}

// GetResponseStats returns response parsing statistics
func (s *ResponseParserService) GetResponseStats() map[string]interface{} {
	return map[string]interface{}{
		"service_status": "active",
		"validation_rules_count": len(s.validationRules),
		"integrity_checking_enabled": true,
		"malformed_response_handling": true,
	}
}

// HealthCheck checks if the response parser service is healthy
func (s *ResponseParserService) HealthCheck(ctx context.Context) error {
	// Test validation rules
	if len(s.validationRules) == 0 {
		return fmt.Errorf("no validation rules configured")
	}

	// Test pattern matching
	testValue := "job_test123"
	if !s.matchesPattern(testValue, `^job_[a-zA-Z0-9_-]+$`) {
		return fmt.Errorf("pattern matching test failed")
	}

	return nil
} 