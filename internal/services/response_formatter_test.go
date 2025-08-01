package services

import (
	"context"
	"testing"
	"time"

	"github.com/pavilion-trust/core-broker/internal/config"
)

func TestNewResponseFormatterService(t *testing.T) {
	cfg := &config.Config{
		ResponseFormattingEnabled: true,
		ResponseValidationEnabled:  true,
	}

	service := NewResponseFormatterService(cfg)

	if service == nil {
		t.Fatal("Expected service to be created")
	}

	if service.config != cfg {
		t.Error("Expected config to be set")
	}

	if service.validator == nil {
		t.Error("Expected validator to be created")
	}

	if service.templates == nil {
		t.Error("Expected templates to be initialized")
	}
}

func TestResponseFormatterService_FormatResponse_Success(t *testing.T) {
	cfg := &config.Config{
		ResponseFormattingEnabled: true,
		ResponseValidationEnabled:  true,
	}

	service := NewResponseFormatterService(cfg)

	parsedResp := &ParsedResponse{
		JobID:      "job_123456",
		Status:     "completed",
		Verified:   true,
		Confidence: 0.95,
		Reason:     "Student ID found in database",
		Evidence:   []string{"student_id_match", "enrollment_active"},
		DPID:       "dp_university_123",
		Timestamp:  "2025-08-02T07:00:00Z",
	}

	requestID := "req_123456"
	processingTime := 150 * time.Millisecond
	requestHash := "hash_abc123"

	ctx := context.Background()
	formatted, err := service.FormatResponse(ctx, parsedResp, requestID, processingTime, requestHash)

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if formatted == nil {
		t.Fatal("Expected formatted response to be returned")
	}

	if formatted.RequestID != requestID {
		t.Errorf("Expected request ID %s, got %s", requestID, formatted.RequestID)
	}

	if formatted.Status != "completed" {
		t.Errorf("Expected status 'completed', got %s", formatted.Status)
	}

	if !formatted.Verified {
		t.Error("Expected verification to be true")
	}

	if formatted.Confidence != 0.95 {
		t.Errorf("Expected confidence 0.95, got %f", formatted.Confidence)
	}

	if formatted.Reason != "Student ID found in database" {
		t.Errorf("Expected reason 'Student ID found in database', got %s", formatted.Reason)
	}

	if len(formatted.Evidence) != 2 {
		t.Errorf("Expected 2 evidence items, got %d", len(formatted.Evidence))
	}

	if formatted.DPID != "dp_university_123" {
		t.Errorf("Expected DP ID 'dp_university_123', got %s", formatted.DPID)
	}

	if formatted.ProcessingTime != "150ms" {
		t.Errorf("Expected processing time '150ms', got %s", formatted.ProcessingTime)
	}

	if formatted.RequestHash != requestHash {
		t.Errorf("Expected request hash %s, got %s", requestHash, formatted.RequestHash)
	}

	if formatted.ResponseHash == "" {
		t.Error("Expected response hash to be generated")
	}
}

func TestResponseFormatterService_FormatResponse_WithExpiration(t *testing.T) {
	cfg := &config.Config{
		ResponseFormattingEnabled: true,
		ResponseValidationEnabled:  true,
		ResponseExpirationHours:   24,
	}

	service := NewResponseFormatterService(cfg)

	parsedResp := &ParsedResponse{
		JobID:      "job_123456",
		Status:     "completed",
		Verified:   true,
		Confidence: 0.95,
		Reason:     "Student ID found",
		DPID:       "dp_university_123",
		Timestamp:  "2025-08-02T07:00:00Z",
	}

	requestID := "req_123456"
	processingTime := 100 * time.Millisecond
	requestHash := "hash_abc123"

	ctx := context.Background()
	formatted, err := service.FormatResponse(ctx, parsedResp, requestID, processingTime, requestHash)

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if formatted.ExpirationTime == "" {
		t.Error("Expected expiration time to be set")
	}
}

