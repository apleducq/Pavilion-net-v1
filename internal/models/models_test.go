package models

import (
	"encoding/json"
	"testing"
	"time"
)

func TestVerificationRequest_Validate(t *testing.T) {
	tests := []struct {
		name    string
		request VerificationRequest
		wantErr bool
	}{
		{
			name: "valid request",
			request: VerificationRequest{
				RPID:      "test-rp",
				UserID:    "test-user",
				ClaimType: "student_verification",
				Identifiers: map[string]string{
					"email": "test@example.com",
				},
			},
			wantErr: false,
		},
		{
			name: "missing rp_id",
			request: VerificationRequest{
				UserID:    "test-user",
				ClaimType: "student_verification",
				Identifiers: map[string]string{
					"email": "test@example.com",
				},
			},
			wantErr: true,
		},
		{
			name: "invalid claim type",
			request: VerificationRequest{
				RPID:      "test-rp",
				UserID:    "test-user",
				ClaimType: "invalid_claim",
				Identifiers: map[string]string{
					"email": "test@example.com",
				},
			},
			wantErr: true,
		},
		{
			name: "empty identifiers",
			request: VerificationRequest{
				RPID:      "test-rp",
				UserID:    "test-user",
				ClaimType: "student_verification",
				Identifiers: map[string]string{},
			},
			wantErr: true,
		},
		{
			name: "invalid identifier key",
			request: VerificationRequest{
				RPID:      "test-rp",
				UserID:    "test-user",
				ClaimType: "student_verification",
				Identifiers: map[string]string{
					"invalid_key": "test@example.com",
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.request.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("VerificationRequest.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestVerificationRequest_ToJSON(t *testing.T) {
	req := VerificationRequest{
		RPID:      "test-rp",
		UserID:    "test-user",
		ClaimType: "student_verification",
		Identifiers: map[string]string{
			"email": "test@example.com",
		},
	}

	data, err := req.ToJSON()
	if err != nil {
		t.Errorf("ToJSON() error = %v", err)
		return
	}

	// Verify it's valid JSON
	var parsed VerificationRequest
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Errorf("Failed to parse JSON: %v", err)
	}

	if parsed.RPID != req.RPID {
		t.Errorf("Expected RPID %s, got %s", req.RPID, parsed.RPID)
	}
}

func TestFromJSON(t *testing.T) {
	jsonData := `{
		"rp_id": "test-rp",
		"user_id": "test-user",
		"claim_type": "student_verification",
		"identifiers": {
			"email": "test@example.com"
		}
	}`

	req, err := FromJSON([]byte(jsonData))
	if err != nil {
		t.Errorf("FromJSON() error = %v", err)
		return
	}

	if req.RPID != "test-rp" {
		t.Errorf("Expected RPID test-rp, got %s", req.RPID)
	}
}

func TestNewVerificationResponse(t *testing.T) {
	requestID := "test-request-123"
	response := NewVerificationResponse("verified", 0.95, requestID)

	if response.Status != "verified" {
		t.Errorf("Expected status verified, got %s", response.Status)
	}

	if response.ConfidenceScore != 0.95 {
		t.Errorf("Expected confidence score 0.95, got %f", response.ConfidenceScore)
	}

	if response.RequestID != requestID {
		t.Errorf("Expected request ID %s, got %s", requestID, response.RequestID)
	}

	if response.VerificationID == "" {
		t.Error("Expected verification ID to be set")
	}

	// Check that expires_at is in the future
	expiresAt, err := time.Parse(time.RFC3339, response.ExpiresAt)
	if err != nil {
		t.Errorf("Failed to parse expires_at: %v", err)
	}

	if expiresAt.Before(time.Now()) {
		t.Error("Expected expires_at to be in the future")
	}
}

func TestNewError(t *testing.T) {
	error := NewError("TEST_ERROR", "Test error message", "test-request-123")

	if error.Code != "TEST_ERROR" {
		t.Errorf("Expected code TEST_ERROR, got %s", error.Code)
	}

	if error.Message != "Test error message" {
		t.Errorf("Expected message 'Test error message', got %s", error.Message)
	}

	if error.RequestID != "test-request-123" {
		t.Errorf("Expected request ID test-request-123, got %s", error.RequestID)
	}

	if error.Timestamp == "" {
		t.Error("Expected timestamp to be set")
	}
}

func TestNewErrorResponse(t *testing.T) {
	errorResponse := NewErrorResponse("TEST_ERROR", "Test error message", "test-request-123")

	if errorResponse.Error.Code != "TEST_ERROR" {
		t.Errorf("Expected code TEST_ERROR, got %s", errorResponse.Error.Code)
	}

	if errorResponse.Error.Message != "Test error message" {
		t.Errorf("Expected message 'Test error message', got %s", errorResponse.Error.Message)
	}
}

func TestValidationErrors(t *testing.T) {
	validationErrors := &ValidationErrors{}

	validationErrors.AddError("email", "Invalid email format", "invalid-email")
	validationErrors.AddError("name", "Name is required", "")

	if !validationErrors.HasErrors() {
		t.Error("Expected validation errors to have errors")
	}

	if len(validationErrors.Errors) != 2 {
		t.Errorf("Expected 2 validation errors, got %d", len(validationErrors.Errors))
	}

	if validationErrors.Errors[0].Field != "email" {
		t.Errorf("Expected first error field to be 'email', got %s", validationErrors.Errors[0].Field)
	}

	if validationErrors.Errors[1].Field != "name" {
		t.Errorf("Expected second error field to be 'name', got %s", validationErrors.Errors[1].Field)
	}
}

func TestVerificationResponse_Validate(t *testing.T) {
	response := NewVerificationResponse("verified", 0.95, "test-request-123")

	if err := response.Validate(); err != nil {
		t.Errorf("Valid response should not have validation errors: %v", err)
	}

	// Test invalid status
	response.Status = "invalid_status"
	if err := response.Validate(); err == nil {
		t.Error("Invalid status should have validation error")
	}

	// Test invalid confidence score
	response.Status = "verified"
	response.ConfidenceScore = 1.5 // Should be <= 1
	if err := response.Validate(); err == nil {
		t.Error("Invalid confidence score should have validation error")
	}
} 