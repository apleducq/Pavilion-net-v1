package services

import (
	"context"
	"fmt"
	"time"

	"github.com/pavilion-trust/core-broker/internal/config"
	"github.com/pavilion-trust/core-broker/internal/models"
)

// ResponseFormatterService handles formatting of verification responses
type ResponseFormatterService struct {
	config *config.Config
	// Response validation
	validator *ResponseValidator
	// Response templates
	templates map[string]*ResponseTemplate
}

// ResponseValidator validates formatted responses
type ResponseValidator struct {
	// Validation rules for formatted responses
	rules map[string]ValidationRule
}

// ResponseTemplate defines response formatting templates
type ResponseTemplate struct {
	TemplateName string                 `json:"template_name"`
	Fields       map[string]FieldSpec   `json:"fields"`
	Required     []string               `json:"required"`
	Optional     []string               `json:"optional"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
}

// FieldSpec defines a field specification
type FieldSpec struct {
	Type        string      `json:"type"`
	Required    bool        `json:"required"`
	Default     interface{} `json:"default,omitempty"`
	Description string      `json:"description"`
	Example     interface{} `json:"example,omitempty"`
}

// FormattedResponse represents a formatted verification response
type FormattedResponse struct {
	RequestID       string                 `json:"request_id"`
	Status          string                 `json:"status"`
	Verified        bool                   `json:"verified"`
	Confidence      float64                `json:"confidence"`
	Reason          string                 `json:"reason,omitempty"`
	Evidence        []string               `json:"evidence,omitempty"`
	DPID            string                 `json:"dp_id"`
	Timestamp       string                 `json:"timestamp"`
	ExpirationTime  string                 `json:"expiration_time,omitempty"`
	ProcessingTime  string                 `json:"processing_time,omitempty"`
	RequestHash     string                 `json:"request_hash,omitempty"`
	ResponseHash    string                 `json:"response_hash,omitempty"`
	Metadata        map[string]interface{} `json:"metadata,omitempty"`
	ValidationErrors []string              `json:"validation_errors,omitempty"`
}

// NewResponseFormatterService creates a new response formatter service
func NewResponseFormatterService(cfg *config.Config) *ResponseFormatterService {
	service := &ResponseFormatterService{
		config: cfg,
		validator: &ResponseValidator{
			rules: make(map[string]ValidationRule),
		},
		templates: make(map[string]*ResponseTemplate),
	}

	// Initialize response templates
	service.initializeTemplates()

	return service
}

// initializeTemplates sets up response formatting templates
func (s *ResponseFormatterService) initializeTemplates() {
	// Standard verification response template
	s.templates["verification"] = &ResponseTemplate{
		TemplateName: "verification",
		Fields: map[string]FieldSpec{
			"request_id": {
				Type:        "string",
				Required:    true,
				Description: "Unique request identifier",
				Example:     "req_123456789",
			},
			"status": {
				Type:        "string",
				Required:    true,
				Description: "Verification status",
				Example:     "verified",
			},
			"verified": {
				Type:        "boolean",
				Required:    true,
				Description: "Whether the claim was verified",
				Example:     true,
			},
			"confidence": {
				Type:        "float",
				Required:    true,
				Description: "Confidence score (0.0 to 1.0)",
				Example:     0.95,
			},
			"reason": {
				Type:        "string",
				Required:    false,
				Description: "Reason for verification result",
				Example:     "Student enrollment confirmed",
			},
			"evidence": {
				Type:        "array",
				Required:    false,
				Description: "Supporting evidence",
				Example:     []string{"enrollment_record", "academic_standing"},
			},
			"dp_id": {
				Type:        "string",
				Required:    true,
				Description: "Data Provider identifier",
				Example:     "dp_university_001",
			},
			"timestamp": {
				Type:        "string",
				Required:    true,
				Description: "Response timestamp (RFC3339)",
				Example:     "2023-01-01T12:00:00Z",
			},
			"expiration_time": {
				Type:        "string",
				Required:    false,
				Description: "Response expiration time (RFC3339)",
				Example:     "2023-01-02T12:00:00Z",
			},
			"processing_time": {
				Type:        "string",
				Required:    false,
				Description: "Processing time duration",
				Example:     "1.5s",
			},
		},
		Required: []string{"request_id", "status", "verified", "confidence", "dp_id", "timestamp"},
		Optional: []string{"reason", "evidence", "expiration_time", "processing_time", "request_hash", "response_hash", "metadata"},
		Metadata: map[string]interface{}{
			"version": "1.0",
			"format":  "json",
		},
	}
}

// FormatResponse formats a verification response according to API spec
func (s *ResponseFormatterService) FormatResponse(
	ctx context.Context,
	parsedResp *ParsedResponse,
	requestID string,
	processingTime time.Duration,
	requestHash string,
) (*FormattedResponse, error) {
	// Create formatted response
	formatted := &FormattedResponse{
		RequestID:      requestID,
		Status:         parsedResp.Status,
		Verified:       parsedResp.Verified,
		Confidence:     parsedResp.Confidence,
		Reason:         parsedResp.Reason,
		Evidence:       parsedResp.Evidence,
		DPID:           parsedResp.DPID,
		Timestamp:      parsedResp.Timestamp,
		ExpirationTime: parsedResp.ExpirationTime,
		ProcessingTime: processingTime.String(),
		RequestHash:    requestHash,
		ResponseHash:   parsedResp.IntegrityHash,
		Metadata:       parsedResp.Metadata,
	}

	// Add validation errors if any
	if len(parsedResp.ValidationErrors) > 0 {
		formatted.ValidationErrors = parsedResp.ValidationErrors
	}

	// Validate the formatted response
	if err := s.validator.ValidateFormattedResponse(formatted); err != nil {
		return nil, fmt.Errorf("response validation failed: %w", err)
	}

	return formatted, nil
}

// FormatErrorResponse formats an error response
func (s *ResponseFormatterService) FormatErrorResponse(
	ctx context.Context,
	requestID string,
	errorCode string,
	errorMessage string,
	processingTime time.Duration,
) *FormattedResponse {
	return &FormattedResponse{
		RequestID:      requestID,
		Status:         "error",
		Verified:       false,
		Confidence:     0.0,
		Reason:         errorMessage,
		Timestamp:      time.Now().Format(time.RFC3339),
		ProcessingTime: processingTime.String(),
		Metadata: map[string]interface{}{
			"error_code": errorCode,
		},
		ValidationErrors: []string{errorMessage},
	}
}

// ValidateFormattedResponse validates a formatted response
func (rv *ResponseValidator) ValidateFormattedResponse(response *FormattedResponse) error {
	var errors []string

	// Validate required fields
	if response.RequestID == "" {
		errors = append(errors, "request_id is required")
	}

	if response.Status == "" {
		errors = append(errors, "status is required")
	}

	if response.DPID == "" {
		errors = append(errors, "dp_id is required")
	}

	if response.Timestamp == "" {
		errors = append(errors, "timestamp is required")
	}

	// Validate confidence score
	if response.Confidence < 0.0 || response.Confidence > 1.0 {
		errors = append(errors, "confidence must be between 0.0 and 1.0")
	}

	// Validate status consistency
	if response.Status == "verified" && !response.Verified {
		errors = append(errors, "status inconsistency: status is 'verified' but verified is false")
	}

	if response.Status == "not_verified" && response.Verified {
		errors = append(errors, "status inconsistency: status is 'not_verified' but verified is true")
	}

	// Validate timestamp format
	if response.Timestamp != "" {
		if _, err := time.Parse(time.RFC3339, response.Timestamp); err != nil {
			errors = append(errors, "timestamp must be in RFC3339 format")
		}
	}

	// Validate expiration time if present
	if response.ExpirationTime != "" {
		if _, err := time.Parse(time.RFC3339, response.ExpirationTime); err != nil {
			errors = append(errors, "expiration_time must be in RFC3339 format")
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("validation errors: %v", errors)
	}

	return nil
}

// ConvertToVerificationResponse converts a formatted response to models.VerificationResponse
func (s *ResponseFormatterService) ConvertToVerificationResponse(formatted *FormattedResponse) *models.VerificationResponse {
	return &models.VerificationResponse{
		RequestID:       formatted.RequestID,
		Status:          formatted.Status,
		Verified:        formatted.Verified,
		ConfidenceScore: formatted.Confidence,
		Reason:          formatted.Reason,
		Evidence:        formatted.Evidence,
		DPID:            formatted.DPID,
		Timestamp:       formatted.Timestamp,
		ExpiresAt:       formatted.ExpirationTime,
		ProcessingTime:  formatted.ProcessingTime,
		RequestHash:     formatted.RequestHash,
		ResponseHash:    formatted.ResponseHash,
		Metadata:        formatted.Metadata,
		ValidationErrors: formatted.ValidationErrors,
	}
}

// GenerateResponseHash generates a hash for the response
func (s *ResponseFormatterService) GenerateResponseHash(response *FormattedResponse) string {
	// Create a hash of the critical response fields
	criticalData := fmt.Sprintf("%s|%s|%t|%.3f|%s|%s",
		response.RequestID,
		response.Status,
		response.Verified,
		response.Confidence,
		response.DPID,
		response.Timestamp,
	)

	// In production, use a proper cryptographic hash
	// For now, use a simple hash
	hash := fmt.Sprintf("resp_hash_%d", len(criticalData))
	return hash
}

// ValidateResponseIntegrity validates the integrity of a formatted response
func (s *ResponseFormatterService) ValidateResponseIntegrity(response *FormattedResponse) error {
	if response.ResponseHash == "" {
		return fmt.Errorf("missing response hash")
	}

	// Regenerate hash and compare
	expectedHash := s.GenerateResponseHash(response)
	if response.ResponseHash != expectedHash {
		return fmt.Errorf("response hash mismatch")
	}

	return nil
}

// GetResponseTemplate returns a response template by name
func (s *ResponseFormatterService) GetResponseTemplate(templateName string) (*ResponseTemplate, error) {
	template, exists := s.templates[templateName]
	if !exists {
		return nil, fmt.Errorf("template not found: %s", templateName)
	}
	return template, nil
}

// ListTemplates lists all available response templates
func (s *ResponseFormatterService) ListTemplates() []string {
	templates := make([]string, 0, len(s.templates))
	for name := range s.templates {
		templates = append(templates, name)
	}
	return templates
}

// GetFormattedResponseStats returns response formatting statistics
func (s *ResponseFormatterService) GetFormattedResponseStats() map[string]interface{} {
	return map[string]interface{}{
		"service_status": "active",
		"templates_count": len(s.templates),
		"available_templates": s.ListTemplates(),
		"validation_enabled": true,
		"integrity_checking_enabled": true,
	}
}

// HealthCheck checks if the response formatter service is healthy
func (s *ResponseFormatterService) HealthCheck(ctx context.Context) error {
	// Check templates
	if len(s.templates) == 0 {
		return fmt.Errorf("no response templates configured")
	}

	// Test template retrieval
	_, err := s.GetResponseTemplate("verification")
	if err != nil {
		return fmt.Errorf("verification template not found: %w", err)
	}

	// Test response formatting
	testResponse := &FormattedResponse{
		RequestID:  "test_req_123",
		Status:     "verified",
		Verified:   true,
		Confidence: 0.95,
		DPID:       "dp_test",
		Timestamp:  time.Now().Format(time.RFC3339),
	}

	if err := s.validator.ValidateFormattedResponse(testResponse); err != nil {
		return fmt.Errorf("response validation test failed: %w", err)
	}

	return nil
} 