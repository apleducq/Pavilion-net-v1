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
	authorizationService *services.AuthorizationService
	policyService *services.PolicyService
	privacyService *services.PrivacyService
	dpService *services.DPConnectorService
	pullJobService *services.PullJobService
	responseParserService *services.ResponseParserService
	responseFormatterService *services.ResponseFormatterService
	jwsAttestationService *services.JWSAttestationService
	auditService *services.AuditService
	cacheService *services.CacheService
}

// NewVerificationHandler creates a new verification handler
func NewVerificationHandler(cfg *config.Config) *VerificationHandler {
	policyService := services.NewPolicyService(cfg)
	dpService := services.NewDPConnectorService(cfg)
	
	return &VerificationHandler{
		config: cfg,
		authorizationService: services.NewAuthorizationService(cfg, policyService),
		policyService: policyService,
		privacyService: services.NewPrivacyService(cfg),
		dpService: dpService,
		pullJobService: services.NewPullJobService(cfg, dpService),
		responseParserService: services.NewResponseParserService(cfg),
		responseFormatterService: services.NewResponseFormatterService(cfg),
		jwsAttestationService: services.NewJWSAttestationService(cfg),
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
		auditRef := h.auditService.LogVerification(ctx, *req, cachedResult, "CACHE_HIT")
		// Add audit reference to cached result
		if auditRef != nil {
			cachedResult.AuditReference = auditRef.AuditEntryID
		}
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
	
	// Submit pull-job request (T-011)
	jobStatus, err := h.pullJobService.SubmitJob(ctx, privacyReq)
	if err != nil {
		h.auditService.LogVerification(ctx, *req, nil, "JOB_SUBMISSION_ERROR")
		writeError(w, "JOB_SUBMISSION_ERROR", "Failed to submit verification job", http.StatusInternalServerError)
		return
	}
	
	// Wait for job completion (with timeout)
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	
	// Poll for job completion
	var dpResponse *services.DPResponse
	for {
		select {
		case <-ctx.Done():
			h.auditService.LogVerification(ctx, *req, nil, "JOB_TIMEOUT")
			writeError(w, "JOB_TIMEOUT", "Verification job timed out", http.StatusRequestTimeout)
			return
		default:
			// Check job status
			updatedJobStatus, err := h.pullJobService.GetJobStatus(jobStatus.JobID)
			if err != nil {
				h.auditService.LogVerification(ctx, *req, nil, "JOB_STATUS_ERROR")
				writeError(w, "JOB_STATUS_ERROR", "Failed to get job status", http.StatusInternalServerError)
				return
			}
			
			if updatedJobStatus.Status == services.JobCompleted {
				// Parse DP response (T-012)
				parsedResponse, err := h.responseParserService.ParseAndValidateResponse(updatedJobStatus.Result)
				if err != nil {
					h.auditService.LogVerification(ctx, *req, nil, "RESPONSE_PARSE_ERROR")
					writeError(w, "RESPONSE_PARSE_ERROR", "Failed to parse response", http.StatusInternalServerError)
					return
				}
				
				// Convert to DPResponse for compatibility
				dpResponse = h.responseParserService.ConvertToDPResponse(parsedResponse)
				break
			} else if updatedJobStatus.Status == services.JobFailed {
				h.auditService.LogVerification(ctx, *req, nil, "JOB_FAILED")
				writeError(w, "JOB_FAILED", "Verification job failed", http.StatusInternalServerError)
				return
			}
			
			// Wait before polling again
			time.Sleep(100 * time.Millisecond)
		}
	}
	
	// Generate formatted response (T-013)
	response := h.generateFormattedResponse(*req, dpResponse, requestID, ctx)
	
	// Add audit reference to response (T-015)
	auditRef := h.auditService.LogVerification(ctx, *req, response, "SUCCESS")
	if auditRef != nil {
		response.AuditReference = auditRef.AuditEntryID
		// Add audit metadata
		if response.Metadata == nil {
			response.Metadata = make(map[string]interface{})
		}
		response.Metadata["audit_merkle_proof"] = auditRef.MerkleProof
		response.Metadata["audit_timestamp"] = auditRef.Timestamp
		response.Metadata["audit_hash"] = auditRef.Hash
	}
	
	// Cache successful result
	h.cacheService.CacheVerificationResult(*req, response)
	
	// Return response
	writeResponse(w, response)
}

// generateFormattedResponse creates a formatted verification response using T-013 and T-014
func (h *VerificationHandler) generateFormattedResponse(req models.VerificationRequest, dpResponse *services.DPResponse, requestID string, ctx context.Context) *models.VerificationResponse {
	// Parse DP response (T-012)
	parsedResponse, err := h.responseParserService.ParseAndValidateResponse(dpResponse)
	if err != nil {
		// Handle parsing error by creating error response
		errorResponse := h.responseFormatterService.FormatErrorResponse(
			ctx, 
			requestID, 
			"RESPONSE_PARSE_ERROR", 
			"Failed to parse response", 
			0,
		)
		return h.responseFormatterService.ConvertToVerificationResponse(errorResponse)
	}
	
	// Format response (T-013)
	processingTime := time.Since(ctx.Value("start_time").(time.Time))
	requestHash := ctx.Value("request_hash").(string)
	
	formattedResponse, err := h.responseFormatterService.FormatResponse(
		ctx,
		parsedResponse,
		requestID,
		processingTime,
		requestHash,
	)
	if err != nil {
		// Handle formatting error
		errorResponse := h.responseFormatterService.FormatErrorResponse(
			ctx,
			requestID,
			"RESPONSE_FORMAT_ERROR",
			"Failed to format response",
			processingTime,
		)
		return h.responseFormatterService.ConvertToVerificationResponse(errorResponse)
	}
	
	// Generate JWS attestation (T-014)
	jwsResult, err := h.jwsAttestationService.GenerateJWS(
		ctx,
		formattedResponse,
		"pavilion-trust", // issuer
		req.RPID,         // audience
	)
	if err != nil {
		// Log JWS signing error but continue with response
		h.auditService.LogVerification(ctx, req, nil, "JWS_SIGNING_ERROR")
		// Continue without JWS attestation
	} else {
		// Add JWS to response metadata
		if formattedResponse.Metadata == nil {
			formattedResponse.Metadata = make(map[string]string)
		}
		formattedResponse.Metadata["jws_token"] = jwsResult.Token
		formattedResponse.Metadata["jws_id"] = jwsResult.JWSID
		formattedResponse.Metadata["jws_issuer"] = jwsResult.Payload.Claims.Issuer
		formattedResponse.Metadata["jws_audience"] = jwsResult.Payload.Claims.Audience
	}
	
	// Convert to verification response
	verificationResponse := h.responseFormatterService.ConvertToVerificationResponse(formattedResponse)
	
	// Add audit reference placeholder (will be filled by caller)
	// The actual audit reference will be added after audit logging
	verificationResponse.AuditReference = "" // Will be set by caller
	
	return verificationResponse
}

// generateResponse creates a verification response (legacy method)
func (h *VerificationHandler) generateResponse(req models.VerificationRequest, dpResponse *services.DPResponse, requestID string) *models.VerificationResponse {
	// Convert services.DPResponse to models.DPResponse
	modelsDPResponse := &models.DPResponse{
		Status:         dpResponse.Status,
		Verified:       false, // Will be set from VerificationResult if available
		ConfidenceScore: 0.0,  // Will be set from VerificationResult if available
		DPID:          "dp-connector",
		Timestamp:     dpResponse.Timestamp,
	}

	// Extract verification result if available
	if dpResponse.VerificationResult != nil {
		modelsDPResponse.Verified = dpResponse.VerificationResult.Verified
		modelsDPResponse.ConfidenceScore = dpResponse.VerificationResult.Confidence
		modelsDPResponse.Reason = dpResponse.VerificationResult.Reason
		modelsDPResponse.Evidence = dpResponse.VerificationResult.Evidence
	}

	response := models.NewVerificationResponse(modelsDPResponse.Status, modelsDPResponse.ConfidenceScore, requestID)
	
	// Set additional fields
	response.Verified = modelsDPResponse.Verified
	response.Reason = modelsDPResponse.Reason
	response.Evidence = modelsDPResponse.Evidence
	response.DPID = modelsDPResponse.DPID
	
	// JWS attestation is handled in generateFormattedResponse (T-014)
	// Audit references will be implemented in T-015
	
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