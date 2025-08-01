package services

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/pavilion-trust/core-broker/internal/config"
)

// MetricsService handles monitoring and metrics collection
type MetricsService struct {
	config *config.Config
	mu     sync.RWMutex
	
	// Request metrics
	requestCount     int64
	requestLatency   []time.Duration
	errorCount       int64
	
	// Cache metrics
	cacheHits        int64
	cacheMisses      int64
	cacheErrors      int64
	
	// Service-specific metrics
	dpConnectorRequests int64
	dpConnectorErrors   int64
	auditLogEntries     int64
	policyDecisions     int64
	
	// Performance tracking
	startTime time.Time
	lastReset time.Time
}

// MetricType represents different types of metrics
type MetricType string

const (
	MetricTypeCounter   MetricType = "counter"
	MetricTypeGauge     MetricType = "gauge"
	MetricTypeHistogram MetricType = "histogram"
)

// Metric represents a single metric
type Metric struct {
	Name   string                 `json:"name"`
	Type   MetricType             `json:"type"`
	Value  float64                `json:"value"`
	Labels map[string]string      `json:"labels,omitempty"`
	Help   string                 `json:"help,omitempty"`
	Time   time.Time              `json:"timestamp"`
}

// AlertThreshold represents an alerting threshold
type AlertThreshold struct {
	MetricName string  `json:"metric_name"`
	Operator   string  `json:"operator"` // "gt", "lt", "eq", "gte", "lte"
	Value      float64 `json:"value"`
	Severity   string  `json:"severity"` // "warning", "critical"
	Message    string  `json:"message"`
}

// NewMetricsService creates a new metrics service
func NewMetricsService(cfg *config.Config) *MetricsService {
	return &MetricsService{
		config:    cfg,
		startTime: time.Now(),
		lastReset: time.Now(),
	}
}

// RecordRequest records a request with latency
func (s *MetricsService) RecordRequest(latency time.Duration) {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	s.requestCount++
	s.requestLatency = append(s.requestLatency, latency)
	
	// Keep only last 1000 latency measurements
	if len(s.requestLatency) > 1000 {
		s.requestLatency = s.requestLatency[1:]
	}
}

// RecordError records an error
func (s *MetricsService) RecordError() {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	s.errorCount++
}

// RecordCacheHit records a cache hit
func (s *MetricsService) RecordCacheHit() {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	s.cacheHits++
}

// RecordCacheMiss records a cache miss
func (s *MetricsService) RecordCacheMiss() {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	s.cacheMisses++
}

// RecordCacheError records a cache error
func (s *MetricsService) RecordCacheError() {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	s.cacheErrors++
}

// RecordDPConnectorRequest records a DP connector request
func (s *MetricsService) RecordDPConnectorRequest() {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	s.dpConnectorRequests++
}

// RecordDPConnectorError records a DP connector error
func (s *MetricsService) RecordDPConnectorError() {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	s.dpConnectorErrors++
}

// RecordAuditLogEntry records an audit log entry
func (s *MetricsService) RecordAuditLogEntry() {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	s.auditLogEntries++
}

// RecordPolicyDecision records a policy decision
func (s *MetricsService) RecordPolicyDecision() {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	s.policyDecisions++
}

