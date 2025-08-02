package services

import (
	"context"
	"testing"
	"time"

	"github.com/pavilion-trust/core-broker/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewJWSAttestationService(t *testing.T) {
	cfg := &config.Config{
		TLSCertFile: "/tmp/test-jws/cert.pem",
		TLSKeyFile:  "/tmp/test-jws/key.pem",
	}

	service := NewJWSAttestationService(cfg)

	assert.NotNil(t, service)
	assert.NotNil(t, service.config)
	assert.NotNil(t, service.privateKey)
	assert.NotNil(t, service.publicKey)
	assert.NotEmpty(t, service.keyID)
	assert.NotNil(t, service.auditLogger)
}

func TestJWSAttestationService_GenerateJWS_Success(t *testing.T) {
	cfg := &config.Config{
		TLSCertFile: "/tmp/test-jws/cert.pem",
		TLSKeyFile:  "/tmp/test-jws/key.pem",
	}

	service := NewJWSAttestationService(cfg)

	// Create a test response
	response := &FormattedResponse{
		RequestID:      "test-request-123",
		Status:         "verified",
		Confidence:     0.95,
		Timestamp:      time.Now().Format(time.RFC3339),
		ProcessingTime: "150ms",
		RequestHash:    "hash_request_123",
		ResponseHash:   "hash_response_456",
		Metadata: map[string]interface{}{
			"dp_id": "test-dp-001",
		},
	}

	ctx := context.Background()
	issuer := "pavilion-trust"
	audience := "relying-party"

	result, err := service.GenerateJWS(ctx, response, issuer, audience)

	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.NotEmpty(t, result.Token)
	assert.NotEmpty(t, result.JWSID)
	assert.Equal(t, "test-request-123", result.RequestID)
	assert.Equal(t, issuer, result.Payload.Claims.Issuer)
	assert.Equal(t, audience, result.Payload.Claims.Audience)
	assert.True(t, result.Payload.Claims.Verified)
	assert.Equal(t, 0.95, result.Payload.Claims.Confidence)
	assert.Equal(t, "hash_request_123", result.Payload.Claims.RequestHash)
	assert.Equal(t, "hash_response_456", result.Payload.Claims.ResponseHash)
	assert.Equal(t, "150ms", result.Payload.Claims.ProcessingTime)
}

func TestJWSAttestationService_GenerateJWS_WithError(t *testing.T) {
	cfg := &config.Config{
		TLSCertFile: "/tmp/test-jws/cert.pem",
		TLSKeyFile:  "/tmp/test-jws/key.pem",
	}

	service := NewJWSAttestationService(cfg)

	// Create a response with invalid data
	response := &FormattedResponse{
		RequestID:  "test-request-123",
		Status:     "error",
		Confidence: -1.0, // Invalid confidence
		Timestamp:  time.Now().Format(time.RFC3339),
	}

	ctx := context.Background()
	issuer := "pavilion-trust"
	audience := "relying-party"

	result, err := service.GenerateJWS(ctx, response, issuer, audience)

	require.NoError(t, err) // JWS generation should still succeed
	assert.NotNil(t, result)
	assert.False(t, result.Payload.Claims.Verified)
	assert.Equal(t, -1.0, result.Payload.Claims.Confidence)
}

func TestJWSAttestationService_ValidateJWS_Success(t *testing.T) {
	cfg := &config.Config{
		TLSCertFile: "/tmp/test-jws/cert.pem",
		TLSKeyFile:  "/tmp/test-jws/key.pem",
	}

	service := NewJWSAttestationService(cfg)

	// Generate a JWS first
	response := &FormattedResponse{
		RequestID:  "test-request-123",
		Status:     "verified",
		Confidence: 0.95,
		Timestamp:  time.Now().Format(time.RFC3339),
	}

	ctx := context.Background()
	issuer := "pavilion-trust"
	audience := "relying-party"

	jwsResult, err := service.GenerateJWS(ctx, response, issuer, audience)
	require.NoError(t, err)

	// Validate the JWS
	claims, err := service.ValidateJWS(ctx, jwsResult.Token)

	require.NoError(t, err)
	assert.NotNil(t, claims)
	assert.True(t, claims.Verified)
	assert.Equal(t, 0.95, claims.Confidence)
	assert.Equal(t, "test-request-123", claims.RequestID)
	assert.Equal(t, issuer, claims.Issuer)
	assert.Equal(t, audience, claims.Audience)
}

func TestJWSAttestationService_ValidateJWS_InvalidToken(t *testing.T) {
	cfg := &config.Config{
		TLSCertFile: "/tmp/test-jws/cert.pem",
		TLSKeyFile:  "/tmp/test-jws/key.pem",
	}

	service := NewJWSAttestationService(cfg)

	ctx := context.Background()

	// Test with invalid token
	claims, err := service.ValidateJWS(ctx, "invalid.jws.token")

	assert.Error(t, err)
	assert.Nil(t, claims)
	assert.Contains(t, err.Error(), "invalid")
}

