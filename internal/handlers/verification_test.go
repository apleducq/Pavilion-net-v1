package handlers

import (
	"bytes"
	"context"
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
		// Use invalid OPA URL to prevent actual OPA calls in tests
		OPAURL: "http://invalid-opa-url:8181",
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

	// Add validated request to context (simulating validation middleware)
	ctx := httpReq.Context()
	ctx = context.WithValue(ctx, "validated_request", &req)
	httpReq = httpReq.WithContext(ctx)

	// Create response recorder
	w := httptest.NewRecorder()

	// Call handler
	handler.HandleVerification(w, httpReq)

	// Check response status - expect 403 due to policy service failure
	if w.Code != http.StatusForbidden {
		t.Errorf("Expected status 403 (policy violation), got %d", w.Code)
	}

	// Parse response
	var errorResponse models.ErrorResponse
	if err := json.NewDecoder(w.Body).Decode(&errorResponse); err != nil {
		t.Errorf("Failed to decode error response: %v", err)
	}

	// Check error response
	if errorResponse.Error == nil {
		t.Error("Expected error response")
	}
	if errorResponse.Error.Code != "AUTHORIZATION_DENIED" {
		t.Errorf("Expected error code 'AUTHORIZATION_DENIED', got %s", errorResponse.Error.Code)
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

	// Add invalid request to context (simulating validation middleware failure)
	ctx := httpReq.Context()
	ctx = context.WithValue(ctx, "validated_request", nil) // No validated request
	httpReq = httpReq.WithContext(ctx)

	// Create response recorder
	w := httptest.NewRecorder()

	// Call handler
	handler.HandleVerification(w, httpReq)

	// Check response status
	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
} 