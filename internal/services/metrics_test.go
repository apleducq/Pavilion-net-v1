package services

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/pavilion-trust/core-broker/internal/config"
)

func TestMetricsService_RequestTracking(t *testing.T) {
	cfg := &config.Config{}
	service := NewMetricsService(cfg)
	
	t.Run("RecordRequest", func(t *testing.T) {
		latency := 100 * time.Millisecond
		service.RecordRequest(latency)
		
		metrics := service.GetMetrics()
		
		// Find request count metric
		var requestCount float64
		for _, metric := range metrics {
			if metric.Name == "core_broker_requests_total" {
				requestCount = metric.Value
				break
			}
		}
		
		if requestCount != 1.0 {
			t.Errorf("Expected request count 1.0, got %f", requestCount)
		}
	})
	
	t.Run("RecordError", func(t *testing.T) {
		service.RecordError()
		
		metrics := service.GetMetrics()
		
		// Find error count metric
		var errorCount float64
		for _, metric := range metrics {
			if metric.Name == "core_broker_errors_total" {
				errorCount = metric.Value
				break
			}
		}
		
		if errorCount != 1.0 {
			t.Errorf("Expected error count 1.0, got %f", errorCount)
		}
	})
	
	t.Run("ErrorRate", func(t *testing.T) {
		// Record more requests and errors
		service.RecordRequest(50 * time.Millisecond)
		service.RecordRequest(75 * time.Millisecond)
		service.RecordError()
		service.RecordError()
		
		metrics := service.GetMetrics()
		
		// Find error rate metric
		var errorRate float64
		for _, metric := range metrics {
			if metric.Name == "core_broker_error_rate" {
				errorRate = metric.Value
				break
			}
		}
		
		// Should be 3 errors out of 4 requests = 75%
		expectedRate := 75.0
		if errorRate != expectedRate {
			t.Errorf("Expected error rate %.1f%%, got %.1f%%", expectedRate, errorRate)
		}
	})
}

func TestMetricsService_LatencyTracking(t *testing.T) {
	cfg := &config.Config{}
	service := NewMetricsService(cfg)
	
	t.Run("LatencyMetrics", func(t *testing.T) {
		// Record requests with different latencies
		service.RecordRequest(100 * time.Millisecond)
		service.RecordRequest(200 * time.Millisecond)
		service.RecordRequest(300 * time.Millisecond)
		
		metrics := service.GetMetrics()
		
		// Find average latency metric
		var avgLatency float64
		var maxLatency float64
		for _, metric := range metrics {
			if metric.Name == "core_broker_request_latency_avg" {
				avgLatency = metric.Value
			}
			if metric.Name == "core_broker_request_latency_max" {
				maxLatency = metric.Value
			}
		}
		
		// Average should be 200ms (0.2 seconds)
		expectedAvg := 0.2
		if avgLatency != expectedAvg {
			t.Errorf("Expected average latency %.3f, got %.3f", expectedAvg, avgLatency)
		}
		
		// Max should be 300ms (0.3 seconds)
		expectedMax := 0.3
		if maxLatency != expectedMax {
			t.Errorf("Expected max latency %.3f, got %.3f", expectedMax, maxLatency)
		}
	})
}

func TestMetricsService_CacheTracking(t *testing.T) {
	cfg := &config.Config{}
	service := NewMetricsService(cfg)
	
	t.Run("CacheMetrics", func(t *testing.T) {
		// Record cache activity
		service.RecordCacheHit()
		service.RecordCacheHit()
		service.RecordCacheMiss()
		service.RecordCacheError()
		
		metrics := service.GetMetrics()
		
		// Find cache metrics
		var cacheHits, cacheMisses, cacheErrors, cacheHitRate float64
		for _, metric := range metrics {
			switch metric.Name {
			case "core_broker_cache_hits_total":
				cacheHits = metric.Value
			case "core_broker_cache_misses_total":
				cacheMisses = metric.Value
			case "core_broker_cache_errors_total":
				cacheErrors = metric.Value
			case "core_broker_cache_hit_rate":
				cacheHitRate = metric.Value
			}
		}
		
		if cacheHits != 2.0 {
			t.Errorf("Expected cache hits 2.0, got %f", cacheHits)
		}
		
		if cacheMisses != 1.0 {
			t.Errorf("Expected cache misses 1.0, got %f", cacheMisses)
		}
		
		if cacheErrors != 1.0 {
			t.Errorf("Expected cache errors 1.0, got %f", cacheErrors)
		}
		
		// Hit rate should be 2 hits out of 3 total requests = 66.67%
		expectedHitRate := 66.67
		if cacheHitRate != expectedHitRate {
			t.Errorf("Expected cache hit rate %.2f%%, got %.2f%%", expectedHitRate, cacheHitRate)
		}
	})
}

