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

// PolicyHandler handles policy-related HTTP requests
type PolicyHandler struct {
	config  *config.Config
	storage models.PolicyStorage
}

// NewPolicyHandler creates a new policy handler
func NewPolicyHandler(cfg *config.Config, storage models.PolicyStorage) *PolicyHandler {
	return &PolicyHandler{
		config:  cfg,
		storage: storage,
	}
}

// HandleCreatePolicy handles POST /policies
func (h *PolicyHandler) HandleCreatePolicy(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Parse request body
	var policy models.Policy
	if err := json.NewDecoder(r.Body).Decode(&policy); err != nil {
		http.Error(w, fmt.Sprintf("Invalid request body: %v", err), http.StatusBadRequest)
		return
	}

	// Validate policy
	if err := policy.Validate(); err != nil {
		http.Error(w, fmt.Sprintf("Policy validation failed: %v", err), http.StatusBadRequest)
		return
	}

	// Set creation metadata
	policy.CreatedAt = time.Now().Format(time.RFC3339)
	policy.UpdatedAt = policy.CreatedAt
	policy.Status = "draft"

	// Create policy
	if err := h.storage.CreatePolicy(ctx, &policy); err != nil {
		http.Error(w, fmt.Sprintf("Failed to create policy: %v", err), http.StatusInternalServerError)
		return
	}

	// Return success response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)

	response := map[string]interface{}{
		"id":      policy.ID,
		"message": "Policy created successfully",
		"status":  "success",
	}

	json.NewEncoder(w).Encode(response)
}

// HandleGetPolicy handles GET /policies/{id}
func (h *PolicyHandler) HandleGetPolicy(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	vars := mux.Vars(r)
	policyID := vars["id"]

	// Get policy from storage
	policy, err := h.storage.GetPolicy(ctx, policyID)
	if err != nil {
		if err.Error() == fmt.Sprintf("policy not found: %s", policyID) {
			http.Error(w, "Policy not found", http.StatusNotFound)
			return
		}
		http.Error(w, fmt.Sprintf("Failed to get policy: %v", err), http.StatusInternalServerError)
		return
	}

	// Return policy
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(policy)
}

// HandleUpdatePolicy handles PUT /policies/{id}
func (h *PolicyHandler) HandleUpdatePolicy(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	vars := mux.Vars(r)
	policyID := vars["id"]

	// Parse request body
	var policy models.Policy
	if err := json.NewDecoder(r.Body).Decode(&policy); err != nil {
		http.Error(w, fmt.Sprintf("Invalid request body: %v", err), http.StatusBadRequest)
		return
	}

	// Set policy ID from URL
	policy.ID = policyID

	// Validate policy
	if err := policy.Validate(); err != nil {
		http.Error(w, fmt.Sprintf("Policy validation failed: %v", err), http.StatusBadRequest)
		return
	}

	// Update policy
	if err := h.storage.UpdatePolicy(ctx, &policy); err != nil {
		if err.Error() == fmt.Sprintf("policy not found: %s", policyID) {
			http.Error(w, "Policy not found", http.StatusNotFound)
			return
		}
		http.Error(w, fmt.Sprintf("Failed to update policy: %v", err), http.StatusInternalServerError)
		return
	}

	// Return success response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	response := map[string]interface{}{
		"id":      policy.ID,
		"message": "Policy updated successfully",
		"status":  "success",
	}

	json.NewEncoder(w).Encode(response)
}

// HandleDeletePolicy handles DELETE /policies/{id}
func (h *PolicyHandler) HandleDeletePolicy(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	vars := mux.Vars(r)
	policyID := vars["id"]

	// Delete policy
	if err := h.storage.DeletePolicy(ctx, policyID); err != nil {
		if err.Error() == fmt.Sprintf("policy not found: %s", policyID) {
			http.Error(w, "Policy not found", http.StatusNotFound)
			return
		}
		http.Error(w, fmt.Sprintf("Failed to delete policy: %v", err), http.StatusInternalServerError)
		return
	}

	// Return success response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	response := map[string]interface{}{
		"id":      policyID,
		"message": "Policy deleted successfully",
		"status":  "success",
	}

	json.NewEncoder(w).Encode(response)
}