// GetMetrics returns all current metrics
func (s *MetricsService) GetMetrics() []Metric {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	now := time.Now()
	uptime := now.Sub(s.startTime)
	
	var metrics []Metric
	
	// Request metrics
	metrics = append(metrics, Metric{
		Name:   "core_broker_requests_total",
		Type:   MetricTypeCounter,
		Value:  float64(s.requestCount),
		Help:   "Total number of requests",
		Time:   now,
	})
	
	metrics = append(metrics, Metric{
		Name:   "core_broker_errors_total",
		Type:   MetricTypeCounter,
		Value:  float64(s.errorCount),
		Help:   "Total number of errors",
		Time:   now,
	})
	
	// Error rate
	errorRate := 0.0
	if s.requestCount > 0 {
		errorRate = float64(s.errorCount) / float64(s.requestCount) * 100.0
	}
	
	metrics = append(metrics, Metric{
		Name:   "core_broker_error_rate",
		Type:   MetricTypeGauge,
		Value:  errorRate,
		Help:   "Error rate as percentage",
		Time:   now,
	})
	
	// Latency metrics
	if len(s.requestLatency) > 0 {
		avgLatency := s.calculateAverageLatency()
		metrics = append(metrics, Metric{
			Name:   "core_broker_request_latency_avg",
			Type:   MetricTypeGauge,
			Value:  avgLatency.Seconds(),
			Help:   "Average request latency in seconds",
			Time:   now,
		})
		
		maxLatency := s.calculateMaxLatency()
		metrics = append(metrics, Metric{
			Name:   "core_broker_request_latency_max",
			Type:   MetricTypeGauge,
			Value:  maxLatency.Seconds(),
			Help:   "Maximum request latency in seconds",
			Time:   now,
		})
	}
	
	// Cache metrics
	metrics = append(metrics, Metric{
		Name:   "core_broker_cache_hits_total",
		Type:   MetricTypeCounter,
		Value:  float64(s.cacheHits),
		Help:   "Total cache hits",
		Time:   now,
	})
	
	metrics = append(metrics, Metric{
		Name:   "core_broker_cache_misses_total",
		Type:   MetricTypeCounter,
		Value:  float64(s.cacheMisses),
		Help:   "Total cache misses",
		Time:   now,
	})
	
	metrics = append(metrics, Metric{
		Name:   "core_broker_cache_errors_total",
		Type:   MetricTypeCounter,
		Value:  float64(s.cacheErrors),
		Help:   "Total cache errors",
		Time:   now,
	})
	
	// Cache hit rate
	cacheHitRate := 0.0
	totalCacheRequests := s.cacheHits + s.cacheMisses
	if totalCacheRequests > 0 {
		cacheHitRate = float64(s.cacheHits) / float64(totalCacheRequests) * 100.0
	}
	
	metrics = append(metrics, Metric{
		Name:   "core_broker_cache_hit_rate",
		Type:   MetricTypeGauge,
		Value:  cacheHitRate,
		Help:   "Cache hit rate as percentage",
		Time:   now,
	})
	
	// Service-specific metrics
	metrics = append(metrics, Metric{
		Name:   "core_broker_dp_connector_requests_total",
		Type:   MetricTypeCounter,
		Value:  float64(s.dpConnectorRequests),
		Help:   "Total DP connector requests",
		Time:   now,
	})
	
	metrics = append(metrics, Metric{
		Name:   "core_broker_dp_connector_errors_total",
		Type:   MetricTypeCounter,
		Value:  float64(s.dpConnectorErrors),
		Help:   "Total DP connector errors",
		Time:   now,
	})
	
	metrics = append(metrics, Metric{
		Name:   "core_broker_audit_log_entries_total",
		Type:   MetricTypeCounter,
		Value:  float64(s.auditLogEntries),
		Help:   "Total audit log entries",
		Time:   now,
	})
	
	metrics = append(metrics, Metric{
		Name:   "core_broker_policy_decisions_total",
		Type:   MetricTypeCounter,
		Value:  float64(s.policyDecisions),
		Help:   "Total policy decisions",
		Time:   now,
	})
	
	// Uptime metric
	metrics = append(metrics, Metric{
		Name:   "core_broker_uptime_seconds",
		Type:   MetricTypeGauge,
		Value:  uptime.Seconds(),
		Help:   "Service uptime in seconds",
		Time:   now,
	})
	
	return metrics
}

// GetPrometheusMetrics returns metrics in Prometheus format
func (s *MetricsService) GetPrometheusMetrics() string {
	metrics := s.GetMetrics()
	var prometheus string
	
	for _, metric := range metrics {
		// Add help text
		if metric.Help != "" {
			prometheus += fmt.Sprintf("# HELP %s %s\n", metric.Name, metric.Help)
		}
		
		// Add type
		prometheus += fmt.Sprintf("# TYPE %s %s\n", metric.Name, metric.Type)
		
		// Add metric value
		labels := ""
		if len(metric.Labels) > 0 {
			var labelPairs []string
			for k, v := range metric.Labels {
				labelPairs = append(labelPairs, fmt.Sprintf(`%s="%s"`, k, v))
			}
			labels = fmt.Sprintf("{%s}", fmt.Sprintf("%s", labelPairs))
		}
		
		prometheus += fmt.Sprintf("%s%s %f\n", metric.Name, labels, metric.Value)
	}
	
	return prometheus
}

