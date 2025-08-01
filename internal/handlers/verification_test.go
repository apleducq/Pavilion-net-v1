package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/pavilion-trust/core-broker/internal/config"
	"github.com/pavilion-trust/core-broker/internal/models"
)

func TestVerificationHandler_HandleVerification(t *testing.T) {
	// Create test configuration
	cfg := &config.Config{
		Port: "8080",
		Env:  "test",
	}

	// Create handler
	handler := NewVerificationHandler(cfg)

	// Create test request
	req := models.VerificationRequest{
		RPID:   "test-rp",
		UserID: "test-user",
		ClaimType: "student_verification",
		Identifiers: map[string]string{
			"email": "test@example.com",
			"name":  "Test User",
		},
	}

	// Convert to JSON
	reqBody, _ := json.Marshal(req)

	// Create HTTP request
	httpReq := httptest.NewRequest("POST", "/api/v1/verify", bytes.NewBuffer(reqBody))
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer test-token")

	// Create response recorder
	w := httptest.NewRecorder()

	// Call handler
	handler.HandleVerification(w, httpReq)

	// Check response status
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	// Parse response
	var response models.VerificationResponse
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Errorf("Failed to decode response: %v", err)
	}

	// Check response fields
	if response.VerificationID == "" {
		t.Error("Expected verification ID to be set")
	}
	if response.Status == "" {
		t.Error("Expected status to be set")
	}
	if response.ConfidenceScore == 0 {
		t.Error("Expected confidence score to be set")
	}
}

func TestVerificationHandler_HandleVerification_InvalidRequest(t *testing.T) {
	// Create test configuration
	cfg := &config.Config{
		Port: "8080",
		Env:  "test",
	}

	// Create handler
	handler := NewVerificationHandler(cfg)

	// Create invalid request (missing required fields)
	req := models.VerificationRequest{
		RPID: "test-rp",
		// Missing UserID and ClaimType
	}

	// Convert to JSON
	reqBody, _ := json.Marshal(req)

	// Create HTTP request
	httpReq := httptest.NewRequest("POST", "/api/v1/verify", bytes.NewBuffer(reqBody))
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer test-token")

	// Create response recorder
	w := httptest.NewRecorder()

	// Call handler
	handler.HandleVerification(w, httpReq)

	// Check response status
	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
} 