func TestJWSAttestationService_ValidateJWS_ExpiredToken(t *testing.T) {
	cfg := &config.Config{
		TLSCertFile: "/tmp/test-jws/cert.pem",
		TLSKeyFile:  "/tmp/test-jws/key.pem",
	}

	service := NewJWSAttestationService(cfg)

	// Generate a JWS with expired time
	response := &FormattedResponse{
		RequestID:      "test-request-123",
		Status:         "verified",
		Confidence:     0.95,
		Timestamp:      time.Now().Add(-2 * time.Hour).Format(time.RFC3339), // Expired
		ExpirationTime: time.Now().Add(-1 * time.Hour).Format(time.RFC3339), // Expired
	}

	ctx := context.Background()
	issuer := "pavilion-trust"
	audience := "relying-party"

	jwsResult, err := service.GenerateJWS(ctx, response, issuer, audience)
	require.NoError(t, err)

	// Validate the expired JWS
	claims, err := service.ValidateJWS(ctx, jwsResult.Token)

	assert.Error(t, err)
	assert.Nil(t, claims)
	assert.Contains(t, err.Error(), "expired")
}

func TestJWSAttestationService_VerifyJWSClaims_Success(t *testing.T) {
	cfg := &config.Config{
		TLSCertFile: "/tmp/test-jws/cert.pem",
		TLSKeyFile:  "/tmp/test-jws/key.pem",
	}

	service := NewJWSAttestationService(cfg)

	// Create valid claims
	claims := &JWSClaims{
		Verified:       true,
		Confidence:     0.95,
		RequestID:      "test-request-123",
		Issuer:         "pavilion-trust",
		Audience:       "relying-party",
		RequestHash:    "hash_request_123",
		ResponseHash:   "hash_response_456",
		ProcessingTime: "150ms",
	}

	// Verify the claims
	err := service.VerifyJWSClaims(claims)

	assert.NoError(t, err)
}

func TestJWSAttestationService_VerifyJWSClaims_InvalidClaims(t *testing.T) {
	cfg := &config.Config{
		TLSCertFile: "/tmp/test-jws/cert.pem",
		TLSKeyFile:  "/tmp/test-jws/key.pem",
	}

	service := NewJWSAttestationService(cfg)

	// Create invalid claims
	claims := &JWSClaims{
		Verified:   true,
		Confidence: -1.0, // Invalid confidence
		RequestID:  "",   // Empty request ID
		Issuer:     "pavilion-trust",
		Audience:   "relying-party",
	}

	// Verify the claims should fail
	err := service.VerifyJWSClaims(claims)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid")
}

func TestJWSAttestationService_HandleJWSSigningError(t *testing.T) {
	cfg := &config.Config{
		TLSCertFile: "/tmp/test-jws/cert.pem",
		TLSKeyFile:  "/tmp/test-jws/key.pem",
	}

	service := NewJWSAttestationService(cfg)

	// Create a response that might cause signing issues
	response := &FormattedResponse{
		RequestID:  "test-request-123",
		Status:     "verified",
		Confidence: 0.95,
		Timestamp:  time.Now().Format(time.RFC3339),
	}

	ctx := context.Background()
	issuer := "pavilion-trust"
	audience := "relying-party"

	// This should handle any signing errors gracefully
	result, err := service.GenerateJWS(ctx, response, issuer, audience)

	// Should not panic and should handle errors gracefully
	if err != nil {
		assert.Contains(t, err.Error(), "signing")
	} else {
		assert.NotNil(t, result)
	}
}

func TestJWSAttestationService_GetPublicKeyPEM(t *testing.T) {
	cfg := &config.Config{
		TLSCertFile: "/tmp/test-jws/cert.pem",
		TLSKeyFile:  "/tmp/test-jws/key.pem",
	}

	service := NewJWSAttestationService(cfg)

	// Get public key PEM
	pemData, err := service.GetPublicKeyPEM()
	require.NoError(t, err)

	assert.NotEmpty(t, pemData)
	assert.Contains(t, pemData, "-----BEGIN PUBLIC KEY-----")
	assert.Contains(t, pemData, "-----END PUBLIC KEY-----")
}

func TestJWSAttestationService_GetJWK(t *testing.T) {
	cfg := &config.Config{
		TLSCertFile: "/tmp/test-jws/cert.pem",
		TLSKeyFile:  "/tmp/test-jws/key.pem",
	}

	service := NewJWSAttestationService(cfg)

	// Get JWK
	jwk, err := service.GetJWK()
	require.NoError(t, err)

	assert.NotNil(t, jwk)
	assert.NotEmpty(t, jwk["kty"])
	assert.NotEmpty(t, jwk["kid"])
	assert.NotEmpty(t, jwk["n"])
	assert.NotEmpty(t, jwk["e"])
}