// CheckAlertThresholds checks if any metrics exceed alert thresholds
func (s *MetricsService) CheckAlertThresholds() []Alert {
	metrics := s.GetMetrics()
	var alerts []Alert
	
	// Define alert thresholds
	thresholds := []AlertThreshold{
		{
			MetricName: "core_broker_error_rate",
			Operator:   "gt",
			Value:      5.0, // 5% error rate
			Severity:   "warning",
			Message:    "Error rate is above 5%",
		},
		{
			MetricName: "core_broker_error_rate",
			Operator:   "gt",
			Value:      10.0, // 10% error rate
			Severity:   "critical",
			Message:    "Error rate is above 10%",
		},
		{
			MetricName: "core_broker_cache_hit_rate",
			Operator:   "lt",
			Value:      80.0, // 80% cache hit rate
			Severity:   "warning",
			Message:    "Cache hit rate is below 80%",
		},
		{
			MetricName: "core_broker_request_latency_avg",
			Operator:   "gt",
			Value:      2.0, // 2 seconds
			Severity:   "warning",
			Message:    "Average request latency is above 2 seconds",
		},
	}
	
	// Check each threshold
	for _, threshold := range thresholds {
		for _, metric := range metrics {
			if metric.Name == threshold.MetricName {
				if s.evaluateThreshold(metric.Value, threshold.Operator, threshold.Value) {
					alerts = append(alerts, Alert{
						Severity:   threshold.Severity,
						Message:    threshold.Message,
						MetricName: threshold.MetricName,
						Value:      metric.Value,
						Threshold:  threshold.Value,
						Timestamp:  time.Now(),
					})
				}
				break
			}
		}
	}
	
	return alerts
}

// Alert represents an alert
type Alert struct {
	Severity   string    `json:"severity"`
	Message    string    `json:"message"`
	MetricName string    `json:"metric_name"`
	Value      float64   `json:"value"`
	Threshold  float64   `json:"threshold"`
	Timestamp  time.Time `json:"timestamp"`
}

// evaluateThreshold evaluates if a metric value meets the threshold condition
func (s *MetricsService) evaluateThreshold(value, operator string, threshold float64) bool {
	switch operator {
	case "gt":
		return value > threshold
	case "gte":
		return value >= threshold
	case "lt":
		return value < threshold
	case "lte":
		return value <= threshold
	case "eq":
		return value == threshold
	default:
		return false
	}
}

// calculateAverageLatency calculates the average request latency
func (s *MetricsService) calculateAverageLatency() time.Duration {
	if len(s.requestLatency) == 0 {
		return 0
	}
	
	var total time.Duration
	for _, latency := range s.requestLatency {
		total += latency
	}
	
	return total / time.Duration(len(s.requestLatency))
}

// calculateMaxLatency calculates the maximum request latency
func (s *MetricsService) calculateMaxLatency() time.Duration {
	if len(s.requestLatency) == 0 {
		return 0
	}
	
	max := s.requestLatency[0]
	for _, latency := range s.requestLatency {
		if latency > max {
			max = latency
		}
	}
	
	return max
}

// ResetMetrics resets all metrics counters
func (s *MetricsService) ResetMetrics() {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	s.requestCount = 0
	s.errorCount = 0
	s.cacheHits = 0
	s.cacheMisses = 0
	s.cacheErrors = 0
	s.dpConnectorRequests = 0
	s.dpConnectorErrors = 0
	s.auditLogEntries = 0
	s.policyDecisions = 0
	s.requestLatency = nil
	s.lastReset = time.Now()
}

// GetMetricsSummary returns a summary of key metrics
func (s *MetricsService) GetMetricsSummary() map[string]interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	uptime := time.Since(s.startTime)
	errorRate := 0.0
	if s.requestCount > 0 {
		errorRate = float64(s.errorCount) / float64(s.requestCount) * 100.0
	}
	
	cacheHitRate := 0.0
	totalCacheRequests := s.cacheHits + s.cacheMisses
	if totalCacheRequests > 0 {
		cacheHitRate = float64(s.cacheHits) / float64(totalCacheRequests) * 100.0
	}
	
	return map[string]interface{}{
		"uptime":           uptime.String(),
		"request_count":    s.requestCount,
		"error_count":      s.errorCount,
		"error_rate":       errorRate,
		"cache_hits":       s.cacheHits,
		"cache_misses":     s.cacheMisses,
		"cache_hit_rate":   cacheHitRate,
		"dp_requests":      s.dpConnectorRequests,
		"dp_errors":        s.dpConnectorErrors,
		"audit_entries":    s.auditLogEntries,
		"policy_decisions": s.policyDecisions,
		"last_reset":       s.lastReset.Format(time.RFC3339),
	}
}

// HealthCheck checks if the metrics service is healthy
func (s *MetricsService) HealthCheck(ctx context.Context) error {
	// Basic health check - metrics service is always healthy
	// as it's just in-memory counters
	return nil
} 