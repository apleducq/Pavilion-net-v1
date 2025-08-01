package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/pavilion-trust/core-broker/internal/config"
	"github.com/pavilion-trust/core-broker/internal/models"
	"github.com/pavilion-trust/core-broker/internal/services"
)

// VerificationHandler handles verification requests
type VerificationHandler struct {
	config *config.Config
	policyService *services.PolicyService
	privacyService *services.PrivacyService
	dpService *services.DPConnectorService
	auditService *services.AuditService
	cacheService *services.CacheService
}

// NewVerificationHandler creates a new verification handler
func NewVerificationHandler(cfg *config.Config) *VerificationHandler {
	return &VerificationHandler{
		config: cfg,
		policyService: services.NewPolicyService(cfg),
		privacyService: services.NewPrivacyService(cfg),
		dpService: services.NewDPConnectorService(cfg),
		auditService: services.NewAuditService(cfg),
		cacheService: services.NewCacheService(cfg),
	}
}

// HandleVerification processes verification requests
func (h *VerificationHandler) HandleVerification(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	
	// Parse request
	var req models.VerificationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, "INVALID_REQUEST", "Failed to parse request body", http.StatusBadRequest)
		return
	}
	
	// Validate request
	if err := req.Validate(); err != nil {
		writeError(w, "VALIDATION_ERROR", err.Error(), http.StatusBadRequest)
		return
	}
	
	// Get request ID from context
	requestID := getRequestID(ctx)
	
	// Check cache first
	if cachedResult := h.cacheService.GetVerificationResult(req); cachedResult != nil {
		h.auditService.LogVerification(ctx, req, cachedResult, "CACHE_HIT")
		writeResponse(w, cachedResult)
		return
	}
	
	// Enforce policy
	if err := h.policyService.EnforcePolicy(ctx, req); err != nil {
		h.auditService.LogVerification(ctx, req, nil, "POLICY_DENIED")
		writeError(w, "POLICY_VIOLATION", err.Error(), http.StatusForbidden)
		return
	}
	
	// Apply privacy-preserving transformations
	privacyReq, err := h.privacyService.TransformRequest(ctx, req)
	if err != nil {
		h.auditService.LogVerification(ctx, req, nil, "PRIVACY_ERROR")
		writeError(w, "PRIVACY_ERROR", "Failed to apply privacy transformations", http.StatusInternalServerError)
		return
	}
	
	// Communicate with DP Connector
	dpResponse, err := h.dpService.VerifyWithDP(ctx, privacyReq)
	if err != nil {
		h.auditService.LogVerification(ctx, req, nil, "DP_ERROR")
		writeError(w, "DP_ERROR", "Failed to communicate with data provider", http.StatusServiceUnavailable)
		return
	}
	
	// Generate response
	response := h.generateResponse(req, dpResponse, requestID)
	
	// Cache successful result
	if response.Status == "verified" {
		h.cacheService.CacheVerificationResult(req, response)
	}
	
	// Log audit entry
	h.auditService.LogVerification(ctx, req, response, "SUCCESS")
	
	// Return response
	writeResponse(w, response)
}

// generateResponse creates a verification response
func (h *VerificationHandler) generateResponse(req models.VerificationRequest, dpResponse *models.DPResponse, requestID string) *models.VerificationResponse {
	response := &models.VerificationResponse{
		VerificationID: requestID,
		Status:         dpResponse.Status,
		ConfidenceScore: dpResponse.ConfidenceScore,
		Timestamp:      time.Now().Format(time.RFC3339),
		ExpiresAt:      time.Now().Add(h.config.CacheTTL).Format(time.RFC3339),
		RequestID:      requestID,
	}
	
	// TODO: Add JWS attestation (T-014)
	// TODO: Add audit references (T-015)
	
	return response
}

// getRequestID extracts request ID from context
func getRequestID(ctx context.Context) string {
	if requestID, ok := ctx.Value("request_id").(string); ok {
		return requestID
	}
	return "unknown"
}

// writeResponse writes a JSON response
func writeResponse(w http.ResponseWriter, response *models.VerificationResponse) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// writeError writes a structured error response
func writeError(w http.ResponseWriter, code, message string, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	
	response := map[string]interface{}{
		"error": map[string]interface{}{
			"code":    code,
			"message": message,
			"timestamp": time.Now().Format(time.RFC3339),
		},
	}
	
	json.NewEncoder(w).Encode(response)
} 