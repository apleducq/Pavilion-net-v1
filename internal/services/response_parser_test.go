package services

import (
	"testing"

	"github.com/pavilion-trust/core-broker/internal/config"
)

func TestNewResponseParserService(t *testing.T) {
	cfg := &config.Config{}

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

func TestResponseParserService_ParseAndValidateResponse_Success(t *testing.T) {
	cfg := &config.Config{}

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

	if parsed.Timestamp != "2025-08-02T07:00:00Z" {
		t.Errorf("Expected timestamp '2025-08-02T07:00:00Z', got %s", parsed.Timestamp)
	}
}

func TestResponseParserService_ParseAndValidateResponse_Invalid(t *testing.T) {
	cfg := &config.Config{}

	service := NewResponseParserService(cfg)

	dpResponse := &DPResponse{
		JobID:  "job_123456",
		Status: "invalid_status",
		VerificationResult: &VerificationResult{
			Verified:   true,
			Confidence: 1.5, // Invalid confidence > 1.0
			Timestamp:  "2025-08-02T07:00:00Z",
		},
		Timestamp: "2025-08-02T07:00:00Z",
	}

	_, err := service.ParseAndValidateResponse(dpResponse)

	if err == nil {
		t.Error("Expected error for invalid response")
	}
}

func TestResponseParserService_ConvertToDPResponse(t *testing.T) {
	cfg := &config.Config{}

	service := NewResponseParserService(cfg)

	parsed := &ParsedResponse{
		JobID:      "job_123456",
		Status:     "completed",
		Verified:   true,
		Confidence: 0.95,
		Reason:     "Student ID found",
		Evidence:   []string{"student_id_match"},
		DPID:       "dp_university_123",
		Timestamp:  "2025-08-02T07:00:00Z",
	}

	dpResponse := service.ConvertToDPResponse(parsed)

	if dpResponse == nil {
		t.Fatal("Expected DP response to be returned")
	}

	if dpResponse.Status != "completed" {
		t.Errorf("Expected status 'completed', got %s", dpResponse.Status)
	}

	if !dpResponse.Verified {
		t.Error("Expected verification to be true")
	}

	if dpResponse.ConfidenceScore != 0.95 {
		t.Errorf("Expected confidence 0.95, got %f", dpResponse.ConfidenceScore)
	}

	if dpResponse.Reason != "Student ID found" {
		t.Errorf("Expected reason 'Student ID found', got %s", dpResponse.Reason)
	}

	if dpResponse.DPID != "dp_university_123" {
		t.Errorf("Expected DP ID 'dp_university_123', got %s", dpResponse.DPID)
	}

	if dpResponse.Timestamp != "2025-08-02T07:00:00Z" {
		t.Errorf("Expected timestamp '2025-08-02T07:00:00Z', got %s", dpResponse.Timestamp)
	}
}

func TestResponseParserService_ValidateDPResponse(t *testing.T) {
	cfg := &config.Config{}

	service := NewResponseParserService(cfg)

	dpResponse := &DPResponse{
		JobID:  "job_123456",
		Status: "completed",
		VerificationResult: &VerificationResult{
			Verified:   true,
			Confidence: 0.95,
			Reason:     "Student ID found",
			Timestamp:  "2025-08-02T07:00:00Z",
		},
		Timestamp: "2025-08-02T07:00:00Z",
	}

	// Test validation through parsing
	parsed, err := service.ParseAndValidateResponse(dpResponse)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if parsed == nil {
		t.Fatal("Expected parsed response to be returned")
	}
}

func TestResponseParserService_ValidateDPResponse_Invalid(t *testing.T) {
	cfg := &config.Config{}

	service := NewResponseParserService(cfg)

	dpResponse := &DPResponse{
		JobID:  "job_123456",
		Status: "invalid_status",
		VerificationResult: &VerificationResult{
			Verified:   true,
			Confidence: 1.5, // Invalid confidence
			Timestamp:  "2025-08-02T07:00:00Z",
		},
		Timestamp: "2025-08-02T07:00:00Z",
	}

	// Test validation through parsing
	_, err := service.ParseAndValidateResponse(dpResponse)
	if err == nil {
		t.Error("Expected error for invalid DP response")
	}
}
