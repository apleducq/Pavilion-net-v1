package services

import (
	"context"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/pavilion-trust/core-broker/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewJWSAttestationService(t *testing.T) {
	cfg := &config.Config{
		JWSPath: "/tmp/test-jws",
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
		JWSPath: "/tmp/test-jws",
	}
	
	service := NewJWSAttestationService(cfg)
	
	// Create a test response
	response := &FormattedResponse{
		RequestID:    "test-request-123",
		Status:       "verified",
		Confidence:   0.95,
		Timestamp:    time.Now(),
		ExpiresAt:    time.Now().Add(24 * time.Hour),
		ProcessingTime: 150 * time.Millisecond,
		RequestHash:  "hash_request_123",
		ResponseHash: "hash_response_456",
		Metadata: map[string]string{
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
		JWSPath: "/tmp/test-jws",
	}
	
	service := NewJWSAttestationService(cfg)
	
	// Create a response with invalid data
	response := &FormattedResponse{
		RequestID:    "test-request-123",
		Status:       "error",
		Confidence:   -1.0, // Invalid confidence
		Timestamp:    time.Now(),
		ExpiresAt:    time.Now().Add(24 * time.Hour),
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
		JWSPath: "/tmp/test-jws",
	}
	
	service := NewJWSAttestationService(cfg)
	
	// Generate a JWS first
	response := &FormattedResponse{
		RequestID:    "test-request-123",
		Status:       "verified",
		Confidence:   0.95,
		Timestamp:    time.Now(),
		ExpiresAt:    time.Now().Add(24 * time.Hour),
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
		JWSPath: "/tmp/test-jws",
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
		JWSPath: "/tmp/test-jws",
	}
	
	service := NewJWSAttestationService(cfg)
	
	// Generate a JWS with expired time
	response := &FormattedResponse{
		RequestID:    "test-request-123",
		Status:       "verified",
		Confidence:   0.95,
		Timestamp:    time.Now().Add(-2 * time.Hour), // Expired
		ExpiresAt:    time.Now().Add(-1 * time.Hour), // Expired
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
		JWSPath: "/tmp/test-jws",
	}
	
	service := NewJWSAttestationService(cfg)
	
	// Create valid claims
	claims := &JWSClaims{
		Verified:   true,
		Confidence: 0.95,
		RequestID:  "test-request-123",
		Issuer:     "pavilion-trust",
		Audience:   "relying-party",
		NotBefore:  time.Now().Add(-1 * time.Hour),
		ExpiresAt:  time.Now().Add(1 * time.Hour),
		IssuedAt:   time.Now().Add(-30 * time.Minute),
		JWTID:      "jwt-123",
	}
	
	err := service.VerifyJWSClaims(claims)
	
	assert.NoError(t, err)
}

func TestJWSAttestationService_VerifyJWSClaims_InvalidClaims(t *testing.T) {
	cfg := &config.Config{
		JWSPath: "/tmp/test-jws",
	}
	
	service := NewJWSAttestationService(cfg)
	
	// Test with invalid confidence
	claims := &JWSClaims{
		Verified:   true,
		Confidence: 1.5, // Invalid confidence > 1.0
		RequestID:  "test-request-123",
		Issuer:     "pavilion-trust",
		Audience:   "relying-party",
		NotBefore:  time.Now().Add(-1 * time.Hour),
		ExpiresAt:  time.Now().Add(1 * time.Hour),
		IssuedAt:   time.Now().Add(-30 * time.Minute),
		JWTID:      "jwt-123",
	}
	
	err := service.VerifyJWSClaims(claims)
	
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "confidence")
}

func TestJWSAttestationService_HandleJWSSigningError(t *testing.T) {
	cfg := &config.Config{
		JWSPath: "/tmp/test-jws",
	}
	
	service := NewJWSAttestationService(cfg)
	
	// Test with a signing error
	signingError := fmt.Errorf("failed to sign JWS token")
	requestID := "test-request-123"
	
	err := service.HandleJWSSigningError(signingError, requestID)
	
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "JWS signing failed")
	assert.Contains(t, err.Error(), requestID)
}

