package services

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"math/big"
	"strings"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/pavilion-trust/core-broker/internal/config"
)

// JWSAttestationService handles JWS token generation and validation
type JWSAttestationService struct {
	config *config.Config
	// Private key for signing
	privateKey *rsa.PrivateKey
	// Public key for verification
	publicKey *rsa.PublicKey
	// Key ID for JWS header
	keyID string
	// Audit logger for JWS events
	auditLogger *JWSAuditLogger
}

// JWSAuditLogger handles JWS-related audit logging
type JWSAuditLogger struct {
	mu     sync.Mutex
	events []JWSAuditEvent
}

// JWSAuditEvent represents a JWS audit event
type JWSAuditEvent struct {
	Timestamp   string            `json:"timestamp"`
	EventType   string            `json:"event_type"`
	RequestID   string            `json:"request_id"`
	JWSID       string            `json:"jws_id,omitempty"`
	Description string            `json:"description"`
	Status      string            `json:"status"`
	Metadata    map[string]string `json:"metadata,omitempty"`
}

// JWSClaims represents the claims in a JWS token
type JWSClaims struct {
	jwt.RegisteredClaims
	// Verification-specific claims
	Verified        bool      `json:"verified"`
	Confidence      float64   `json:"confidence"`
	Reason          string    `json:"reason,omitempty"`
	Evidence        []string  `json:"evidence,omitempty"`
	DPID            string    `json:"dp_id"`
	RequestID       string    `json:"request_id"`
	ProcessingTime  string    `json:"processing_time,omitempty"`
	RequestHash     string    `json:"request_hash,omitempty"`
	ResponseHash    string    `json:"response_hash,omitempty"`
	AttestationType string    `json:"attestation_type"`
	Issuer          string    `json:"iss"`
	Audience        string    `json:"aud"`
	NotBefore       time.Time `json:"nbf"`
	ExpiresAt       time.Time `json:"exp"`
	IssuedAt        time.Time `json:"iat"`
	JWTID           string    `json:"jti"`
}

// JWSHeader represents the header of a JWS token
type JWSHeader struct {
	Algorithm string `json:"alg"`
	Type      string `json:"typ"`
	KeyID     string `json:"kid"`
}

// JWSPayload represents the payload of a JWS token
type JWSPayload struct {
	Claims JWSClaims `json:"claims"`
}

// JWSResult represents a JWS token result
type JWSResult struct {
	Token        string            `json:"token"`
	Header       JWSHeader         `json:"header"`
	Payload      JWSPayload        `json:"payload"`
	Signature    string            `json:"signature"`
	JWSID        string            `json:"jws_id"`
	RequestID    string            `json:"request_id"`
	CreatedAt    time.Time         `json:"created_at"`
	ExpiresAt    time.Time         `json:"expires_at"`
	Metadata     map[string]string `json:"metadata,omitempty"`
}

// NewJWSAttestationService creates a new JWS attestation service
func NewJWSAttestationService(cfg *config.Config) *JWSAttestationService {
	service := &JWSAttestationService{
		config: cfg,
		keyID:  "pavilion-core-broker-v1",
		auditLogger: &JWSAuditLogger{
			events: make([]JWSAuditEvent, 0),
		},
	}

	// Generate or load RSA key pair
	if err := service.initializeKeys(); err != nil {
		// In production, this should be a fatal error
		// For now, we'll continue with a placeholder
		fmt.Printf("Warning: Failed to initialize JWS keys: %v\n", err)
	}

	return service
}

// initializeKeys initializes the RSA key pair for JWS signing
func (s *JWSAttestationService) initializeKeys() error {
	// In production, load keys from secure storage or environment
	// For now, generate a new key pair
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return fmt.Errorf("failed to generate RSA key: %w", err)
	}

	s.privateKey = privateKey
	s.publicKey = &privateKey.PublicKey

	return nil
}