func TestResponseFormatterService_FormatErrorResponse(t *testing.T) {
	cfg := &config.Config{
		ResponseFormattingEnabled: true,
		ResponseValidationEnabled:  true,
	}

	service := NewResponseFormatterService(cfg)

	requestID := "req_123456"
	errorCode := "DP_CONNECTION_ERROR"
	errorMessage := "Failed to connect to data provider"
	processingTime := 200 * time.Millisecond

	ctx := context.Background()
	formatted := service.FormatErrorResponse(ctx, requestID, errorCode, errorMessage, processingTime)

	if formatted == nil {
		t.Fatal("Expected formatted error response to be returned")
	}

	if formatted.RequestID != requestID {
		t.Errorf("Expected request ID %s, got %s", requestID, formatted.RequestID)
	}

	if formatted.Status != "error" {
		t.Errorf("Expected status 'error', got %s", formatted.Status)
	}

	if formatted.Verified {
		t.Error("Expected verification to be false for error response")
	}

	if formatted.Confidence != 0.0 {
		t.Errorf("Expected confidence 0.0, got %f", formatted.Confidence)
	}

	if formatted.Reason != errorMessage {
		t.Errorf("Expected reason '%s', got %s", errorMessage, formatted.Reason)
	}

	if formatted.ProcessingTime != "200ms" {
		t.Errorf("Expected processing time '200ms', got %s", formatted.ProcessingTime)
	}
}

func TestResponseValidator_ValidateFormattedResponse(t *testing.T) {
	cfg := &config.Config{
		ResponseFormattingEnabled: true,
		ResponseValidationEnabled:  true,
	}

	service := NewResponseFormatterService(cfg)

	formatted := &FormattedResponse{
		RequestID:  "req_123456",
		Status:     "completed",
		Verified:   true,
		Confidence: 0.95,
		Reason:     "Student ID found",
		DPID:       "dp_university_123",
		Timestamp:  "2025-08-02T07:00:00Z",
		ProcessingTime: "150ms",
		RequestHash: "hash_abc123",
		ResponseHash: "hash_def456",
	}

	err := service.validator.ValidateFormattedResponse(formatted)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
}

func TestResponseValidator_ValidateFormattedResponse_Invalid(t *testing.T) {
	cfg := &config.Config{
		ResponseFormattingEnabled: true,
		ResponseValidationEnabled:  true,
	}

	service := NewResponseFormatterService(cfg)

	formatted := &FormattedResponse{
		RequestID:  "", // Missing request ID
		Status:     "invalid_status", // Invalid status
		Verified:   true,
		Confidence: 1.5, // Invalid confidence
		DPID:       "", // Missing DP ID
		Timestamp:  "2025-08-02T07:00:00Z",
	}

	err := service.validator.ValidateFormattedResponse(formatted)
	if err == nil {
		t.Error("Expected error for invalid formatted response")
	}
}

func TestResponseFormatterService_ConvertToVerificationResponse(t *testing.T) {
	cfg := &config.Config{
		ResponseFormattingEnabled: true,
		ResponseValidationEnabled:  true,
	}

	service := NewResponseFormatterService(cfg)

	formatted := &FormattedResponse{
		RequestID:  "req_123456",
		Status:     "completed",
		Verified:   true,
		Confidence: 0.95,
		Reason:     "Student ID found",
		Evidence:   []string{"student_id_match", "enrollment_active"},
		DPID:       "dp_university_123",
		Timestamp:  "2025-08-02T07:00:00Z",
		ProcessingTime: "150ms",
		RequestHash: "hash_abc123",
		ResponseHash: "hash_def456",
		Metadata: map[string]interface{}{
			"audit_id": "audit_123",
		},
	}

	verificationResp := service.ConvertToVerificationResponse(formatted)

	if verificationResp == nil {
		t.Fatal("Expected verification response to be returned")
	}

	if verificationResp.RequestID != "req_123456" {
		t.Errorf("Expected request ID 'req_123456', got %s", verificationResp.RequestID)
	}

	if verificationResp.Status != "completed" {
		t.Errorf("Expected status 'completed', got %s", verificationResp.Status)
	}

	if !verificationResp.Verified {
		t.Error("Expected verification to be true")
	}

	if verificationResp.Confidence != 0.95 {
		t.Errorf("Expected confidence 0.95, got %f", verificationResp.Confidence)
	}
}

