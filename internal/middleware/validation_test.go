package middleware

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/pavilion-trust/core-broker/internal/models"
)

func TestValidationMiddleware_ValidRequest(t *testing.T) {
	validRequest := models.VerificationRequest{
		RPID:      "test-rp",
		UserID:    "test-user",
		ClaimType: "student_verification",
		Identifiers: map[string]string{
			"email": "test@example.com",
		},
	}

	requestBody, _ := json.Marshal(validRequest)

	req := httptest.NewRequest("POST", "/api/v1/verify", bytes.NewBuffer(requestBody))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()

	handlerCalled := false
	handler := ValidationMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true
		
		// Check that validated request is in context
		validatedReq := getValidatedRequest(r.Context())
		if validatedReq == nil {
			t.Error("Expected validated request in context")
		}
		
		if validatedReq.RPID != validRequest.RPID {
			t.Errorf("Expected RPID %s, got %s", validRequest.RPID, validatedReq.RPID)
		}
	}))

	handler.ServeHTTP(w, req)

	if !handlerCalled {
		t.Error("Expected handler to be called")
	}

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

func TestValidationMiddleware_InvalidJSON(t *testing.T) {
	req := httptest.NewRequest("POST", "/api/v1/verify", bytes.NewBufferString("invalid json"))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()

	handlerCalled := false
	handler := ValidationMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true
	}))

	handler.ServeHTTP(w, req)

	if handlerCalled {
		t.Error("Handler should not be called for invalid JSON")
	}

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}

	// Check error response structure
	var response map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Errorf("Failed to decode response: %v", err)
	}

	if errorObj, ok := response["error"].(map[string]interface{}); ok {
		if errorObj["code"] != "INVALID_JSON" {
			t.Errorf("Expected error code INVALID_JSON, got %s", errorObj["code"])
		}
	} else {
		t.Error("Expected error object in response")
	}
}

func TestValidationMiddleware_InvalidRequest(t *testing.T) {
	invalidRequest := models.VerificationRequest{
		RPID:      "", // Missing required field
		UserID:    "test-user",
		ClaimType: "student_verification",
		Identifiers: map[string]string{
			"email": "test@example.com",
		},
	}

	requestBody, _ := json.Marshal(invalidRequest)

	req := httptest.NewRequest("POST", "/api/v1/verify", bytes.NewBuffer(requestBody))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()

	handlerCalled := false
	handler := ValidationMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true
	}))

	handler.ServeHTTP(w, req)

	if handlerCalled {
		t.Error("Handler should not be called for invalid request")
	}

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}

	// Check error response structure
	var response map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Errorf("Failed to decode response: %v", err)
	}

	if errorObj, ok := response["error"].(map[string]interface{}); ok {
		if errorObj["code"] != "VALIDATION_FAILED" {
			t.Errorf("Expected error code VALIDATION_FAILED, got %s", errorObj["code"])
		}
	} else {
		t.Error("Expected error object in response")
	}
}

func TestValidationMiddleware_NonPOSTRequest(t *testing.T) {
	req := httptest.NewRequest("GET", "/api/v1/verify", nil)
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()

	handlerCalled := false
	handler := ValidationMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true
	}))

	handler.ServeHTTP(w, req)

	if !handlerCalled {
		t.Error("Handler should be called for non-POST requests")
	}

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

func TestValidationMiddleware_NonJSONContent(t *testing.T) {
	req := httptest.NewRequest("POST", "/api/v1/verify", bytes.NewBufferString("plain text"))
	req.Header.Set("Content-Type", "text/plain")

	w := httptest.NewRecorder()

	handlerCalled := false
	handler := ValidationMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true
	}))

	handler.ServeHTTP(w, req)

	if !handlerCalled {
		t.Error("Handler should be called for non-JSON content")
	}

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

func TestGetValidatedRequest(t *testing.T) {
	ctx := context.Background()
	
	// Test with no validated request in context
	req := getValidatedRequest(ctx)
	if req != nil {
		t.Error("Expected nil when no validated request in context")
	}
	
	// Test with validated request in context
	testRequest := &models.VerificationRequest{
		RPID:      "test-rp",
		UserID:    "test-user",
		ClaimType: "student_verification",
		Identifiers: map[string]string{
			"email": "test@example.com",
		},
	}
	
	ctxWithRequest := context.WithValue(ctx, "validated_request", testRequest)
	req = getValidatedRequest(ctxWithRequest)
	
	if req == nil {
		t.Error("Expected validated request from context")
	}
	
	if req.RPID != testRequest.RPID {
		t.Errorf("Expected RPID %s, got %s", testRequest.RPID, req.RPID)
	}
} 