func TestJWSAttestationService_GetPublicKeyPEM(t *testing.T) {
	cfg := &config.Config{
		JWSPath: "/tmp/test-jws",
	}
	
	service := NewJWSAttestationService(cfg)
	
	pemData, err := service.GetPublicKeyPEM()
	
	require.NoError(t, err)
	assert.NotEmpty(t, pemData)
	assert.True(t, strings.HasPrefix(pemData, "-----BEGIN PUBLIC KEY-----"))
	assert.True(t, strings.HasSuffix(pemData, "-----END PUBLIC KEY-----\n"))
	
	// Verify it's valid PEM
	block, _ := pem.Decode([]byte(pemData))
	assert.NotNil(t, block)
	assert.Equal(t, "PUBLIC KEY", block.Type)
}

func TestJWSAttestationService_GetJWK(t *testing.T) {
	cfg := &config.Config{
		JWSPath: "/tmp/test-jws",
	}
	
	service := NewJWSAttestationService(cfg)
	
	jwk, err := service.GetJWK()
	
	require.NoError(t, err)
	assert.NotNil(t, jwk)
	assert.Equal(t, "RSA", jwk["kty"])
	assert.Equal(t, "RS256", jwk["alg"])
	assert.NotEmpty(t, jwk["kid"])
	assert.NotEmpty(t, jwk["n"]) // modulus
	assert.NotEmpty(t, jwk["e"]) // exponent
}

func TestJWSAttestationService_GenerateJWSID(t *testing.T) {
	cfg := &config.Config{
		JWSPath: "/tmp/test-jws",
	}
	
	service := NewJWSAttestationService(cfg)
	
	jwsID1 := service.generateJWSID()
	jwsID2 := service.generateJWSID()
	
	assert.NotEmpty(t, jwsID1)
	assert.NotEmpty(t, jwsID2)
	assert.NotEqual(t, jwsID1, jwsID2) // Should be unique
	assert.True(t, strings.HasPrefix(jwsID1, "jws_"))
	assert.True(t, strings.HasPrefix(jwsID2, "jws_"))
}

func TestJWSAuditLogger_LogEvent(t *testing.T) {
	logger := &JWSAuditLogger{
		events: make([]JWSAuditEvent, 0),
	}
	
	eventType := "jws_generated"
	description := "JWS token generated successfully"
	requestID := "test-request-123"
	jwsID := "jws_abc123"
	status := "success"
	metadata := map[string]string{
		"issuer":   "pavilion-trust",
		"audience": "relying-party",
	}
	
	logger.LogEvent(eventType, description, requestID, jwsID, status, metadata)
	
	events := logger.GetAuditEvents()
	assert.Len(t, events, 1)
	
	event := events[0]
	assert.Equal(t, eventType, event.EventType)
	assert.Equal(t, description, event.Description)
	assert.Equal(t, requestID, event.RequestID)
	assert.Equal(t, jwsID, event.JWSID)
	assert.Equal(t, status, event.Status)
	assert.Equal(t, metadata, event.Metadata)
	assert.NotEmpty(t, event.Timestamp)
}

func TestJWSAuditLogger_GetAuditEvents(t *testing.T) {
	logger := &JWSAuditLogger{
		events: make([]JWSAuditEvent, 0),
	}
	
	// Add multiple events
	logger.LogEvent("jws_generated", "Event 1", "req1", "jws1", "success", nil)
	logger.LogEvent("jws_validated", "Event 2", "req2", "jws2", "success", nil)
	logger.LogEvent("jws_expired", "Event 3", "req3", "jws3", "error", nil)
	
	events := logger.GetAuditEvents()
	
	assert.Len(t, events, 3)
	assert.Equal(t, "jws_generated", events[0].EventType)
	assert.Equal(t, "jws_validated", events[1].EventType)
	assert.Equal(t, "jws_expired", events[2].EventType)
}

