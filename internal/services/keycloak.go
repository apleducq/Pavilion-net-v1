package services

import (
	"context"
	"crypto/rsa"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/pavilion-trust/core-broker/internal/config"
)

// KeycloakService handles JWT validation and user authentication
type KeycloakService struct {
	config     *config.Config
	client     *http.Client
	publicKeys map[string]*rsa.PublicKey
	lastFetch  time.Time
}

// UserInfo represents the authenticated user information
type UserInfo struct {
	Subject    string            `json:"sub"`
	Realm      string            `json:"realm_access"`
	ResourceID string            `json:"azp"`
	Email      string            `json:"email"`
	Roles      []string          `json:"realm_access.roles"`
	Claims     map[string]string `json:"-"`
}

// NewKeycloakService creates a new Keycloak service
func NewKeycloakService(cfg *config.Config) *KeycloakService {
	return &KeycloakService{
		config: cfg,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
		publicKeys: make(map[string]*rsa.PublicKey),
	}
}

// ValidateToken validates a JWT token and returns user information
func (s *KeycloakService) ValidateToken(ctx context.Context, tokenString string) (*UserInfo, error) {
	// Parse the token without validation first to get the key ID
	token, err := jwt.Parse(tokenString, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to parse token: %w", err)
	}

	// Extract key ID from token header
	keyID, ok := token.Header["kid"].(string)
	if !ok {
		return nil, fmt.Errorf("missing key ID in token header")
	}

	// Fetch public keys if needed
	if err := s.fetchPublicKeys(ctx); err != nil {
		return nil, fmt.Errorf("failed to fetch public keys: %w", err)
	}

	// Get the public key for this token
	publicKey, exists := s.publicKeys[keyID]
	if !exists {
		return nil, fmt.Errorf("unknown key ID: %s", keyID)
	}

	// Parse and validate the token
	parsedToken, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		// Validate the signing method
		if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return publicKey, nil
	})

	if err != nil {
		return nil, fmt.Errorf("token validation failed: %w", err)
	}

	if !parsedToken.Valid {
		return nil, fmt.Errorf("invalid token")
	}

	// Extract claims
	claims, ok := parsedToken.Claims.(jwt.MapClaims)
	if !ok {
		return nil, fmt.Errorf("invalid token claims")
	}

	// Validate required claims
	if err := s.validateClaims(claims); err != nil {
		return nil, fmt.Errorf("claim validation failed: %w", err)
	}

	// Extract user information
	userInfo, err := s.extractUserInfo(claims)
	if err != nil {
		return nil, fmt.Errorf("failed to extract user info: %w", err)
	}

	return userInfo, nil
}

// fetchPublicKeys fetches public keys from Keycloak
func (s *KeycloakService) fetchPublicKeys(ctx context.Context) error {
	// Only fetch if we haven't fetched recently (cache for 1 hour)
	if time.Since(s.lastFetch) < time.Hour {
		return nil
	}

	// Fetch JWKS (JSON Web Key Set) from Keycloak
	jwksURL := fmt.Sprintf("%s/realms/%s/protocol/openid-connect/certs",
		s.config.KeycloakURL, s.config.KeycloakRealm)

	req, err := http.NewRequestWithContext(ctx, "GET", jwksURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to fetch JWKS: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("JWKS request failed with status: %d", resp.StatusCode)
	}

	// Parse JWKS response
	var jwks struct {
		Keys []struct {
			Kid string `json:"kid"`
			Kty string `json:"kty"`
			Alg string `json:"alg"`
			Use string `json:"use"`
			N   string `json:"n"`
			E   string `json:"e"`
		} `json:"keys"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&jwks); err != nil {
		return fmt.Errorf("failed to decode JWKS: %w", err)
	}

	// Convert JWKS to public keys
	for _, key := range jwks.Keys {
		if key.Use == "sig" && key.Kty == "RSA" {
			publicKey, err := s.convertJWKToPublicKey()
			if err != nil {
				continue // Skip invalid keys
			}
			s.publicKeys[key.Kid] = publicKey
		}
	}

	s.lastFetch = time.Now()
	return nil
}

// convertJWKToPublicKey converts JWK to RSA public key
func (s *KeycloakService) convertJWKToPublicKey() (*rsa.PublicKey, error) {
	// This is a simplified implementation
	// In production, you'd want to use a proper JWK library
	// For now, we'll use a mock implementation
	return &rsa.PublicKey{
		N: nil,   // Would be decoded from base64url
		E: 65537, // Default exponent
	}, nil
}

// validateClaims validates required JWT claims
func (s *KeycloakService) validateClaims(claims jwt.MapClaims) error {
	// Check if token is expired
	if exp, ok := claims["exp"].(float64); ok {
		if time.Unix(int64(exp), 0).Before(time.Now()) {
			return fmt.Errorf("token expired")
		}
	}

	// Check if token is not yet valid
	if nbf, ok := claims["nbf"].(float64); ok {
		if time.Unix(int64(nbf), 0).After(time.Now()) {
			return fmt.Errorf("token not yet valid")
		}
	}

	// Check issuer
	if iss, ok := claims["iss"].(string); ok {
		expectedIssuer := fmt.Sprintf("%s/realms/%s", s.config.KeycloakURL, s.config.KeycloakRealm)
		if iss != expectedIssuer {
			return fmt.Errorf("invalid issuer: %s", iss)
		}
	}

	// Check audience
	if aud, ok := claims["aud"].(string); ok {
		if aud != "pavilion-broker" {
			return fmt.Errorf("invalid audience: %s", aud)
		}
	}

	return nil
}

// extractUserInfo extracts user information from JWT claims
func (s *KeycloakService) extractUserInfo(claims jwt.MapClaims) (*UserInfo, error) {
	userInfo := &UserInfo{
		Claims: make(map[string]string),
	}

	// Extract subject
	if sub, ok := claims["sub"].(string); ok {
		userInfo.Subject = sub
	}

	// Extract resource ID (client ID)
	if azp, ok := claims["azp"].(string); ok {
		userInfo.ResourceID = azp
	}

	// Extract email
	if email, ok := claims["email"].(string); ok {
		userInfo.Email = email
	}

	// Extract realm access
	if realmAccess, ok := claims["realm_access"].(map[string]interface{}); ok {
		if roles, ok := realmAccess["roles"].([]interface{}); ok {
			for _, role := range roles {
				if roleStr, ok := role.(string); ok {
					userInfo.Roles = append(userInfo.Roles, roleStr)
				}
			}
		}
	}

	// Store all claims for potential future use
	for key, value := range claims {
		if str, ok := value.(string); ok {
			userInfo.Claims[key] = str
		}
	}

	return userInfo, nil
}

// HasRole checks if the user has a specific role
func (u *UserInfo) HasRole(role string) bool {
	for _, userRole := range u.Roles {
		if userRole == role {
			return true
		}
	}
	return false
}

// HasAnyRole checks if the user has any of the specified roles
func (u *UserInfo) HasAnyRole(roles ...string) bool {
	for _, requiredRole := range roles {
		if u.HasRole(requiredRole) {
			return true
		}
	}
	return false
}

// HealthCheck checks if the Keycloak service is healthy
func (s *KeycloakService) HealthCheck(ctx context.Context) error {
	// Try to fetch public keys as a health check
	return s.fetchPublicKeys(ctx)
}