// HandleListPolicies handles GET /policies
func (h *PolicyHandler) HandleListPolicies(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Parse query parameters
	filters := make(map[string]interface{})

	if status := r.URL.Query().Get("status"); status != "" {
		filters["status"] = status
	}

	if createdBy := r.URL.Query().Get("created_by"); createdBy != "" {
		filters["created_by"] = createdBy
	}

	// Get policies from storage
	policies, err := h.storage.ListPolicies(ctx, filters)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to list policies: %v", err), http.StatusInternalServerError)
		return
	}

	// Return policies
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	response := map[string]interface{}{
		"policies": policies,
		"count":    len(policies),
		"status":   "success",
	}

	json.NewEncoder(w).Encode(response)
}

// HandleCreateTemplate handles POST /policies/templates
func (h *PolicyHandler) HandleCreateTemplate(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Parse request body
	var template models.PolicyTemplate
	if err := json.NewDecoder(r.Body).Decode(&template); err != nil {
		http.Error(w, fmt.Sprintf("Invalid request body: %v", err), http.StatusBadRequest)
		return
	}

	// Validate template
	if err := template.Validate(); err != nil {
		http.Error(w, fmt.Sprintf("Template validation failed: %v", err), http.StatusBadRequest)
		return
	}

	// Set creation metadata
	template.CreatedAt = time.Now().Format(time.RFC3339)
	template.UpdatedAt = template.CreatedAt
	template.Status = "active"

	// Create template
	if err := h.storage.CreateTemplate(ctx, &template); err != nil {
		http.Error(w, fmt.Sprintf("Failed to create template: %v", err), http.StatusInternalServerError)
		return
	}

	// Return success response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)

	response := map[string]interface{}{
		"id":      template.ID,
		"message": "Template created successfully",
		"status":  "success",
	}

	json.NewEncoder(w).Encode(response)
}

// HandleGetTemplate handles GET /policies/templates/{id}
func (h *PolicyHandler) HandleGetTemplate(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	vars := mux.Vars(r)
	templateID := vars["id"]

	// Get template from storage
	template, err := h.storage.GetTemplate(ctx, templateID)
	if err != nil {
		if err.Error() == fmt.Sprintf("template not found: %s", templateID) {
			http.Error(w, "Template not found", http.StatusNotFound)
			return
		}
		http.Error(w, fmt.Sprintf("Failed to get template: %v", err), http.StatusInternalServerError)
		return
	}

	// Return template
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(template)
}

// HandleListTemplates handles GET /policies/templates
func (h *PolicyHandler) HandleListTemplates(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Parse query parameters
	filters := make(map[string]interface{})

	if status := r.URL.Query().Get("status"); status != "" {
		filters["status"] = status
	}

	if category := r.URL.Query().Get("category"); category != "" {
		filters["category"] = category
	}

	// Get templates from storage
	templates, err := h.storage.ListTemplates(ctx, filters)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to list templates: %v", err), http.StatusInternalServerError)
		return
	}

	// Return templates
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	response := map[string]interface{}{
		"templates": templates,
		"count":     len(templates),
		"status":    "success",
	}

	json.NewEncoder(w).Encode(response)
}