func TestJWSAttestationService_GetJWSStats(t *testing.T) {
	cfg := &config.Config{
		JWSPath: "/tmp/test-jws",
	}
	
	service := NewJWSAttestationService(cfg)
	
	// Generate some JWS tokens to build stats
	response := &FormattedResponse{
		RequestID:    "test-request-123",
		Status:       "verified",
		Confidence:   0.95,
		Timestamp:    time.Now(),
		ExpiresAt:    time.Now().Add(24 * time.Hour),
	}
	
	ctx := context.Background()
	issuer := "pavilion-trust"
	audience := "relying-party"
	
	_, err := service.GenerateJWS(ctx, response, issuer, audience)
	require.NoError(t, err)
	
	stats := service.GetJWSStats()
	
	assert.NotNil(t, stats)
	assert.Contains(t, stats, "total_generated")
	assert.Contains(t, stats, "total_validated")
	assert.Contains(t, stats, "total_failed")
	assert.Contains(t, stats, "key_id")
	assert.Contains(t, stats, "algorithm")
}

func TestJWSAttestationService_HealthCheck(t *testing.T) {
	cfg := &config.Config{
		JWSPath: "/tmp/test-jws",
	}
	
	service := NewJWSAttestationService(cfg)
	
	ctx := context.Background()
	err := service.HealthCheck(ctx)
	
	assert.NoError(t, err)
}

func TestJWSAttestationService_InitializeKeys(t *testing.T) {
	cfg := &config.Config{
		JWSPath: "/tmp/test-jws",
	}
	
	service := &JWSAttestationService{
		config: cfg,
	}
	
	err := service.initializeKeys()
	
	assert.NoError(t, err)
	assert.NotNil(t, service.privateKey)
	assert.NotNil(t, service.publicKey)
	assert.NotEmpty(t, service.keyID)
	
	// Verify key pair is valid
	assert.Equal(t, service.privateKey.PublicKey, *service.publicKey)
	
	// Test key can be used for signing
	message := []byte("test message")
	signature, err := service.privateKey.Sign(nil, message, nil)
	assert.NoError(t, err)
	assert.NotNil(t, signature)
}

func TestJWSAttestationService_GenerateJWS_WithEvidence(t *testing.T) {
	cfg := &config.Config{
		JWSPath: "/tmp/test-jws",
	}
	
	service := NewJWSAttestationService(cfg)
	
	// Create a response with evidence
	response := &FormattedResponse{
		RequestID:    "test-request-123",
		Status:       "verified",
		Confidence:   0.95,
		Timestamp:    time.Now(),
		ExpiresAt:    time.Now().Add(24 * time.Hour),
		Metadata: map[string]string{
			"evidence": "document_verified,biometric_match",
		},
	}
	
	ctx := context.Background()
	issuer := "pavilion-trust"
	audience := "relying-party"
	
	result, err := service.GenerateJWS(ctx, response, issuer, audience)
	
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Contains(t, result.Payload.Claims.Evidence, "document_verified")
	assert.Contains(t, result.Payload.Claims.Evidence, "biometric_match")
}

func TestJWSAttestationService_GenerateJWS_WithReason(t *testing.T) {
	cfg := &config.Config{
		JWSPath: "/tmp/test-jws",
	}
	
	service := NewJWSAttestationService(cfg)
	
	// Create a response with reason
	response := &FormattedResponse{
		RequestID:    "test-request-123",
		Status:       "verified",
		Confidence:   0.95,
		Timestamp:    time.Now(),
		ExpiresAt:    time.Now().Add(24 * time.Hour),
		Metadata: map[string]string{
			"reason": "User identity verified through multiple factors",
		},
	}
	
	ctx := context.Background()
	issuer := "pavilion-trust"
	audience := "relying-party"
	
	result, err := service.GenerateJWS(ctx, response, issuer, audience)
	
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "User identity verified through multiple factors", result.Payload.Claims.Reason)
} 