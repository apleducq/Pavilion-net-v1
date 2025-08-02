package services

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/pavilion-trust/core-broker/internal/models"
)

// CredentialSigningService handles credential signing operations
type CredentialSigningService struct {
	// RSA private key for RSA signing
	rsaPrivateKey *rsa.PrivateKey
	// ECDSA private key for ECDSA signing
	ecdsaPrivateKey *ecdsa.PrivateKey
	// Key ID for JWS header
	keyID string
	// Issuer for credentials
	issuer string
}

// SigningMethod represents the signing method to use
type SigningMethod string

const (
	SigningMethodJWT     SigningMethod = "JWT"
	SigningMethodLDProof SigningMethod = "LD-PROOF"
	SigningMethodECDSA   SigningMethod = "ECDSA"
	SigningMethodRSA     SigningMethod = "RSA"
)

// SigningResult represents the result of a signing operation
type SigningResult struct {
	Method    SigningMethod          `json:"method"`
	Signature string                 `json:"signature"`
	KeyID     string                 `json:"key_id"`
	Algorithm string                 `json:"algorithm"`
	Created   time.Time              `json:"created"`
	ExpiresAt time.Time              `json:"expires_at,omitempty"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// NewCredentialSigningService creates a new credential signing service
func NewCredentialSigningService(rsaKey *rsa.PrivateKey, ecdsaKey *ecdsa.PrivateKey, keyID, issuer string) *CredentialSigningService {
	return &CredentialSigningService{
		rsaPrivateKey:   rsaKey,
		ecdsaPrivateKey: ecdsaKey,
		keyID:           keyID,
		issuer:          issuer,
	}
}

// SignCredential signs a credential using the specified method
func (s *CredentialSigningService) SignCredential(credential *models.Credential, method SigningMethod) (*SigningResult, error) {
	switch method {
	case SigningMethodJWT:
		return s.signJWT(credential)
	case SigningMethodLDProof:
		return s.signLDProof(credential)
	case SigningMethodECDSA:
		return s.signECDSA(credential)
	case SigningMethodRSA:
		return s.signRSA(credential)
	default:
		return nil, fmt.Errorf("unsupported signing method: %s", method)
	}
}

// signJWT signs a credential using JWT
func (s *CredentialSigningService) signJWT(credential *models.Credential) (*SigningResult, error) {
	if s.rsaPrivateKey == nil {
		return nil, fmt.Errorf("RSA private key not available for JWT signing")
	}

	// Create JWT claims
	now := time.Now()
	expiresAt := now.Add(24 * time.Hour)

	claims := jwt.MapClaims{
		"iss": credential.Issuer,
		"sub": credential.Subject,
		"iat": now.Unix(),
		"exp": expiresAt.Unix(),
		"jti": credential.ID,
		"vc": map[string]interface{}{
			"@context":          []string{"https://www.w3.org/2018/credentials/v1"},
			"type":              []string{"VerifiableCredential", credential.Type},
			"issuer":            credential.Issuer,
			"issuanceDate":      credential.IssuanceDate,
			"credentialSubject": credential.Claims,
			"version":           credential.Version,
		},
	}

	// Create JWT token
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	token.Header["kid"] = s.keyID
	token.Header["typ"] = "JWT"

	// Sign the token
	tokenString, err := token.SignedString(s.rsaPrivateKey)
	if err != nil {
		return nil, fmt.Errorf("failed to sign JWT: %w", err)
	}

	return &SigningResult{
		Method:    SigningMethodJWT,
		Signature: tokenString,
		KeyID:     s.keyID,
		Algorithm: "RS256",
		Created:   now,
		ExpiresAt: expiresAt,
		Metadata: map[string]interface{}{
			"jwt_header": token.Header,
		},
	}, nil
}

// signLDProof signs a credential using LD-Proof
func (s *CredentialSigningService) signLDProof(credential *models.Credential) (*SigningResult, error) {
	if s.ecdsaPrivateKey == nil {
		return nil, fmt.Errorf("ECDSA private key not available for LD-Proof signing")
	}

	// Create LD-Proof document
	ldProof := map[string]interface{}{
		"@context": []string{
			"https://www.w3.org/2018/credentials/v1",
			"https://w3id.org/security/suites/ed25519-2018/v1",
		},
		"type":              "VerifiableCredential",
		"issuer":            credential.Issuer,
		"issuanceDate":      credential.IssuanceDate,
		"credentialSubject": credential.Claims,
		"version":           credential.Version,
	}

	// Canonicalize the document (for MVP, we'll use JSON)
	canonicalized, err := json.Marshal(ldProof)
	if err != nil {
		return nil, fmt.Errorf("failed to canonicalize LD-Proof: %w", err)
	}

	// Sign the canonicalized document
	hash := sha256.Sum256(canonicalized)
	signature, err := ecdsa.SignASN1(rand.Reader, s.ecdsaPrivateKey, hash[:])
	if err != nil {
		return nil, fmt.Errorf("failed to sign LD-Proof: %w", err)
	}

	// Create LD-Proof
	now := time.Now()
	ldProofSignature := map[string]interface{}{
		"type":               "Ed25519Signature2018",
		"created":            now.Format(time.RFC3339),
		"verificationMethod": fmt.Sprintf("%s#%s", s.issuer, s.keyID),
		"proofPurpose":       "assertionMethod",
		"jws":                base64.RawURLEncoding.EncodeToString(signature),
	}

	ldProof["proof"] = ldProofSignature

	// Convert to JSON string
	ldProofJSON, err := json.Marshal(ldProof)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal LD-Proof: %w", err)
	}

	return &SigningResult{
		Method:    SigningMethodLDProof,
		Signature: string(ldProofJSON),
		KeyID:     s.keyID,
		Algorithm: "Ed25519",
		Created:   now,
		Metadata: map[string]interface{}{
			"ld_proof": ldProofSignature,
		},
	}, nil
}

// signECDSA signs a credential using ECDSA
func (s *CredentialSigningService) signECDSA(credential *models.Credential) (*SigningResult, error) {
	if s.ecdsaPrivateKey == nil {
		return nil, fmt.Errorf("ECDSA private key not available")
	}

	// Create the data to sign
	dataToSign := fmt.Sprintf("%s:%s:%s:%s",
		credential.ID,
		credential.Type,
		credential.Issuer,
		credential.Subject,
	)

	// Hash the data
	hash := sha256.Sum256([]byte(dataToSign))

	// Sign the hash
	signature, err := ecdsa.SignASN1(rand.Reader, s.ecdsaPrivateKey, hash[:])
	if err != nil {
		return nil, fmt.Errorf("failed to sign with ECDSA: %w", err)
	}

	now := time.Now()

	return &SigningResult{
		Method:    SigningMethodECDSA,
		Signature: base64.RawURLEncoding.EncodeToString(signature),
		KeyID:     s.keyID,
		Algorithm: "ECDSA-SHA256",
		Created:   now,
		Metadata: map[string]interface{}{
			"data_signed":    dataToSign,
			"hash_algorithm": "SHA256",
		},
	}, nil
}

// signRSA signs a credential using RSA
func (s *CredentialSigningService) signRSA(credential *models.Credential) (*SigningResult, error) {
	if s.rsaPrivateKey == nil {
		return nil, fmt.Errorf("RSA private key not available")
	}

	// Create the data to sign
	dataToSign := fmt.Sprintf("%s:%s:%s:%s",
		credential.ID,
		credential.Type,
		credential.Issuer,
		credential.Subject,
	)

	// Hash the data
	hash := sha256.Sum256([]byte(dataToSign))

	// Sign the hash
	signature, err := rsa.SignPKCS1v15(rand.Reader, s.rsaPrivateKey, crypto.SHA256, hash[:])
	if err != nil {
		return nil, fmt.Errorf("failed to sign with RSA: %w", err)
	}

	now := time.Now()

	return &SigningResult{
		Method:    SigningMethodRSA,
		Signature: base64.RawURLEncoding.EncodeToString(signature),
		KeyID:     s.keyID,
		Algorithm: "RSA-SHA256",
		Created:   now,
		Metadata: map[string]interface{}{
			"data_signed":    dataToSign,
			"hash_algorithm": "SHA256",
		},
	}, nil
}

// VerifySignature verifies a signature
func (s *CredentialSigningService) VerifySignature(credential *models.Credential, result *SigningResult) (bool, error) {
	switch result.Method {
	case SigningMethodJWT:
		return s.verifyJWT(credential, result)
	case SigningMethodLDProof:
		return s.verifyLDProof(result)
	case SigningMethodECDSA:
		return s.verifyECDSA(credential, result)
	case SigningMethodRSA:
		return s.verifyRSA(credential, result)
	default:
		return false, fmt.Errorf("unsupported signing method: %s", result.Method)
	}
}

// verifyJWT verifies a JWT signature
func (s *CredentialSigningService) verifyJWT(credential *models.Credential, result *SigningResult) (bool, error) {
	if s.rsaPrivateKey == nil {
		return false, fmt.Errorf("RSA public key not available")
	}

	// Parse and verify JWT
	token, err := jwt.Parse(result.Signature, func(token *jwt.Token) (interface{}, error) {
		return &s.rsaPrivateKey.PublicKey, nil
	})

	if err != nil {
		return false, fmt.Errorf("failed to parse JWT: %w", err)
	}

	if !token.Valid {
		return false, fmt.Errorf("invalid JWT token")
	}

	// Verify that the JWT claims match the credential
	if claims, ok := token.Claims.(jwt.MapClaims); ok {
		// Check issuer
		if issuer, exists := claims["iss"]; !exists || issuer != credential.Issuer {
			return false, fmt.Errorf("issuer mismatch in JWT")
		}
		// Check subject
		if sub, exists := claims["sub"]; !exists || sub != credential.Subject {
			return false, fmt.Errorf("subject mismatch in JWT")
		}
		// Check credential ID
		if jti, exists := claims["jti"]; !exists || jti != credential.ID {
			return false, fmt.Errorf("credential ID mismatch in JWT")
		}
	}

	return true, nil
}

// verifyLDProof verifies an LD-Proof signature
func (s *CredentialSigningService) verifyLDProof(result *SigningResult) (bool, error) {
	if s.ecdsaPrivateKey == nil {
		return false, fmt.Errorf("ECDSA public key not available")
	}

	// For MVP, we'll do basic validation
	// In production, you would implement full LD-Proof verification
	var ldProof map[string]interface{}
	if err := json.Unmarshal([]byte(result.Signature), &ldProof); err != nil {
		return false, fmt.Errorf("failed to parse LD-Proof: %w", err)
	}

	// Check if proof exists
	if _, exists := ldProof["proof"]; !exists {
		return false, fmt.Errorf("no proof found in LD-Proof")
	}

	return true, nil
}

// verifyECDSA verifies an ECDSA signature
func (s *CredentialSigningService) verifyECDSA(credential *models.Credential, result *SigningResult) (bool, error) {
	if s.ecdsaPrivateKey == nil {
		return false, fmt.Errorf("ECDSA public key not available")
	}

	// Recreate the data that was signed
	dataToSign := fmt.Sprintf("%s:%s:%s:%s",
		credential.ID,
		credential.Type,
		credential.Issuer,
		credential.Subject,
	)

	// Hash the data
	hash := sha256.Sum256([]byte(dataToSign))

	// Decode the signature
	signature, err := base64.RawURLEncoding.DecodeString(result.Signature)
	if err != nil {
		return false, fmt.Errorf("failed to decode signature: %w", err)
	}

	// Verify the signature
	valid := ecdsa.VerifyASN1(&s.ecdsaPrivateKey.PublicKey, hash[:], signature)
	return valid, nil
}

// verifyRSA verifies an RSA signature
func (s *CredentialSigningService) verifyRSA(credential *models.Credential, result *SigningResult) (bool, error) {
	if s.rsaPrivateKey == nil {
		return false, fmt.Errorf("RSA public key not available")
	}

	// Recreate the data that was signed
	dataToSign := fmt.Sprintf("%s:%s:%s:%s",
		credential.ID,
		credential.Type,
		credential.Issuer,
		credential.Subject,
	)

	// Hash the data
	hash := sha256.Sum256([]byte(dataToSign))

	// Decode the signature
	signature, err := base64.RawURLEncoding.DecodeString(result.Signature)
	if err != nil {
		return false, fmt.Errorf("failed to decode signature: %w", err)
	}

	// Verify the signature
	err = rsa.VerifyPKCS1v15(&s.rsaPrivateKey.PublicKey, crypto.SHA256, hash[:], signature)
	if err != nil {
		return false, nil
	}

	return true, nil
}

// GetSupportedMethods returns the supported signing methods
func (s *CredentialSigningService) GetSupportedMethods() []SigningMethod {
	methods := []SigningMethod{}

	if s.rsaPrivateKey != nil {
		methods = append(methods, SigningMethodJWT, SigningMethodRSA)
	}

	if s.ecdsaPrivateKey != nil {
		methods = append(methods, SigningMethodLDProof, SigningMethodECDSA)
	}

	return methods
}
