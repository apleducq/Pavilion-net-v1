package services

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/pavilion-trust/core-broker/internal/models"
)

// AuditLogger handles audit logging for policy evaluations
type AuditLogger struct {
	storage AuditStorage
}

// AuditStorage interface defines methods for audit storage
type AuditStorage interface {
	LogEvaluation(ctx context.Context, entry *models.AuditEntry) error
	GetAuditLogs(ctx context.Context, filters map[string]interface{}) ([]*models.AuditEntry, error)
	GetAuditLog(ctx context.Context, requestID string) (*models.AuditEntry, error)
}

// AuditStorageImpl implements the AuditStorage interface
type AuditStorageImpl struct {
	// For MVP, we'll use in-memory storage
	// In production, this would be a database
	logs map[string]*models.AuditEntry
	mu   sync.RWMutex
}

// NewAuditStorage creates a new audit storage implementation
func NewAuditStorage() *AuditStorageImpl {
	return &AuditStorageImpl{
		logs: make(map[string]*models.AuditEntry),
	}
}

// LogEvaluation logs a policy evaluation
func (as *AuditStorageImpl) LogEvaluation(ctx context.Context, entry *models.AuditEntry) error {
	as.mu.Lock()
	defer as.mu.Unlock()

	as.logs[entry.RequestID] = entry
	return nil
}

// GetAuditLogs retrieves audit logs with filters
func (as *AuditStorageImpl) GetAuditLogs(ctx context.Context, filters map[string]interface{}) ([]*models.AuditEntry, error) {
	as.mu.RLock()
	defer as.mu.RUnlock()

	var logs []*models.AuditEntry

	for _, entry := range as.logs {
		// Apply filters
		if rpID, ok := filters["rp_id"].(string); ok && entry.RPID != rpID {
			continue
		}

		if dpID, ok := filters["dp_id"].(string); ok && entry.DPID != dpID {
			continue
		}

		if claimType, ok := filters["claim_type"].(string); ok && entry.ClaimType != claimType {
			continue
		}

		if status, ok := filters["status"].(string); ok && entry.Status != status {
			continue
		}

		logs = append(logs, entry)
	}

	return logs, nil
}

// GetAuditLog retrieves a specific audit log by request ID
func (as *AuditStorageImpl) GetAuditLog(ctx context.Context, requestID string) (*models.AuditEntry, error) {
	as.mu.RLock()
	defer as.mu.RUnlock()

	entry, exists := as.logs[requestID]
	if !exists {
		return nil, fmt.Errorf("audit log not found: %s", requestID)
	}

	return entry, nil
}

// NewAuditLogger creates a new audit logger
func NewAuditLogger(storage AuditStorage) *AuditLogger {
	return &AuditLogger{
		storage: storage,
	}
}

// LogPolicyEvaluation logs a policy evaluation
func (al *AuditLogger) LogPolicyEvaluation(ctx context.Context, request *models.PolicyEvaluationRequest, response *models.PolicyEvaluationResponse, policy *models.Policy) error {
	// Create privacy hash (no raw PII)
	privacyHash := al.createPrivacyHash(request)

	// Create Merkle proof (simplified for MVP)
	merkleProof := al.createMerkleProof(request, response)

	// Create audit entry
	entry := &models.AuditEntry{
		Timestamp:      time.Now().Format(time.RFC3339),
		RequestID:      request.RequestID,
		RPID:           "", // Will be set by caller
		DPID:           "", // Will be set by caller
		ClaimType:      "", // Will be set by caller
		PrivacyHash:    privacyHash,
		MerkleProof:    merkleProof,
		PolicyDecision: al.formatPolicyDecision(response),
		Status:         fmt.Sprintf("%t", response.Allowed),
		Metadata: map[string]interface{}{
			"policy_id":       policy.ID,
			"policy_name":     policy.Name,
			"confidence":      response.Confidence,
			"processing_time": response.ProcessingTime,
		},
	}

	return al.storage.LogEvaluation(ctx, entry)
}

// LogPolicyCreation logs a policy creation
func (al *AuditLogger) LogPolicyCreation(ctx context.Context, policy *models.Policy, createdBy string) error {
	entry := &models.AuditEntry{
		Timestamp:      time.Now().Format(time.RFC3339),
		RequestID:      fmt.Sprintf("policy-create-%s", policy.ID),
		RPID:           createdBy,
		ClaimType:      "policy_creation",
		PrivacyHash:    al.createPolicyHash(policy),
		MerkleProof:    al.createPolicyMerkleProof(policy),
		PolicyDecision: "created",
		Status:         "success",
		Metadata: map[string]interface{}{
			"policy_id":   policy.ID,
			"policy_name": policy.Name,
			"action":      "create",
		},
	}

	return al.storage.LogEvaluation(ctx, entry)
}

// LogPolicyUpdate logs a policy update
func (al *AuditLogger) LogPolicyUpdate(ctx context.Context, policy *models.Policy, updatedBy string) error {
	entry := &models.AuditEntry{
		Timestamp:      time.Now().Format(time.RFC3339),
		RequestID:      fmt.Sprintf("policy-update-%s", policy.ID),
		RPID:           updatedBy,
		ClaimType:      "policy_update",
		PrivacyHash:    al.createPolicyHash(policy),
		MerkleProof:    al.createPolicyMerkleProof(policy),
		PolicyDecision: "updated",
		Status:         "success",
		Metadata: map[string]interface{}{
			"policy_id":   policy.ID,
			"policy_name": policy.Name,
			"action":      "update",
		},
	}

	return al.storage.LogEvaluation(ctx, entry)
}