func TestResponseFormatterService_GenerateResponseHash(t *testing.T) {
	cfg := &config.Config{
		ResponseFormattingEnabled: true,
		ResponseValidationEnabled:  true,
	}

	service := NewResponseFormatterService(cfg)

	formatted := &FormattedResponse{
		RequestID:  "req_123456",
		Status:     "completed",
		Verified:   true,
		Confidence: 0.95,
		Reason:     "Student ID found",
		DPID:       "dp_university_123",
		Timestamp:  "2025-08-02T07:00:00Z",
		ProcessingTime: "150ms",
		RequestHash: "hash_abc123",
	}

	hash := service.GenerateResponseHash(formatted)

	if hash == "" {
		t.Error("Expected response hash to be generated")
	}

	// Same response should generate same hash
	hash2 := service.GenerateResponseHash(formatted)
	if hash != hash2 {
		t.Error("Expected same hash for same response")
	}
}

func TestResponseFormatterService_ValidateResponseIntegrity(t *testing.T) {
	cfg := &config.Config{
		ResponseFormattingEnabled: true,
		ResponseValidationEnabled:  true,
	}

	service := NewResponseFormatterService(cfg)

	formatted := &FormattedResponse{
		RequestID:  "req_123456",
		Status:     "completed",
		Verified:   true,
		Confidence: 0.95,
		Reason:     "Student ID found",
		DPID:       "dp_university_123",
		Timestamp:  "2025-08-02T07:00:00Z",
		ProcessingTime: "150ms",
		RequestHash: "hash_abc123",
	}

	// Generate response hash
	formatted.ResponseHash = service.GenerateResponseHash(formatted)

	err := service.ValidateResponseIntegrity(formatted)
	if err != nil {
		t.Errorf("Expected no error for valid integrity, got %v", err)
	}
}

func TestResponseFormatterService_ValidateResponseIntegrity_Invalid(t *testing.T) {
	cfg := &config.Config{
		ResponseFormattingEnabled: true,
		ResponseValidationEnabled:  true,
	}

	service := NewResponseFormatterService(cfg)

	formatted := &FormattedResponse{
		RequestID:  "req_123456",
		Status:     "completed",
		Verified:   true,
		Confidence: 0.95,
		Reason:     "Student ID found",
		DPID:       "dp_university_123",
		Timestamp:  "2025-08-02T07:00:00Z",
		ProcessingTime: "150ms",
		RequestHash: "hash_abc123",
		ResponseHash: "invalid_hash",
	}

	err := service.ValidateResponseIntegrity(formatted)
	if err == nil {
		t.Error("Expected error for invalid response hash")
	}
}

func TestResponseFormatterService_GetResponseTemplate(t *testing.T) {
	cfg := &config.Config{
		ResponseFormattingEnabled: true,
		ResponseValidationEnabled:  true,
	}

	service := NewResponseFormatterService(cfg)

	template, err := service.GetResponseTemplate("verification")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if template == nil {
		t.Fatal("Expected template to be returned")
	}

	if template.TemplateName != "verification" {
		t.Errorf("Expected template name 'verification', got %s", template.TemplateName)
	}

	if template.Fields == nil {
		t.Error("Expected template fields to be set")
	}
}

