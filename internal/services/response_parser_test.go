package services

import (
	"context"
	"testing"
	"time"

	"github.com/pavilion-trust/core-broker/internal/config"
)

func TestNewResponseParserService(t *testing.T) {
	cfg := &config.Config{
		ResponseValidationEnabled: true,
		IntegrityCheckEnabled:     true,
	}

	service := NewResponseParserService(cfg)

	if service == nil {
		t.Fatal("Expected service to be created")
	}

	if service.config != cfg {
		t.Error("Expected config to be set")
	}

	if service.validationRules == nil {
		t.Error("Expected validation rules to be initialized")
	}

	if service.integrityChecker == nil {
		t.Error("Expected integrity checker to be created")
	}
}

func TestResponseParserService_ParseDPResponse_Success(t *testing.T) {
	cfg := &config.Config{
		ResponseValidationEnabled: true,
		IntegrityCheckEnabled:     true,
	}

	service := NewResponseParserService(cfg)

	dpResponse := &DPResponse{
		JobID:  "job_123456",
		Status: "completed",
		VerificationResult: &VerificationResult{
			Verified:   true,
			Confidence: 0.95,
			Reason:     "Student ID found in database",
			Evidence:   []string{"student_id_match", "enrollment_active"},
			Timestamp:  "2025-08-02T07:00:00Z",
		},
		Timestamp: "2025-08-02T07:00:00Z",
		Metadata: map[string]interface{}{
			"dp_id": "dp_university_123",
		},
	}

	parsed, err := service.ParseDPResponse(dpResponse)

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if parsed == nil {
		t.Fatal("Expected parsed response to be returned")
	}

	if parsed.JobID != "job_123456" {
		t.Errorf("Expected job ID 'job_123456', got %s", parsed.JobID)
	}

	if parsed.Status != "completed" {
		t.Errorf("Expected status 'completed', got %s", parsed.Status)
	}

	if !parsed.Verified {
		t.Error("Expected verification to be true")
	}

	if parsed.Confidence != 0.95 {
		t.Errorf("Expected confidence 0.95, got %f", parsed.Confidence)
	}

	if parsed.Reason != "Student ID found in database" {
		t.Errorf("Expected reason 'Student ID found in database', got %s", parsed.Reason)
	}

	if len(parsed.Evidence) != 2 {
		t.Errorf("Expected 2 evidence items, got %d", len(parsed.Evidence))
	}

	if parsed.DPID != "dp_university_123" {
		t.Errorf("Expected DP ID 'dp_university_123', got %s", parsed.DPID)
	}
}

func TestResponseParserService_ParseDPResponse_InvalidJobID(t *testing.T) {
	cfg := &config.Config{
		ResponseValidationEnabled: true,
		IntegrityCheckEnabled:     true,
	}

	service := NewResponseParserService(cfg)

	dpResponse := &DPResponse{
		JobID:  "invalid_job_id", // Invalid format
		Status: "completed",
		VerificationResult: &VerificationResult{
			Verified:   true,
			Confidence: 0.95,
			Timestamp:  "2025-08-02T07:00:00Z",
		},
		Timestamp: "2025-08-02T07:00:00Z",
	}

	parsed, err := service.ParseDPResponse(dpResponse)

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(parsed.ValidationErrors) == 0 {
		t.Error("Expected validation errors for invalid job ID")
	}
}

func TestResponseParserService_ParseDPResponse_InvalidStatus(t *testing.T) {
	cfg := &config.Config{
		ResponseValidationEnabled: true,
		IntegrityCheckEnabled:     true,
	}

	service := NewResponseParserService(cfg)

	dpResponse := &DPResponse{
		JobID:  "job_123456",
		Status: "invalid_status", // Invalid status
		VerificationResult: &VerificationResult{
			Verified:   true,
			Confidence: 0.95,
			Timestamp:  "2025-08-02T07:00:00Z",
		},
		Timestamp: "2025-08-02T07:00:00Z",
	}

	parsed, err := service.ParseDPResponse(dpResponse)

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(parsed.ValidationErrors) == 0 {
		t.Error("Expected validation errors for invalid status")
	}
}