// GenerateJWS generates a JWS token for a verification response
func (s *JWSAttestationService) GenerateJWS(
	ctx context.Context,
	response *FormattedResponse,
	issuer string,
	audience string,
) (*JWSResult, error) {
	if s.privateKey == nil {
		return nil, fmt.Errorf("JWS private key not initialized")
	}

	// Generate JWS ID
	jwsID := s.generateJWSID()

	// Create claims
	now := time.Now()
	expiresAt := now.Add(24 * time.Hour) // JWS expires in 24 hours

	claims := JWSClaims{
		Verified:        response.Verified,
		Confidence:      response.Confidence,
		Reason:          response.Reason,
		Evidence:        response.Evidence,
		DPID:            response.DPID,
		RequestID:       response.RequestID,
		ProcessingTime:  response.ProcessingTime,
		RequestHash:     response.RequestHash,
		ResponseHash:    response.ResponseHash,
		AttestationType: "verification_result",
		Issuer:          issuer,
		Audience:        audience,
		NotBefore:       now,
		ExpiresAt:       expiresAt,
		IssuedAt:        now,
		JWTID:           jwsID,
	}

	// Create JWT token
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	token.Header["kid"] = s.keyID
	token.Header["typ"] = "JWT"

	// Sign the token
	tokenString, err := token.SignedString(s.privateKey)
	if err != nil {
		s.auditLogger.LogEvent("jws_signing_failed", "JWS signing failed", response.RequestID, jwsID, "error", map[string]string{
			"error": err.Error(),
		})
		return nil, fmt.Errorf("failed to sign JWS token: %w", err)
	}

	// Parse the signed token to extract components
	parsedToken, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		return s.publicKey, nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to parse signed token: %w", err)
	}

	// Extract header and payload
	header := JWSHeader{
		Algorithm: parsedToken.Method.Alg(),
		Type:      "JWT",
		KeyID:     s.keyID,
	}

	payload := JWSPayload{
		Claims: claims,
	}

	// Extract signature
	parts := strings.Split(tokenString, ".")
	if len(parts) != 3 {
		return nil, fmt.Errorf("invalid JWS token format")
	}
	signature := parts[2]

	// Create JWS result
	result := &JWSResult{
		Token:     tokenString,
		Header:    header,
		Payload:   payload,
		Signature: signature,
		JWSID:     jwsID,
		RequestID: response.RequestID,
		CreatedAt: now,
		ExpiresAt: expiresAt,
		Metadata: map[string]string{
			"algorithm":     header.Algorithm,
			"key_id":        header.KeyID,
			"attestation_type": claims.AttestationType,
		},
	}

	// Log successful JWS generation
	s.auditLogger.LogEvent("jws_generated", "JWS token generated successfully", response.RequestID, jwsID, "success", map[string]string{
		"verified":   fmt.Sprintf("%t", response.Verified),
		"confidence": fmt.Sprintf("%.3f", response.Confidence),
		"dp_id":      response.DPID,
	})

	return result, nil
}

// ValidateJWS validates a JWS token
func (s *JWSAttestationService) ValidateJWS(ctx context.Context, tokenString string) (*JWSClaims, error) {
	if s.publicKey == nil {
		return nil, fmt.Errorf("JWS public key not initialized")
	}

	// Parse and validate the token
	token, err := jwt.ParseWithClaims(tokenString, &JWSClaims{}, func(token *jwt.Token) (interface{}, error) {
		// Validate algorithm
		if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}

		// Validate key ID
		if kid, ok := token.Header["kid"].(string); ok {
			if kid != s.keyID {
				return nil, fmt.Errorf("invalid key ID: %s", kid)
			}
		}

		return s.publicKey, nil
	})

	if err != nil {
		s.auditLogger.LogEvent("jws_validation_failed", "JWS validation failed", "", "", "error", map[string]string{
			"error": err.Error(),
		})
		return nil, fmt.Errorf("JWS validation failed: %w", err)
	}

	if !token.Valid {
		return nil, fmt.Errorf("invalid JWS token")
	}

	// Extract claims
	claims, ok := token.Claims.(*JWSClaims)
	if !ok {
		return nil, fmt.Errorf("invalid JWS claims")
	}

	// Log successful validation
	s.auditLogger.LogEvent("jws_validated", "JWS token validated successfully", claims.RequestID, claims.JWTID, "success", map[string]string{
		"verified":   fmt.Sprintf("%t", claims.Verified),
		"confidence": fmt.Sprintf("%.3f", claims.Confidence),
		"dp_id":      claims.DPID,
	})

	return claims, nil
}

// VerifyJWSClaims verifies the claims in a JWS token
func (s *JWSAttestationService) VerifyJWSClaims(claims *JWSClaims) error {
	// Check if token is expired
	if time.Now().After(claims.ExpiresAt) {
		return fmt.Errorf("JWS token is expired")
	}

	// Check if token is not yet valid
	if time.Now().Before(claims.NotBefore) {
		return fmt.Errorf("JWS token is not yet valid")
	}

	// Validate confidence score
	if claims.Confidence < 0.0 || claims.Confidence > 1.0 {
		return fmt.Errorf("invalid confidence score: %f", claims.Confidence)
	}

	// Validate DP ID
	if claims.DPID == "" {
		return fmt.Errorf("missing DP ID in claims")
	}

	// Validate request ID
	if claims.RequestID == "" {
		return fmt.Errorf("missing request ID in claims")
	}

	return nil
}

// HandleJWSSigningError handles JWS signing errors
func (s *JWSAttestationService) HandleJWSSigningError(err error, requestID string) error {
	// Log the signing error
	s.auditLogger.LogEvent("jws_signing_error", "JWS signing error occurred", requestID, "", "error", map[string]string{
		"error": err.Error(),
	})

	// In production, you might want to:
	// 1. Retry with a different key
	// 2. Use a fallback signing method
	// 3. Alert monitoring systems
	// 4. Return a degraded response

	return fmt.Errorf("JWS signing failed: %w", err)
}