func TestJWSAttestationService_GenerateJWSID(t *testing.T) {
	cfg := &config.Config{
		TLSCertFile: "/tmp/test-jws/cert.pem",
		TLSKeyFile:  "/tmp/test-jws/key.pem",
	}

	service := NewJWSAttestationService(cfg)

	// Generate JWS ID
	jwsID1 := service.generateJWSID()
	jwsID2 := service.generateJWSID()

	assert.NotEmpty(t, jwsID1)
	assert.NotEmpty(t, jwsID2)
	assert.NotEqual(t, jwsID1, jwsID2) // Should be unique
	assert.Contains(t, jwsID1, "jws_")
}

func TestJWSAuditLogger_LogEvent(t *testing.T) {
	cfg := &config.Config{
		TLSCertFile: "/tmp/test-jws/cert.pem",
		TLSKeyFile:  "/tmp/test-jws/key.pem",
	}

	service := NewJWSAttestationService(cfg)

	// Test logging JWS events
	eventType := "jws_generated"
	description := "JWS token generated successfully"
	requestID := "test-request-123"
	jwsID := "jws_test123"
	status := "success"
	metadata := map[string]string{
		"issuer":   "pavilion-trust",
		"audience": "relying-party",
	}

	// This should not panic
	service.auditLogger.LogEvent(eventType, description, requestID, jwsID, status, metadata)
}

func TestJWSAuditLogger_GetAuditEvents(t *testing.T) {
	cfg := &config.Config{
		TLSCertFile: "/tmp/test-jws/cert.pem",
		TLSKeyFile:  "/tmp/test-jws/key.pem",
	}

	service := NewJWSAttestationService(cfg)

	// Get audit events
	events := service.auditLogger.GetAuditEvents()

	// Should return events (might be empty in test)
	assert.NotNil(t, events)
}

func TestJWSAttestationService_GetJWSStats(t *testing.T) {
	cfg := &config.Config{
		TLSCertFile: "/tmp/test-jws/cert.pem",
		TLSKeyFile:  "/tmp/test-jws/key.pem",
	}

	service := NewJWSAttestationService(cfg)

	// Get JWS statistics
	stats := service.GetJWSStats()

	assert.NotNil(t, stats)
	assert.Contains(t, stats, "total_generated")
	assert.Contains(t, stats, "total_validated")
	assert.Contains(t, stats, "error_count")
	assert.Contains(t, stats, "average_generation_time")
}

func TestJWSAttestationService_HealthCheck(t *testing.T) {
	cfg := &config.Config{
		TLSCertFile: "/tmp/test-jws/cert.pem",
		TLSKeyFile:  "/tmp/test-jws/key.pem",
	}

	service := NewJWSAttestationService(cfg)

	ctx := context.Background()
	err := service.HealthCheck(ctx)

	assert.NoError(t, err)
}

func TestJWSAttestationService_InitializeKeys(t *testing.T) {
	cfg := &config.Config{
		TLSCertFile: "/tmp/test-jws/cert.pem",
		TLSKeyFile:  "/tmp/test-jws/key.pem",
	}

	service := NewJWSAttestationService(cfg)

	// Test key initialization
	err := service.initializeKeys()

	assert.NoError(t, err)
	assert.NotNil(t, service.privateKey)
	assert.NotNil(t, service.publicKey)
	assert.NotEmpty(t, service.keyID)
}

func TestJWSAttestationService_GenerateJWS_WithEvidence(t *testing.T) {
	cfg := &config.Config{
		TLSCertFile: "/tmp/test-jws/cert.pem",
		TLSKeyFile:  "/tmp/test-jws/key.pem",
	}

	service := NewJWSAttestationService(cfg)

	// Create a test response with evidence
	response := &FormattedResponse{
		RequestID:  "test-request-123",
		Status:     "verified",
		Confidence: 0.95,
		Timestamp:  time.Now().Format(time.RFC3339),
		Evidence:   []string{"document_verified", "biometric_match"},
		Reason:     "Multiple verification factors confirmed",
	}

	ctx := context.Background()
	issuer := "pavilion-trust"
	audience := "relying-party"

	result, err := service.GenerateJWS(ctx, response, issuer, audience)

	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.NotEmpty(t, result.Token)
	assert.Equal(t, "Multiple verification factors confirmed", result.Payload.Claims.Reason)
}

func TestJWSAttestationService_GenerateJWS_WithReason(t *testing.T) {
	cfg := &config.Config{
		TLSCertFile: "/tmp/test-jws/cert.pem",
		TLSKeyFile:  "/tmp/test-jws/key.pem",
	}

	service := NewJWSAttestationService(cfg)

	// Create a test response with reason
	response := &FormattedResponse{
		RequestID:  "test-request-123",
		Status:     "verified",
		Confidence: 0.95,
		Timestamp:  time.Now().Format(time.RFC3339),
		Reason:     "Student enrollment confirmed through university records",
	}

	ctx := context.Background()
	issuer := "pavilion-trust"
	audience := "relying-party"

	result, err := service.GenerateJWS(ctx, response, issuer, audience)

	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.NotEmpty(t, result.Token)
	assert.Equal(t, "Student enrollment confirmed through university records", result.Payload.Claims.Reason)
}