func TestResponseParserService_ParseDPResponse_InvalidConfidence(t *testing.T) {
	cfg := &config.Config{
		ResponseValidationEnabled: true,
		IntegrityCheckEnabled:     true,
	}

	service := NewResponseParserService(cfg)

	dpResponse := &DPResponse{
		JobID:  "job_123456",
		Status: "completed",
		VerificationResult: &VerificationResult{
			Verified:   true,
			Confidence: 1.5, // Invalid confidence (should be 0-1)
			Timestamp:  "2025-08-02T07:00:00Z",
		},
		Timestamp: "2025-08-02T07:00:00Z",
	}

	parsed, err := service.ParseDPResponse(dpResponse)

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(parsed.ValidationErrors) == 0 {
		t.Error("Expected validation errors for invalid confidence")
	}
}

func TestResponseParserService_ParseDPResponse_MissingRequiredFields(t *testing.T) {
	cfg := &config.Config{
		ResponseValidationEnabled: true,
		IntegrityCheckEnabled:     true,
	}

	service := NewResponseParserService(cfg)

	dpResponse := &DPResponse{
		JobID:  "job_123456",
		Status: "completed",
		// Missing VerificationResult
		Timestamp: "2025-08-02T07:00:00Z",
	}

	parsed, err := service.ParseDPResponse(dpResponse)

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(parsed.ValidationErrors) == 0 {
		t.Error("Expected validation errors for missing required fields")
	}
}

func TestResponseParserService_ValidateParsedResponse(t *testing.T) {
	cfg := &config.Config{
		ResponseValidationEnabled: true,
		IntegrityCheckEnabled:     true,
	}

	service := NewResponseParserService(cfg)

	parsed := &ParsedResponse{
		JobID:      "job_123456",
		Status:     "completed",
		Verified:   true,
		Confidence: 0.95,
		Reason:     "Student ID found",
		DPID:       "dp_university_123",
		Timestamp:  "2025-08-02T07:00:00Z",
	}

	errors := service.validateParsedResponse(parsed)

	if len(errors) > 0 {
		t.Errorf("Expected no validation errors, got %v", errors)
	}
}

func TestResponseParserService_ValidateParsedResponse_WithErrors(t *testing.T) {
	cfg := &config.Config{
		ResponseValidationEnabled: true,
		IntegrityCheckEnabled:     true,
	}

	service := NewResponseParserService(cfg)

	parsed := &ParsedResponse{
		JobID:      "invalid_job_id", // Invalid format
		Status:     "invalid_status",  // Invalid status
		Verified:   true,
		Confidence: 1.5, // Invalid confidence
		DPID:       "",  // Missing DP ID
		Timestamp:  "2025-08-02T07:00:00Z",
	}

	errors := service.validateParsedResponse(parsed)

	if len(errors) == 0 {
		t.Error("Expected validation errors")
	}
}

func TestResponseParserService_ValidateField(t *testing.T) {
	cfg := &config.Config{
		ResponseValidationEnabled: true,
		IntegrityCheckEnabled:     true,
	}

	service := NewResponseParserService(cfg)

	// Test valid string field
	err := service.validateField("job_id", "job_123456")
	if err != nil {
		t.Errorf("Expected no error for valid job ID, got %v", err)
	}

	// Test invalid string field
	err = service.validateField("job_id", "invalid_job_id")
	if err == nil {
		t.Error("Expected error for invalid job ID")
	}

	// Test valid boolean field
	err = service.validateField("verified", true)
	if err != nil {
		t.Errorf("Expected no error for valid boolean, got %v", err)
	}

	// Test valid float field
	err = service.validateField("confidence", 0.95)
	if err != nil {
		t.Errorf("Expected no error for valid confidence, got %v", err)
	}

	// Test invalid float field
	err = service.validateField("confidence", 1.5)
	if err == nil {
		t.Error("Expected error for invalid confidence")
	}
}

