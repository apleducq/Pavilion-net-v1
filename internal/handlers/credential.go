package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/pavilion-trust/core-broker/internal/config"
	"github.com/pavilion-trust/core-broker/internal/models"
	"github.com/pavilion-trust/core-broker/internal/services"
)

// CredentialHandler handles credential-related API requests
type CredentialHandler struct {
	config *config.Config
	// Credential signing service for creating and managing credentials
	signingService *services.CredentialSigningService
	// Credential storage (in-memory for MVP, would be database in production)
	credentials map[string]*models.Credential
}

// NewCredentialHandler creates a new credential handler
func NewCredentialHandler(cfg *config.Config, signingService *services.CredentialSigningService) *CredentialHandler {
	return &CredentialHandler{
		config:         cfg,
		signingService: signingService,
		credentials:    make(map[string]*models.Credential),
	}
}

// CreateCredentialRequest represents a request to create a new credential
type CreateCredentialRequest struct {
	Type         string                 `json:"type" validate:"required"`
	Subject      string                 `json:"subject" validate:"required"`
	Claims       map[string]interface{} `json:"claims" validate:"required"`
	SigningMethod string                `json:"signing_method" validate:"required"`
	ExpirationDate string               `json:"expiration_date,omitempty"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
}

// CreateCredentialResponse represents the response from creating a credential
type CreateCredentialResponse struct {
	Credential *models.Credential     `json:"credential"`
	Signature  *services.SigningResult `json:"signature"`
	Status     string                  `json:"status"`
	Message    string                  `json:"message"`
}

// GetCredentialResponse represents the response for getting a credential
type GetCredentialResponse struct {
	Credential *models.Credential     `json:"credential"`
	Signature  *services.SigningResult `json:"signature,omitempty"`
	Status     string                  `json:"status"`
	Message    string                  `json:"message"`
}

// RevokeCredentialRequest represents a request to revoke a credential
type RevokeCredentialRequest struct {
	Reason string `json:"reason" validate:"required"`
}

// RevokeCredentialResponse represents the response from revoking a credential
type RevokeCredentialResponse struct {
	CredentialID string    `json:"credential_id"`
	RevokedAt    time.Time `json:"revoked_at"`
	Reason       string    `json:"reason"`
	Status       string    `json:"status"`
	Message      string    `json:"message"`
}

// CredentialStatusResponse represents the response for checking credential status
type CredentialStatusResponse struct {
	CredentialID string    `json:"credential_id"`
	Status       string    `json:"status"`
	IssuedAt     time.Time `json:"issued_at,omitempty"`
	RevokedAt    time.Time `json:"revoked_at,omitempty"`
	Reason       string    `json:"reason,omitempty"`
	Message      string    `json:"message"`
}

// HandleCreateCredential handles POST /credentials endpoint
func (h *CredentialHandler) HandleCreateCredential(w http.ResponseWriter, r *http.Request) {
	var req CreateCredentialRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate request
	if req.Type == "" || req.Subject == "" || req.SigningMethod == "" {
		http.Error(w, "Missing required fields", http.StatusBadRequest)
		return
	}

	// Create credential
	credential := &models.Credential{
		ID:           fmt.Sprintf("cred-%s", time.Now().Format("20060102150405")),
		Type:         req.Type,
		Issuer:       h.config.Issuer, // Use configured issuer
		Subject:      req.Subject,
		IssuanceDate: time.Now().Format(time.RFC3339),
		ExpirationDate: req.ExpirationDate,
		Version:      "1.0",
		Claims:       req.Claims,
		Status:       "valid",
		Metadata:     req.Metadata,
		Proof: models.CredentialProof{
			Type:               "JwtProof2020",
			Created:            time.Now().Format(time.RFC3339),
			VerificationMethod: fmt.Sprintf("%s#%s", h.config.Issuer, "key-1"),
			ProofPurpose:       "assertionMethod",
		},
	}

	// Sign the credential
	signingMethod := services.SigningMethod(req.SigningMethod)
	signature, err := h.signingService.SignCredential(credential, signingMethod)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to sign credential: %v", err), http.StatusInternalServerError)
		return
	}

	// Store the credential
	h.credentials[credential.ID] = credential

	// Return response
	response := CreateCredentialResponse{
		Credential: credential,
		Signature:  signature,
		Status:     "success",
		Message:    "Credential created successfully",
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(response)
}

// HandleGetCredential handles GET /credentials/{id} endpoint
func (h *CredentialHandler) HandleGetCredential(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	credentialID := vars["id"]

	credential, exists := h.credentials[credentialID]
	if !exists {
		http.Error(w, "Credential not found", http.StatusNotFound)
		return
	}

	response := GetCredentialResponse{
		Credential: credential,
		Status:     "success",
		Message:    "Credential retrieved successfully",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// HandleListCredentials handles GET /credentials endpoint
func (h *CredentialHandler) HandleListCredentials(w http.ResponseWriter, r *http.Request) {
	credentials := make([]*models.Credential, 0, len(h.credentials))
	for _, cred := range h.credentials {
		credentials = append(credentials, cred)
	}

	response := map[string]interface{}{
		"credentials": credentials,
		"count":       len(credentials),
		"status":      "success",
		"message":     "Credentials retrieved successfully",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// HandleRevokeCredential handles POST /credentials/{id}/revoke endpoint
func (h *CredentialHandler) HandleRevokeCredential(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	credentialID := vars["id"]

	credential, exists := h.credentials[credentialID]
	if !exists {
		http.Error(w, "Credential not found", http.StatusNotFound)
		return
	}

	var req RevokeCredentialRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Update credential status
	credential.Status = "revoked"
	credential.Metadata["revoked_at"] = time.Now().Format(time.RFC3339)
	credential.Metadata["revocation_reason"] = req.Reason

	response := RevokeCredentialResponse{
		CredentialID: credentialID,
		RevokedAt:    time.Now(),
		Reason:       req.Reason,
		Status:       "success",
		Message:      "Credential revoked successfully",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// HandleGetCredentialStatus handles GET /credentials/{id}/status endpoint
func (h *CredentialHandler) HandleGetCredentialStatus(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	credentialID := vars["id"]

	credential, exists := h.credentials[credentialID]
	if !exists {
		http.Error(w, "Credential not found", http.StatusNotFound)
		return
	}

	// Parse issuance date
	issuedAt, _ := time.Parse(time.RFC3339, credential.IssuanceDate)
	
	response := CredentialStatusResponse{
		CredentialID: credentialID,
		Status:       credential.Status,
		IssuedAt:     issuedAt,
		Message:      "Credential status retrieved successfully",
	}

	// Add revocation info if revoked
	if credential.Status == "revoked" {
		if revokedAtStr, ok := credential.Metadata["revoked_at"].(string); ok {
			if revokedAt, err := time.Parse(time.RFC3339, revokedAtStr); err == nil {
				response.RevokedAt = revokedAt
			}
		}
		if reason, ok := credential.Metadata["revocation_reason"].(string); ok {
			response.Reason = reason
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// HandleVerifyCredential handles POST /credentials/{id}/verify endpoint
func (h *CredentialHandler) HandleVerifyCredential(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	credentialID := vars["id"]

	credential, exists := h.credentials[credentialID]
	if !exists {
		http.Error(w, "Credential not found", http.StatusNotFound)
		return
	}

	// For MVP, we'll do basic validation
	// In production, you would verify the signature and check against a blockchain
	valid := credential.Status == "valid"

	response := map[string]interface{}{
		"credential_id": credentialID,
		"valid":         valid,
		"status":        credential.Status,
		"message":       "Credential verification completed",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
} 