// LogPolicyDeletion logs a policy deletion
func (al *AuditLogger) LogPolicyDeletion(ctx context.Context, policyID string, deletedBy string) error {
	entry := &models.AuditEntry{
		Timestamp:      time.Now().Format(time.RFC3339),
		RequestID:      fmt.Sprintf("policy-delete-%s", policyID),
		RPID:           deletedBy,
		ClaimType:      "policy_deletion",
		PrivacyHash:    al.createSimpleHash(policyID),
		MerkleProof:    al.createSimpleMerkleProof(policyID),
		PolicyDecision: "deleted",
		Status:         "success",
		Metadata: map[string]interface{}{
			"policy_id": policyID,
			"action":    "delete",
		},
	}

	return al.storage.LogEvaluation(ctx, entry)
}

// GetAuditLogs retrieves audit logs with filters
func (al *AuditLogger) GetAuditLogs(ctx context.Context, filters map[string]interface{}) ([]*models.AuditEntry, error) {
	return al.storage.GetAuditLogs(ctx, filters)
}

// GetAuditLog retrieves a specific audit log
func (al *AuditLogger) GetAuditLog(ctx context.Context, requestID string) (*models.AuditEntry, error) {
	return al.storage.GetAuditLog(ctx, requestID)
}

// createPrivacyHash creates a privacy-preserving hash of the request
func (al *AuditLogger) createPrivacyHash(request *models.PolicyEvaluationRequest) string {
	// Create a hash that doesn't expose raw PII
	// Only hash non-sensitive metadata
	data := map[string]interface{}{
		"policy_id":        request.PolicyID,
		"request_id":       request.RequestID,
		"timestamp":        request.Timestamp,
		"credential_count": len(request.Credentials),
	}

	jsonData, _ := json.Marshal(data)
	hash := sha256.Sum256(jsonData)
	return fmt.Sprintf("%x", hash)
}

// createMerkleProof creates a Merkle proof for the evaluation
func (al *AuditLogger) createMerkleProof(request *models.PolicyEvaluationRequest, response *models.PolicyEvaluationResponse) string {
	// For MVP, create a simplified Merkle proof
	// In production, this would be a proper Merkle tree implementation

	data := map[string]interface{}{
		"request_id":   request.RequestID,
		"policy_id":    request.PolicyID,
		"evaluated_at": response.EvaluatedAt,
		"allowed":      response.Allowed,
		"confidence":   response.Confidence,
	}

	jsonData, _ := json.Marshal(data)
	hash := sha256.Sum256(jsonData)
	return fmt.Sprintf("%x", hash)
}

// createPolicyHash creates a hash of the policy
func (al *AuditLogger) createPolicyHash(policy *models.Policy) string {
	// Create a hash of the policy structure (excluding sensitive data)
	data := map[string]interface{}{
		"id":         policy.ID,
		"name":       policy.Name,
		"version":    policy.Version,
		"created_at": policy.CreatedAt,
		"created_by": policy.CreatedBy,
		"status":     policy.Status,
	}

	jsonData, _ := json.Marshal(data)
	hash := sha256.Sum256(jsonData)
	return fmt.Sprintf("%x", hash)
}

// createPolicyMerkleProof creates a Merkle proof for the policy
func (al *AuditLogger) createPolicyMerkleProof(policy *models.Policy) string {
	data := map[string]interface{}{
		"id":         policy.ID,
		"name":       policy.Name,
		"version":    policy.Version,
		"created_at": policy.CreatedAt,
		"updated_at": policy.UpdatedAt,
	}

	jsonData, _ := json.Marshal(data)
	hash := sha256.Sum256(jsonData)
	return fmt.Sprintf("%x", hash)
}

// createSimpleHash creates a simple hash of a string
func (al *AuditLogger) createSimpleHash(data string) string {
	hash := sha256.Sum256([]byte(data))
	return fmt.Sprintf("%x", hash)
}

// createSimpleMerkleProof creates a simple Merkle proof
func (al *AuditLogger) createSimpleMerkleProof(data string) string {
	hash := sha256.Sum256([]byte(data))
	return fmt.Sprintf("%x", hash)
}

// formatPolicyDecision formats the policy decision for logging
func (al *AuditLogger) formatPolicyDecision(response *models.PolicyEvaluationResponse) string {
	if response.Allowed {
		return fmt.Sprintf("allowed: %s (confidence: %.2f)", response.Reason, response.Confidence)
	}
	return fmt.Sprintf("denied: %s (confidence: %.2f)", response.Reason, response.Confidence)
}

// GetAuditStats returns audit statistics
func (al *AuditLogger) GetAuditStats(ctx context.Context) (map[string]interface{}, error) {
	logs, err := al.storage.GetAuditLogs(ctx, map[string]interface{}{})
	if err != nil {
		return nil, err
	}

	stats := map[string]interface{}{
		"total_logs":     len(logs),
		"allowed_count":  0,
		"denied_count":   0,
		"creation_count": 0,
		"update_count":   0,
		"deletion_count": 0,
	}

	for _, log := range logs {
		if log.Status == "success" {
			stats["allowed_count"] = stats["allowed_count"].(int) + 1
		} else {
			stats["denied_count"] = stats["denied_count"].(int) + 1
		}

		switch log.ClaimType {
		case "policy_creation":
			stats["creation_count"] = stats["creation_count"].(int) + 1
		case "policy_update":
			stats["update_count"] = stats["update_count"].(int) + 1
		case "policy_deletion":
			stats["deletion_count"] = stats["deletion_count"].(int) + 1
		}
	}

	return stats, nil
}