// HandleEvaluatePolicy handles POST /policies/evaluate
func (h *PolicyHandler) HandleEvaluatePolicy(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Parse request body
	var request models.PolicyEvaluationRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, fmt.Sprintf("Invalid request body: %v", err), http.StatusBadRequest)
		return
	}

	// Validate request
	if err := request.Validate(); err != nil {
		http.Error(w, fmt.Sprintf("Request validation failed: %v", err), http.StatusBadRequest)
		return
	}

	// Get policy from storage
	policy, err := h.storage.GetPolicy(ctx, request.PolicyID)
	if err != nil {
		if err.Error() == fmt.Sprintf("policy not found: %s", request.PolicyID) {
			http.Error(w, "Policy not found", http.StatusNotFound)
			return
		}
		http.Error(w, fmt.Sprintf("Failed to get policy: %v", err), http.StatusInternalServerError)
		return
	}

	// Create rule engine and credential validator
	ruleEngine := services.NewRuleEngine()
	credentialValidator := services.NewCredentialValidator()

	// Validate credentials
	validationResults, err := credentialValidator.ValidateCredentials(ctx, request.Credentials)
	if err != nil {
		http.Error(w, fmt.Sprintf("Credential validation failed: %v", err), http.StatusInternalServerError)
		return
	}

	// Check if all credentials are valid
	for _, result := range validationResults {
		if !result.Valid {
			response := models.NewPolicyEvaluationResponse(
				request.RequestID,
				request.PolicyID,
				false,
				fmt.Sprintf("Credential validation failed: %s", result.Reason),
				0.0,
			)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(response)
			return
		}
	}

	// Evaluate policy using rule engine
	response, err := ruleEngine.EvaluatePolicy(ctx, policy, request.Credentials)
	if err != nil {
		http.Error(w, fmt.Sprintf("Policy evaluation failed: %v", err), http.StatusInternalServerError)
		return
	}

	// Set the request ID
	response.RequestID = request.RequestID

	// Return evaluation response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// HandleHealth handles GET /policies/health
func (h *PolicyHandler) HandleHealth(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Check storage health by listing policies with limit
	_, err := h.storage.ListPolicies(ctx, map[string]interface{}{})
	if err != nil {
		http.Error(w, fmt.Sprintf("Storage health check failed: %v", err), http.StatusInternalServerError)
		return
	}

	// Return health status
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	response := map[string]interface{}{
		"status":    "healthy",
		"timestamp": time.Now().Format(time.RFC3339),
		"service":   "policy-engine",
	}

	json.NewEncoder(w).Encode(response)
}

// HandleGetAuditLogs handles GET /policies/audit
func (h *PolicyHandler) HandleGetAuditLogs(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Parse query parameters
	filters := make(map[string]interface{})

	if rpID := r.URL.Query().Get("rp_id"); rpID != "" {
		filters["rp_id"] = rpID
	}

	if dpID := r.URL.Query().Get("dp_id"); dpID != "" {
		filters["dp_id"] = dpID
	}

	if claimType := r.URL.Query().Get("claim_type"); claimType != "" {
		filters["claim_type"] = claimType
	}

	if status := r.URL.Query().Get("status"); status != "" {
		filters["status"] = status
	}

	// Create audit logger
	auditStorage := services.NewAuditStorage()
	auditLogger := services.NewAuditLogger(auditStorage)

	// Get audit logs
	logs, err := auditLogger.GetAuditLogs(ctx, filters)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get audit logs: %v", err), http.StatusInternalServerError)
		return
	}

	// Return audit logs
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	response := map[string]interface{}{
		"logs":   logs,
		"count":  len(logs),
		"status": "success",
	}

	json.NewEncoder(w).Encode(response)
}

// HandleGetAuditLog handles GET /policies/audit/{request_id}
func (h *PolicyHandler) HandleGetAuditLog(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	vars := mux.Vars(r)
	requestID := vars["request_id"]

	// Create audit logger
	auditStorage := services.NewAuditStorage()
	auditLogger := services.NewAuditLogger(auditStorage)

	// Get specific audit log
	log, err := auditLogger.GetAuditLog(ctx, requestID)
	if err != nil {
		if err.Error() == fmt.Sprintf("audit log not found: %s", requestID) {
			http.Error(w, "Audit log not found", http.StatusNotFound)
			return
		}
		http.Error(w, fmt.Sprintf("Failed to get audit log: %v", err), http.StatusInternalServerError)
		return
	}

	// Return audit log
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(log)
}

// HandleGetAuditStats handles GET /policies/audit/stats
func (h *PolicyHandler) HandleGetAuditStats(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Create audit logger
	auditStorage := services.NewAuditStorage()
	auditLogger := services.NewAuditLogger(auditStorage)

	// Get audit statistics
	stats, err := auditLogger.GetAuditStats(ctx)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get audit stats: %v", err), http.StatusInternalServerError)
		return
	}

	// Return audit statistics
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(stats)
}