func TestResponseParserService_MatchesPattern(t *testing.T) {
	cfg := &config.Config{
		ResponseValidationEnabled: true,
		IntegrityCheckEnabled:     true,
	}

	service := NewResponseParserService(cfg)

	// Test valid pattern
	if !service.matchesPattern("job_123456", `^job_[a-zA-Z0-9_-]+$`) {
		t.Error("Expected pattern to match valid job ID")
	}

	// Test invalid pattern
	if service.matchesPattern("invalid_job_id", `^job_[a-zA-Z0-9_-]+$`) {
		t.Error("Expected pattern to not match invalid job ID")
	}
}

func TestResponseParserService_GenerateIntegrityHash(t *testing.T) {
	cfg := &config.Config{
		ResponseValidationEnabled: true,
		IntegrityCheckEnabled:     true,
	}

	service := NewResponseParserService(cfg)

	parsed := &ParsedResponse{
		JobID:      "job_123456",
		Status:     "completed",
		Verified:   true,
		Confidence: 0.95,
		Reason:     "Student ID found",
		DPID:       "dp_university_123",
		Timestamp:  "2025-08-02T07:00:00Z",
	}

	hash := service.generateIntegrityHash(parsed)

	if hash == "" {
		t.Error("Expected integrity hash to be generated")
	}

	// Same response should generate same hash
	hash2 := service.generateIntegrityHash(parsed)
	if hash != hash2 {
		t.Error("Expected same hash for same response")
	}
}

func TestResponseParserService_ValidateResponseIntegrity(t *testing.T) {
	cfg := &config.Config{
		ResponseValidationEnabled: true,
		IntegrityCheckEnabled:     true,
	}

	service := NewResponseParserService(cfg)

	parsed := &ParsedResponse{
		JobID:      "job_123456",
		Status:     "completed",
		Verified:   true,
		Confidence: 0.95,
		Reason:     "Student ID found",
		DPID:       "dp_university_123",
		Timestamp:  "2025-08-02T07:00:00Z",
	}

	// Generate integrity hash
	parsed.IntegrityHash = service.generateIntegrityHash(parsed)

	err := service.ValidateResponseIntegrity(parsed)
	if err != nil {
		t.Errorf("Expected no error for valid integrity, got %v", err)
	}
}

func TestResponseParserService_ValidateResponseIntegrity_Invalid(t *testing.T) {
	cfg := &config.Config{
		ResponseValidationEnabled: true,
		IntegrityCheckEnabled:     true,
	}

	service := NewResponseParserService(cfg)

	parsed := &ParsedResponse{
		JobID:      "job_123456",
		Status:     "completed",
		Verified:   true,
		Confidence: 0.95,
		Reason:     "Student ID found",
		DPID:       "dp_university_123",
		Timestamp:  "2025-08-02T07:00:00Z",
		IntegrityHash: "invalid_hash",
	}

	err := service.ValidateResponseIntegrity(parsed)
	if err == nil {
		t.Error("Expected error for invalid integrity hash")
	}
}

func TestResponseParserService_HandleMalformedResponse(t *testing.T) {
	cfg := &config.Config{
		ResponseValidationEnabled: true,
		IntegrityCheckEnabled:     true,
	}

	service := NewResponseParserService(cfg)

	// Simulate malformed response
	rawResponse := []byte(`{"invalid": json`)
	parseError := "unexpected end of JSON input"

	parsed, err := service.HandleMalformedResponse(rawResponse, parseError)

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if parsed == nil {
		t.Fatal("Expected parsed response to be returned")
	}

	if parsed.Status != "failed" {
		t.Errorf("Expected status 'failed', got %s", parsed.Status)
	}

	if !parsed.Verified {
		t.Error("Expected verification to be false for malformed response")
	}

	if parsed.Confidence != 0.0 {
		t.Errorf("Expected confidence 0.0, got %f", parsed.Confidence)
	}

	if parsed.Reason == "" {
		t.Error("Expected reason to be set for malformed response")
	}
}

