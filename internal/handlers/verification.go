package handlers

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/pavilion-trust/core-broker/internal/config"
	"github.com/pavilion-trust/core-broker/internal/models"
	"github.com/pavilion-trust/core-broker/internal/services"
)

// VerificationHandler handles verification requests
type VerificationHandler struct {
	config *config.Config
	authorizationService *services.AuthorizationService
	policyService *services.PolicyService
	privacyService *services.PrivacyService
	dpService *services.DPConnectorService
	auditService *services.AuditService
	cacheService *services.CacheService
}

// NewVerificationHandler creates a new verification handler
func NewVerificationHandler(cfg *config.Config) *VerificationHandler {
	policyService := services.NewPolicyService(cfg)
	return &VerificationHandler{
		config: cfg,
		authorizationService: services.NewAuthorizationService(cfg, policyService),
		policyService: policyService,
		privacyService: services.NewPrivacyService(cfg),
		dpService: services.NewDPConnectorService(cfg),
		auditService: services.NewAuditService(cfg),
		cacheService: services.NewCacheService(cfg),
	}
}

// HandleVerification processes verification requests
func (h *VerificationHandler) HandleVerification(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	
	// Get validated request from context (set by validation middleware)
	req := getValidatedRequestFromContext(ctx)
	if req == nil {
		writeError(w, "INVALID_REQUEST", "Request validation failed", http.StatusBadRequest)
		return
	}
	
	// Get request ID from context
	requestID := getRequestID(ctx)
	
	// Check cache first
	if cachedResult := h.cacheService.GetVerificationResult(*req); cachedResult != nil {
		h.auditService.LogVerification(ctx, *req, cachedResult, "CACHE_HIT")
		writeResponse(w, cachedResult)
		return
	}
	
	// Perform authorization checks
	authDecision, err := h.authorizationService.AuthorizeRequest(ctx, *req)
	if err != nil {
		h.auditService.LogVerification(ctx, *req, nil, "AUTHORIZATION_ERROR")
		writeError(w, "AUTHORIZATION_ERROR", "Authorization service error", http.StatusInternalServerError)
		return
	}
	
	if !authDecision.Allowed {
		h.auditService.LogVerification(ctx, *req, nil, "AUTHORIZATION_DENIED")
		writeError(w, "AUTHORIZATION_DENIED", authDecision.Reason, http.StatusForbidden)
		return
	}
	
	// Apply privacy-preserving transformations
	privacyReq, err := h.privacyService.TransformRequest(ctx, *req)
	if err != nil {
		h.auditService.LogVerification(ctx, *req, nil, "PRIVACY_ERROR")
		writeError(w, "PRIVACY_ERROR", "Failed to apply privacy transformations", http.StatusInternalServerError)
		return
	}
	
	// Communicate with DP Connector
	dpResponse, err := h.dpService.VerifyWithDP(ctx, privacyReq)
	if err != nil {
		h.auditService.LogVerification(ctx, *req, nil, "DP_ERROR")
		writeError(w, "DP_ERROR", "Failed to communicate with data provider", http.StatusServiceUnavailable)
		return
	}
	
	// Generate response
	response := h.generateResponse(*req, dpResponse, requestID)
	
	// Cache successful result
	if response.Status == "verified" {
		h.cacheService.CacheVerificationResult(*req, response)
	}
	
	// Log audit entry
	h.auditService.LogVerification(ctx, *req, response, "SUCCESS")
	
	// Return response
	writeResponse(w, response)
}

// generateResponse creates a verification response
func (h *VerificationHandler) generateResponse(req models.VerificationRequest, dpResponse *models.DPResponse, requestID string) *models.VerificationResponse {
	response := models.NewVerificationResponse(dpResponse.Status, dpResponse.ConfidenceScore, requestID)
	
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
	errorResponse := models.NewErrorResponse(code, message, "unknown")
	
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	
	if data, err := errorResponse.ToJSON(); err == nil {
		w.Write(data)
	} else {
		// Fallback error response
		w.Write([]byte(`{"error":{"code":"INTERNAL_ERROR","message":"Failed to format error response"}}`))
	}
}

// getValidatedRequestFromContext retrieves the validated request from context
func getValidatedRequestFromContext(ctx context.Context) *models.VerificationRequest {
	if req, ok := ctx.Value("validated_request").(*models.VerificationRequest); ok {
		return req
	}
	return nil
} 