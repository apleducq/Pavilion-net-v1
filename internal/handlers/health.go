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
	config                   *config.Config
	cacheService             *services.CacheService
	configCacheService       *services.ConfigCacheService
	policyService            *services.PolicyService
	authorizationService     *services.AuthorizationService
	dpService                *services.DPConnectorService
	auditService             *services.AuditService
	keycloakService          *services.KeycloakService
	privacyService           *services.PrivacyService
	privacyGuaranteesService *services.PrivacyGuaranteesService
	// Performance metrics
	startTime    time.Time
	requestCount int64
	errorCount   int64
}

// NewHealthHandler creates a new health handler
func NewHealthHandler(cfg *config.Config) *HealthHandler {
	policyService := services.NewPolicyService(cfg)
	return &HealthHandler{
		config:                   cfg,
		cacheService:             services.NewCacheService(cfg),
		configCacheService:       services.NewConfigCacheService(cfg),
		policyService:            policyService,
		authorizationService:     services.NewAuthorizationService(cfg, policyService),
		dpService:                services.NewDPConnectorService(cfg),
		auditService:             services.NewAuditService(cfg),
		keycloakService:          services.NewKeycloakService(cfg),
		privacyService:           services.NewPrivacyService(cfg),
		privacyGuaranteesService: services.NewPrivacyGuaranteesService(cfg),
		startTime:                time.Now(),
	}
}

// HandleHealth processes health check requests with enhanced metrics and graceful degradation
func (h *HealthHandler) HandleHealth(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Increment request count
	h.requestCount++

	// Check service dependencies with graceful degradation
	health := &HealthResponse{
		Status:       "healthy",
		Timestamp:    time.Now().Format(time.RFC3339),
		Version:      "0.1.0",
		Environment:  h.config.Env,
		Dependencies: make(map[string]DependencyStatus),
		Performance: PerformanceMetrics{
			Uptime:       time.Since(h.startTime).String(),
			RequestCount: h.requestCount,
			ErrorCount:   h.errorCount,
			ErrorRate:    h.calculateErrorRate(),
		},
	}

	// Check cache service
	if err := h.cacheService.HealthCheck(ctx); err != nil {
		health.Dependencies["cache"] = DependencyStatus{
			Status: "unhealthy",
			Error:  err.Error(),
		}
		h.errorCount++
		health.Status = "degraded"
	} else {
		health.Dependencies["cache"] = DependencyStatus{
			Status: "healthy",
		}
		// Add cache performance metrics
		if cacheMetrics := h.cacheService.GetCacheMetrics(); cacheMetrics != nil {
			dependency := health.Dependencies["cache"]
			dependency.Metrics = cacheMetrics
			health.Dependencies["cache"] = dependency
		}
	}

	// Check config cache service
	if err := h.configCacheService.HealthCheck(ctx); err != nil {
		health.Dependencies["config_cache"] = DependencyStatus{
			Status: "unhealthy",
			Error:  err.Error(),
		}
		h.errorCount++
		health.Status = "degraded"
	} else {
		health.Dependencies["config_cache"] = DependencyStatus{
			Status: "healthy",
		}
		// Add config cache performance metrics
		if configMetrics := h.configCacheService.GetCachePerformance(); configMetrics != nil {
			dependency := health.Dependencies["config_cache"]
			dependency.Metrics = configMetrics
			health.Dependencies["config_cache"] = dependency
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

// calculateErrorRate calculates the error rate as a percentage
func (h *HealthHandler) calculateErrorRate() float64 {
	if h.requestCount == 0 {
		return 0.0
	}
	return float64(h.errorCount) / float64(h.requestCount) * 100.0
}

// HealthResponse represents the health check response
type HealthResponse struct {
	Status       string                      `json:"status"`
	Timestamp    string                      `json:"timestamp"`
	Version      string                      `json:"version"`
	Environment  string                      `json:"environment"`
	Dependencies map[string]DependencyStatus `json:"dependencies"`
	Performance  PerformanceMetrics          `json:"performance"`
}

// DependencyStatus represents the status of a dependency
type DependencyStatus struct {
	Status  string                 `json:"status"`
	Error   string                 `json:"error,omitempty"`
	Metrics map[string]interface{} `json:"metrics,omitempty"`
}

// PerformanceMetrics represents performance metrics
type PerformanceMetrics struct {
	Uptime       string  `json:"uptime"`
	RequestCount int64   `json:"request_count"`
	ErrorCount   int64   `json:"error_count"`
	ErrorRate    float64 `json:"error_rate"`
}