func TestMetricsService_ServiceSpecificTracking(t *testing.T) {
	cfg := &config.Config{}
	service := NewMetricsService(cfg)
	
	t.Run("ServiceMetrics", func(t *testing.T) {
		// Record service-specific metrics
		service.RecordDPConnectorRequest()
		service.RecordDPConnectorRequest()
		service.RecordDPConnectorError()
		service.RecordAuditLogEntry()
		service.RecordPolicyDecision()
		
		metrics := service.GetMetrics()
		
		// Find service-specific metrics
		var dpRequests, dpErrors, auditEntries, policyDecisions float64
		for _, metric := range metrics {
			switch metric.Name {
			case "core_broker_dp_connector_requests_total":
				dpRequests = metric.Value
			case "core_broker_dp_connector_errors_total":
				dpErrors = metric.Value
			case "core_broker_audit_log_entries_total":
				auditEntries = metric.Value
			case "core_broker_policy_decisions_total":
				policyDecisions = metric.Value
			}
		}
		
		if dpRequests != 2.0 {
			t.Errorf("Expected DP requests 2.0, got %f", dpRequests)
		}
		
		if dpErrors != 1.0 {
			t.Errorf("Expected DP errors 1.0, got %f", dpErrors)
		}
		
		if auditEntries != 1.0 {
			t.Errorf("Expected audit entries 1.0, got %f", auditEntries)
		}
		
		if policyDecisions != 1.0 {
			t.Errorf("Expected policy decisions 1.0, got %f", policyDecisions)
		}
	})
}

func TestMetricsService_PrometheusFormat(t *testing.T) {
	cfg := &config.Config{}
	service := NewMetricsService(cfg)
	
	// Record some metrics
	service.RecordRequest(100 * time.Millisecond)
	service.RecordError()
	service.RecordCacheHit()
	
	t.Run("PrometheusFormat", func(t *testing.T) {
		prometheus := service.GetPrometheusMetrics()
		
		// Check for required Prometheus format elements
		if !strings.Contains(prometheus, "# HELP") {
			t.Error("Expected HELP comments in Prometheus format")
		}
		
		if !strings.Contains(prometheus, "# TYPE") {
			t.Error("Expected TYPE comments in Prometheus format")
		}
		
		if !strings.Contains(prometheus, "core_broker_requests_total") {
			t.Error("Expected requests metric in Prometheus format")
		}
		
		if !strings.Contains(prometheus, "core_broker_errors_total") {
			t.Error("Expected errors metric in Prometheus format")
		}
		
		if !strings.Contains(prometheus, "core_broker_cache_hits_total") {
			t.Error("Expected cache hits metric in Prometheus format")
		}
	})
}

func TestMetricsService_AlertThresholds(t *testing.T) {
	cfg := &config.Config{}
	service := NewMetricsService(cfg)
	
	t.Run("AlertThresholds", func(t *testing.T) {
		// Record high error rate to trigger alerts
		for i := 0; i < 10; i++ {
			service.RecordRequest(100 * time.Millisecond)
		}
		for i := 0; i < 6; i++ {
			service.RecordError() // 60% error rate
		}
		
		alerts := service.CheckAlertThresholds()
		
		// Should have alerts for high error rate
		if len(alerts) == 0 {
			t.Error("Expected alerts for high error rate")
		}
		
		// Check for specific alerts
		foundWarning := false
		foundCritical := false
		for _, alert := range alerts {
			if alert.Severity == "warning" && strings.Contains(alert.Message, "Error rate is above 5%") {
				foundWarning = true
			}
			if alert.Severity == "critical" && strings.Contains(alert.Message, "Error rate is above 10%") {
				foundCritical = true
			}
		}
		
		if !foundWarning {
			t.Error("Expected warning alert for error rate above 5%")
		}
		
		if !foundCritical {
			t.Error("Expected critical alert for error rate above 10%")
		}
	})
	
	t.Run("NoAlerts", func(t *testing.T) {
		// Reset metrics
		service.ResetMetrics()
		
		// Record low error rate
		for i := 0; i < 10; i++ {
			service.RecordRequest(100 * time.Millisecond)
		}
		service.RecordError() // 10% error rate
		
		alerts := service.CheckAlertThresholds()
		
		// Should not have critical alerts
		for _, alert := range alerts {
			if alert.Severity == "critical" {
				t.Error("Expected no critical alerts for 10% error rate")
			}
		}
	})
}