// GetPublicKeyPEM returns the public key in PEM format
func (s *JWSAttestationService) GetPublicKeyPEM() (string, error) {
	if s.publicKey == nil {
		return "", fmt.Errorf("public key not initialized")
	}

	// Encode public key to PEM format
	pubKeyBytes, err := x509.MarshalPKIXPublicKey(s.publicKey)
	if err != nil {
		return "", fmt.Errorf("failed to marshal public key: %w", err)
	}

	pubKeyPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PUBLIC KEY",
		Bytes: pubKeyBytes,
	})

	return string(pubKeyPEM), nil
}

// GetJWK returns the public key in JWK format
func (s *JWSAttestationService) GetJWK() (map[string]interface{}, error) {
	if s.publicKey == nil {
		return nil, fmt.Errorf("public key not initialized")
	}

	// Convert RSA public key to JWK format
	nBytes := s.publicKey.N.Bytes()
	eBytes := big.NewInt(int64(s.publicKey.E)).Bytes()

	jwk := map[string]interface{}{
		"kty": "RSA",
		"kid": s.keyID,
		"n":   base64.RawURLEncoding.EncodeToString(nBytes),
		"e":   base64.RawURLEncoding.EncodeToString(eBytes),
		"alg": "RS256",
		"use": "sig",
	}

	return jwk, nil
}

// generateJWSID generates a unique JWS ID
func (s *JWSAttestationService) generateJWSID() string {
	// Generate random bytes for JWS ID
	bytes := make([]byte, 16)
	rand.Read(bytes)
	return fmt.Sprintf("jws_%x", bytes)
}

// LogEvent logs a JWS audit event
func (jal *JWSAuditLogger) LogEvent(eventType, description, requestID, jwsID, status string, metadata map[string]string) {
	jal.mu.Lock()
	defer jal.mu.Unlock()

	event := JWSAuditEvent{
		Timestamp:   time.Now().Format(time.RFC3339),
		EventType:   eventType,
		RequestID:   requestID,
		JWSID:       jwsID,
		Description: description,
		Status:      status,
		Metadata:    metadata,
	}

	jal.events = append(jal.events, event)

	// Keep only the last 1000 events
	if len(jal.events) > 1000 {
		jal.events = jal.events[len(jal.events)-1000:]
	}
}

// GetAuditEvents returns JWS audit events
func (jal *JWSAuditLogger) GetAuditEvents() []JWSAuditEvent {
	jal.mu.Lock()
	defer jal.mu.Unlock()

	events := make([]JWSAuditEvent, len(jal.events))
	copy(events, jal.events)
	return events
}

// GetJWSStats returns JWS attestation statistics
func (s *JWSAttestationService) GetJWSStats() map[string]interface{} {
	events := s.auditLogger.GetAuditEvents()
	
	stats := map[string]interface{}{
		"service_status": "active",
		"key_id":         s.keyID,
		"algorithm":      "RS256",
		"total_events":   len(events),
		"generated_count": 0,
		"validated_count": 0,
		"error_count":     0,
	}

	// Count event types
	for _, event := range events {
		switch event.EventType {
		case "jws_generated":
			stats["generated_count"] = stats["generated_count"].(int) + 1
		case "jws_validated":
			stats["validated_count"] = stats["validated_count"].(int) + 1
		case "jws_signing_failed", "jws_validation_failed", "jws_signing_error":
			stats["error_count"] = stats["error_count"].(int) + 1
		}
	}

	return stats
}

// HealthCheck checks if the JWS attestation service is healthy
func (s *JWSAttestationService) HealthCheck(ctx context.Context) error {
	// Check if keys are initialized
	if s.privateKey == nil {
		return fmt.Errorf("private key not initialized")
	}

	if s.publicKey == nil {
		return fmt.Errorf("public key not initialized")
	}

	// Test JWS generation and validation
	testClaims := JWSClaims{
		Verified:   true,
		Confidence: 0.95,
		DPID:       "dp_test",
		RequestID:  "test_req_123",
		Issuer:     "pavilion-core-broker",
		Audience:   "pavilion-rp",
		NotBefore:  time.Now(),
		ExpiresAt:  time.Now().Add(time.Hour),
		IssuedAt:   time.Now(),
		JWTID:      "test_jws_123",
	}

	// Create test token
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, testClaims)
	token.Header["kid"] = s.keyID
	token.Header["typ"] = "JWT"

	// Sign token
	tokenString, err := token.SignedString(s.privateKey)
	if err != nil {
		return fmt.Errorf("JWS signing test failed: %w", err)
	}

	// Validate token
	_, err = s.ValidateJWS(ctx, tokenString)
	if err != nil {
		return fmt.Errorf("JWS validation test failed: %w", err)
	}

	return nil
} 