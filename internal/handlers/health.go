package handlers

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/pavilion-trust/core-broker/internal/config"
	"github.com/pavilion-trust/core-broker/internal/services"
)

// HealthHandler handles health check requests
type HealthHandler struct {
	config *config.Config
	cacheService *services.CacheService
	policyService *services.PolicyService
	authorizationService *services.AuthorizationService
	dpService *services.DPConnectorService
	auditService *services.AuditService
	keycloakService *services.KeycloakService
	privacyService *services.PrivacyService
	privacyGuaranteesService *services.PrivacyGuaranteesService
}

// NewHealthHandler creates a new health handler
func NewHealthHandler(cfg *config.Config) *HealthHandler {
	policyService := services.NewPolicyService(cfg)
	return &HealthHandler{
		config: cfg,
		cacheService: services.NewCacheService(cfg),
		policyService: policyService,
		authorizationService: services.NewAuthorizationService(cfg, policyService),
		dpService: services.NewDPConnectorService(cfg),
		auditService: services.NewAuditService(cfg),
		keycloakService: services.NewKeycloakService(cfg),
		privacyService: services.NewPrivacyService(cfg),
		privacyGuaranteesService: services.NewPrivacyGuaranteesService(cfg),
	}
}

// HandleHealth processes health check requests
func (h *HealthHandler) HandleHealth(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	
	// Check service dependencies
	health := &HealthResponse{
		Status:    "healthy",
		Timestamp: time.Now().Format(time.RFC3339),
		Version:   "0.1.0",
		Environment: h.config.Env,
		Dependencies: make(map[string]DependencyStatus),
	}
	
	// Check cache service
	if err := h.cacheService.HealthCheck(ctx); err != nil {
		health.Dependencies["cache"] = DependencyStatus{
			Status: "unhealthy",
			Error:  err.Error(),
		}
		health.Status = "degraded"
	} else {
		health.Dependencies["cache"] = DependencyStatus{
			Status: "healthy",
		}
	}
	
	// Check policy service
	if err := h.policyService.HealthCheck(ctx); err != nil {
		health.Dependencies["policy"] = DependencyStatus{
			Status: "unhealthy",
			Error:  err.Error(),
		}
		health.Status = "degraded"
	} else {
		health.Dependencies["policy"] = DependencyStatus{
			Status: "healthy",
		}
	}
	
	// Check authorization service
	if err := h.authorizationService.HealthCheck(ctx); err != nil {
		health.Dependencies["authorization"] = DependencyStatus{
			Status: "unhealthy",
			Error:  err.Error(),
		}
		health.Status = "degraded"
	} else {
		health.Dependencies["authorization"] = DependencyStatus{
			Status: "healthy",
		}
	}
	
	// Check privacy service
	if err := h.privacyService.HealthCheck(ctx); err != nil {
		health.Dependencies["privacy"] = DependencyStatus{
			Status: "unhealthy",
			Error:  err.Error(),
		}
		health.Status = "degraded"
	} else {
		health.Dependencies["privacy"] = DependencyStatus{
			Status: "healthy",
		}
	}
	
	// Check privacy guarantees service
	if err := h.privacyGuaranteesService.HealthCheck(ctx); err != nil {
		health.Dependencies["privacy_guarantees"] = DependencyStatus{
			Status: "unhealthy",
			Error:  err.Error(),
		}
		health.Status = "degraded"
	} else {
		health.Dependencies["privacy_guarantees"] = DependencyStatus{
			Status: "healthy",
		}
	}
	
	// Check DP connector service
	if err := h.dpService.HealthCheck(ctx); err != nil {
		health.Dependencies["dp_connector"] = DependencyStatus{
			Status: "unhealthy",
			Error:  err.Error(),
		}
		health.Status = "degraded"
	} else {
		health.Dependencies["dp_connector"] = DependencyStatus{
			Status: "healthy",
		}
	}
	
	// Check audit service
	if err := h.auditService.HealthCheck(ctx); err != nil {
		health.Dependencies["audit"] = DependencyStatus{
			Status: "unhealthy",
			Error:  err.Error(),
		}
		health.Status = "degraded"
	} else {
		health.Dependencies["audit"] = DependencyStatus{
			Status: "healthy",
		}
	}
	
	// Check Keycloak service
	if err := h.keycloakService.HealthCheck(ctx); err != nil {
		health.Dependencies["keycloak"] = DependencyStatus{
			Status: "unhealthy",
			Error:  err.Error(),
		}
		health.Status = "degraded"
	} else {
		health.Dependencies["keycloak"] = DependencyStatus{
			Status: "healthy",
		}
	}
	
	// If any dependency is unhealthy, mark overall status as unhealthy
	for _, dep := range health.Dependencies {
		if dep.Status == "unhealthy" {
			health.Status = "unhealthy"
			break
		}
	}
	
	// Set appropriate HTTP status code
	statusCode := http.StatusOK
	if health.Status == "unhealthy" {
		statusCode = http.StatusServiceUnavailable
	} else if health.Status == "degraded" {
		statusCode = http.StatusOK // Still OK but degraded
	}
	
	// Write response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(health)
}

// HealthResponse represents the health check response
type HealthResponse struct {
	Status       string                        `json:"status"`
	Timestamp    string                        `json:"timestamp"`
	Version      string                        `json:"version"`
	Environment  string                        `json:"environment"`
	Dependencies map[string]DependencyStatus   `json:"dependencies"`
}

// DependencyStatus represents the status of a dependency
type DependencyStatus struct {
	Status string `json:"status"`
	Error  string `json:"error,omitempty"`
} 