func TestMetricsService_ThresholdEvaluation(t *testing.T) {
	cfg := &config.Config{}
	service := NewMetricsService(cfg)
	
	t.Run("ThresholdEvaluation", func(t *testing.T) {
		// Test different operators
		testCases := []struct {
			value    float64
			operator string
			threshold float64
			expected  bool
		}{
			{5.0, "gt", 3.0, true},
			{2.0, "gt", 3.0, false},
			{3.0, "gte", 3.0, true},
			{2.0, "gte", 3.0, false},
			{2.0, "lt", 3.0, true},
			{4.0, "lt", 3.0, false},
			{3.0, "lte", 3.0, true},
			{4.0, "lte", 3.0, false},
			{3.0, "eq", 3.0, true},
			{4.0, "eq", 3.0, false},
		}
		
		for _, tc := range testCases {
			result := service.evaluateThreshold(tc.value, tc.operator, tc.threshold)
			if result != tc.expected {
				t.Errorf("evaluateThreshold(%.1f, %s, %.1f) = %v, expected %v", 
					tc.value, tc.operator, tc.threshold, result, tc.expected)
			}
		}
	})
}

func TestMetricsService_LatencyCalculations(t *testing.T) {
	cfg := &config.Config{}
	service := NewMetricsService(cfg)
	
	t.Run("LatencyCalculations", func(t *testing.T) {
		// Record latencies
		service.RecordRequest(100 * time.Millisecond)
		service.RecordRequest(200 * time.Millisecond)
		service.RecordRequest(300 * time.Millisecond)
		
		avgLatency := service.calculateAverageLatency()
		maxLatency := service.calculateMaxLatency()
		
		expectedAvg := 200 * time.Millisecond
		if avgLatency != expectedAvg {
			t.Errorf("Expected average latency %v, got %v", expectedAvg, avgLatency)
		}
		
		expectedMax := 300 * time.Millisecond
		if maxLatency != expectedMax {
			t.Errorf("Expected max latency %v, got %v", expectedMax, maxLatency)
		}
	})
	
	t.Run("EmptyLatency", func(t *testing.T) {
		// Test with no latency data
		service.ResetMetrics()
		
		avgLatency := service.calculateAverageLatency()
		maxLatency := service.calculateMaxLatency()
		
		if avgLatency != 0 {
			t.Errorf("Expected zero average latency for empty data, got %v", avgLatency)
		}
		
		if maxLatency != 0 {
			t.Errorf("Expected zero max latency for empty data, got %v", maxLatency)
		}
	})
}

func TestMetricsService_ResetMetrics(t *testing.T) {
	cfg := &config.Config{}
	service := NewMetricsService(cfg)
	
	t.Run("ResetMetrics", func(t *testing.T) {
		// Record some metrics
		service.RecordRequest(100 * time.Millisecond)
		service.RecordError()
		service.RecordCacheHit()
		
		// Reset
		service.ResetMetrics()
		
		// Check that metrics are reset
		summary := service.GetMetricsSummary()
		
		if summary["request_count"] != int64(0) {
			t.Error("Expected request count to be reset to 0")
		}
		
		if summary["error_count"] != int64(0) {
			t.Error("Expected error count to be reset to 0")
		}
		
		if summary["cache_hits"] != int64(0) {
			t.Error("Expected cache hits to be reset to 0")
		}
	})
}

func TestMetricsService_GetMetricsSummary(t *testing.T) {
	cfg := &config.Config{}
	service := NewMetricsService(cfg)
	
	t.Run("GetMetricsSummary", func(t *testing.T) {
		// Record some metrics
		service.RecordRequest(100 * time.Millisecond)
		service.RecordRequest(200 * time.Millisecond)
		service.RecordError()
		service.RecordCacheHit()
		service.RecordCacheMiss()
		
		summary := service.GetMetricsSummary()
		
		// Check required fields
		requiredFields := []string{
			"uptime", "request_count", "error_count", "error_rate",
			"cache_hits", "cache_misses", "cache_hit_rate",
			"dp_requests", "dp_errors", "audit_entries", "policy_decisions",
			"last_reset",
		}
		
		for _, field := range requiredFields {
			if summary[field] == nil {
				t.Errorf("Expected field %s in metrics summary", field)
			}
		}
		
		// Check specific values
		if summary["request_count"] != int64(2) {
			t.Errorf("Expected request count 2, got %v", summary["request_count"])
		}
		
		if summary["error_count"] != int64(1) {
			t.Errorf("Expected error count 1, got %v", summary["error_count"])
		}
		
		if summary["cache_hits"] != int64(1) {
			t.Errorf("Expected cache hits 1, got %v", summary["cache_hits"])
		}
		
		if summary["cache_misses"] != int64(1) {
			t.Errorf("Expected cache misses 1, got %v", summary["cache_misses"])
		}
	})
}

func TestMetricsService_HealthCheck(t *testing.T) {
	cfg := &config.Config{}
	service := NewMetricsService(cfg)
	
	t.Run("HealthCheck", func(t *testing.T) {
		ctx := context.Background()
		err := service.HealthCheck(ctx)
		if err != nil {
			t.Errorf("Health check failed: %v", err)
		}
	})
} 