func TestResponseParserService_ParseAndValidateResponse(t *testing.T) {
	cfg := &config.Config{
		ResponseValidationEnabled: true,
		IntegrityCheckEnabled:     true,
	}

	service := NewResponseParserService(cfg)

	dpResponse := &DPResponse{
		JobID:  "job_123456",
		Status: "completed",
		VerificationResult: &VerificationResult{
			Verified:   true,
			Confidence: 0.95,
			Reason:     "Student ID found in database",
			Evidence:   []string{"student_id_match", "enrollment_active"},
			Timestamp:  "2025-08-02T07:00:00Z",
		},
		Timestamp: "2025-08-02T07:00:00Z",
		Metadata: map[string]interface{}{
			"dp_id": "dp_university_123",
		},
	}

	parsed, err := service.ParseAndValidateResponse(dpResponse)

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if parsed == nil {
		t.Fatal("Expected parsed response to be returned")
	}

	if parsed.JobID != "job_123456" {
		t.Errorf("Expected job ID 'job_123456', got %s", parsed.JobID)
	}

	if !parsed.Verified {
		t.Error("Expected verification to be true")
	}
}

func TestResponseParserService_ConvertToDPResponse(t *testing.T) {
	cfg := &config.Config{
		ResponseValidationEnabled: true,
		IntegrityCheckEnabled:     true,
	}

	service := NewResponseParserService(cfg)

	parsed := &ParsedResponse{
		JobID:      "job_123456",
		Status:     "completed",
		Verified:   true,
		Confidence: 0.95,
		Reason:     "Student ID found",
		Evidence:   []string{"student_id_match", "enrollment_active"},
		DPID:       "dp_university_123",
		Timestamp:  "2025-08-02T07:00:00Z",
	}

	dpResponse := service.ConvertToDPResponse(parsed)

	if dpResponse == nil {
		t.Fatal("Expected DP response to be returned")
	}

	if dpResponse.JobID != "job_123456" {
		t.Errorf("Expected job ID 'job_123456', got %s", dpResponse.JobID)
	}

	if dpResponse.Status != "completed" {
		t.Errorf("Expected status 'completed', got %s", dpResponse.Status)
	}

	if dpResponse.VerificationResult == nil {
		t.Fatal("Expected verification result to be present")
	}

	if !dpResponse.VerificationResult.Verified {
		t.Error("Expected verification to be true")
	}

	if dpResponse.VerificationResult.Confidence != 0.95 {
		t.Errorf("Expected confidence 0.95, got %f", dpResponse.VerificationResult.Confidence)
	}
}

func TestResponseParserService_GetResponseStats(t *testing.T) {
	cfg := &config.Config{
		ResponseValidationEnabled: true,
		IntegrityCheckEnabled:     true,
	}

	service := NewResponseParserService(cfg)

	stats := service.GetResponseStats()

	if stats == nil {
		t.Fatal("Expected stats to be returned")
	}

	// Check that stats contain expected fields
	expectedFields := []string{"total_responses", "valid_responses", "invalid_responses", "validation_errors"}
	for _, field := range expectedFields {
		if _, exists := stats[field]; !exists {
			t.Errorf("Expected stats to contain field: %s", field)
		}
	}
}

func TestResponseParserService_HealthCheck(t *testing.T) {
	cfg := &config.Config{
		ResponseValidationEnabled: true,
		IntegrityCheckEnabled:     true,
	}

	service := NewResponseParserService(cfg)

	ctx := context.Background()
	err := service.HealthCheck(ctx)

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
} 