func TestResponseFormatterService_GetResponseTemplate_NotFound(t *testing.T) {
	cfg := &config.Config{
		ResponseFormattingEnabled: true,
		ResponseValidationEnabled:  true,
	}

	service := NewResponseFormatterService(cfg)

	_, err := service.GetResponseTemplate("nonexistent_template")
	if err == nil {
		t.Error("Expected error for non-existent template")
	}
}

func TestResponseFormatterService_ListTemplates(t *testing.T) {
	cfg := &config.Config{
		ResponseFormattingEnabled: true,
		ResponseValidationEnabled:  true,
	}

	service := NewResponseFormatterService(cfg)

	templates := service.ListTemplates()

	if len(templates) == 0 {
		t.Error("Expected templates to be returned")
	}

	// Check that verification template is included
	found := false
	for _, template := range templates {
		if template == "verification" {
			found = true
			break
		}
	}

	if !found {
		t.Error("Expected verification template to be included")
	}
}

func TestResponseFormatterService_GetFormattedResponseStats(t *testing.T) {
	cfg := &config.Config{
		ResponseFormattingEnabled: true,
		ResponseValidationEnabled:  true,
	}

	service := NewResponseFormatterService(cfg)

	stats := service.GetFormattedResponseStats()

	if stats == nil {
		t.Fatal("Expected stats to be returned")
	}

	// Check that stats contain expected fields
	expectedFields := []string{"total_responses", "successful_responses", "error_responses", "validation_errors"}
	for _, field := range expectedFields {
		if _, exists := stats[field]; !exists {
			t.Errorf("Expected stats to contain field: %s", field)
		}
	}
}

func TestResponseFormatterService_HealthCheck(t *testing.T) {
	cfg := &config.Config{
		ResponseFormattingEnabled: true,
		ResponseValidationEnabled:  true,
	}

	service := NewResponseFormatterService(cfg)

	ctx := context.Background()
	err := service.HealthCheck(ctx)

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
}

func TestResponseFormatterService_InitializeTemplates(t *testing.T) {
	cfg := &config.Config{
		ResponseFormattingEnabled: true,
		ResponseValidationEnabled:  true,
	}

	service := NewResponseFormatterService(cfg)

	// Check that verification template is initialized
	template, exists := service.templates["verification"]
	if !exists {
		t.Fatal("Expected verification template to be initialized")
	}

	if template.TemplateName != "verification" {
		t.Errorf("Expected template name 'verification', got %s", template.TemplateName)
	}

	// Check required fields
	requiredFields := []string{"request_id", "status", "verified", "confidence"}
	for _, field := range requiredFields {
		if _, exists := template.Fields[field]; !exists {
			t.Errorf("Expected template to contain required field: %s", field)
		}
	}
}

func TestResponseFormatterService_FormatResponse_WithMetadata(t *testing.T) {
	cfg := &config.Config{
		ResponseFormattingEnabled: true,
		ResponseValidationEnabled:  true,
	}

	service := NewResponseFormatterService(cfg)

	parsedResp := &ParsedResponse{
		JobID:      "job_123456",
		Status:     "completed",
		Verified:   true,
		Confidence: 0.95,
		Reason:     "Student ID found",
		DPID:       "dp_university_123",
		Timestamp:  "2025-08-02T07:00:00Z",
		Metadata: map[string]interface{}{
			"audit_id": "audit_123",
			"session_id": "session_456",
		},
	}

	requestID := "req_123456"
	processingTime := 100 * time.Millisecond
	requestHash := "hash_abc123"

	ctx := context.Background()
	formatted, err := service.FormatResponse(ctx, parsedResp, requestID, processingTime, requestHash)

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if formatted.Metadata == nil {
		t.Error("Expected metadata to be preserved")
	}

	if formatted.Metadata["audit_id"] != "audit_123" {
		t.Errorf("Expected audit_id 'audit_123', got %v", formatted.Metadata["audit_id"])
	}

	if formatted.Metadata["session_id"] != "session_456" {
		t.Errorf("Expected session_id 'session_456', got %v", formatted.Metadata["session_id"])
	}
} 