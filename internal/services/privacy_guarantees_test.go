package services

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/pavilion-trust/core-broker/internal/config"
)

func TestNewPrivacyGuaranteesService(t *testing.T) {
	cfg := &config.Config{}
	service := NewPrivacyGuaranteesService(cfg)
	
	if service.config != cfg {
		t.Error("Privacy guarantees service should have the correct config")
	}
	
	if service.securePool == nil {
		t.Error("Privacy guarantees service should have a secure pool")
	}
	
	if service.auditLogger == nil {
		t.Error("Privacy guarantees service should have an audit logger")
	}
	
	if service.minimizationSettings == nil {
		t.Error("Privacy guarantees service should have minimization settings")
	}
	
	if len(service.validationRules) == 0 {
		t.Error("Privacy guarantees service should have validation rules")
	}
}

func TestPrivacyGuaranteesService_SecureStoreAndRetrieve(t *testing.T) {
	cfg := &config.Config{}
	service := NewPrivacyGuaranteesService(cfg)
	
	testData := []byte("sensitive test data")
	key := "test_key"
	
	// Test secure store
	err := service.SecureStore(key, testData)
	if err != nil {
		t.Fatalf("SecureStore failed: %v", err)
	}
	
	// Test secure retrieve
	retrieved, err := service.SecureRetrieve(key)
	if err != nil {
		t.Fatalf("SecureRetrieve failed: %v", err)
	}
	
	if string(retrieved) != string(testData) {
		t.Errorf("Retrieved data does not match original data")
	}
	
	// Test that retrieved data is a copy, not the original
	if &retrieved[0] == &testData[0] {
		t.Error("Retrieved data should be a copy, not the original")
	}
}

func TestPrivacyGuaranteesService_SecureWipe(t *testing.T) {
	cfg := &config.Config{}
	service := NewPrivacyGuaranteesService(cfg)
	
	testData := []byte("sensitive test data")
	key := "test_key"
	
	// Store data
	err := service.SecureStore(key, testData)
	if err != nil {
		t.Fatalf("SecureStore failed: %v", err)
	}
	
	// Verify data exists
	_, err = service.SecureRetrieve(key)
	if err != nil {
		t.Fatalf("SecureRetrieve failed: %v", err)
	}
	
	// Wipe data
	err = service.SecureWipe(key)
	if err != nil {
		t.Fatalf("SecureWipe failed: %v", err)
	}
	
	// Verify data is gone
	_, err = service.SecureRetrieve(key)
	if err == nil {
		t.Error("Data should be wiped and not retrievable")
	}
}

func TestPrivacyGuaranteesService_ValidatePrivacyCompliance(t *testing.T) {
	cfg := &config.Config{}
	service := NewPrivacyGuaranteesService(cfg)
	
	testCases := []struct {
		fieldName string
		value     string
		expectError bool
	}{
		{"user_id", "test123", false},
		{"user_id", "", true}, // Required field
		{"first_name", "John", false},
		{"first_name", strings.Repeat("a", 51), true}, // Too long
		{"email", "test@example.com", false},
		{"email", "invalid-email", true}, // Invalid email
		{"phone", "1234567890", false},
		{"phone", "123", true}, // Too short
		{"unknown_field", "test", false}, // Default validation
	}
	
	for _, tc := range testCases {
		err := service.ValidatePrivacyCompliance(tc.fieldName, tc.value)
		
		if tc.expectError {
			if err == nil {
				t.Errorf("Expected error for field '%s' with value '%s', but got none", tc.fieldName, tc.value)
			}
		} else {
			if err != nil {
				t.Errorf("Unexpected error for field '%s' with value '%s': %v", tc.fieldName, tc.value, err)
			}
		}
	}
}

func TestPrivacyGuaranteesService_MinimizeData(t *testing.T) {
	cfg := &config.Config{}
	service := NewPrivacyGuaranteesService(cfg)
	
	testCases := []struct {
		fieldName string
		value     string
		expectedMinimization string
	}{
		{"user_id", "test123", "hash"}, // Should be hashed
		{"first_name", "John", "hash"}, // Should be hashed
		{"email", "test@example.com", "hash"}, // Should be hashed
		{"unknown_field", "test", "hash"}, // Default minimization
	}
	
	for _, tc := range testCases {
		result, err := service.MinimizeData(tc.fieldName, tc.value)
		if err != nil {
			t.Errorf("MinimizeData failed for field '%s': %v", tc.fieldName, err)
			continue
		}
		
		// For hash minimization, result should be different from input
		if tc.expectedMinimization == "hash" {
			if result == tc.value {
				t.Errorf("Field '%s' should be hashed, but got original value", tc.fieldName)
			}
			
			// Hash should be 64 characters (SHA-256)
			if len(result) != 64 {
				t.Errorf("Field '%s' hash should be 64 characters, got %d", tc.fieldName, len(result))
			}
		}
	}
}

func TestPrivacyGuaranteesService_DataMinimizationMethods(t *testing.T) {
	cfg := &config.Config{}
	service := NewPrivacyGuaranteesService(cfg)
	
	// Test hashValue
	original := "test123"
	hashed := service.hashValue(original)
	if hashed == original {
		t.Error("Hash should be different from original value")
	}
	if len(hashed) != 64 {
		t.Errorf("Hash should be 64 characters, got %d", len(hashed))
	}
	
	// Test truncateValue
	longValue := strings.Repeat("a", 100)
	truncated := service.truncateValue(longValue, 50)
	if len(truncated) != 53 { // 50 + "..."
		t.Errorf("Truncated value should be 53 characters, got %d", len(truncated))
	}
	if !strings.HasSuffix(truncated, "...") {
		t.Error("Truncated value should end with '...'")
	}
	
	// Test maskValue
	masked := service.maskValue("test123")
	if len(masked) != 7 {
		t.Errorf("Masked value should be 7 characters, got %d", len(masked))
	}
	if !strings.HasPrefix(masked, "t") || !strings.HasSuffix(masked, "3") {
		t.Error("Masked value should keep first and last characters")
	}
}

func TestPrivacyGuaranteesService_CleanupExpiredData(t *testing.T) {
	cfg := &config.Config{}
	service := NewPrivacyGuaranteesService(cfg)
	
	// Store some test data
	testData := []byte("test data")
	key := "test_key"
	
	err := service.SecureStore(key, testData)
	if err != nil {
		t.Fatalf("SecureStore failed: %v", err)
	}
	
	// Verify data exists
	_, err = service.SecureRetrieve(key)
	if err != nil {
		t.Fatalf("SecureRetrieve failed: %v", err)
	}
	
	// Manually expire the data by modifying the access time
	service.securePool.mu.Lock()
	if buffer, exists := service.securePool.pools[key]; exists {
		buffer.accessed = time.Now().Add(-10 * time.Minute) // Expire it
	}
	service.securePool.mu.Unlock()
	
	// Run cleanup
	service.CleanupExpiredData()
	
	// Verify data is cleaned up
	_, err = service.SecureRetrieve(key)
	if err == nil {
		t.Error("Expired data should be cleaned up")
	}
}

func TestPrivacyGuaranteesService_PrivacyAuditLogger(t *testing.T) {
	cfg := &config.Config{}
	service := NewPrivacyGuaranteesService(cfg)
	
	// Test logging an event
	service.auditLogger.LogEvent(
		"test_event",
		"Test privacy audit event",
		"user123",
		"req456",
		[]string{"field1", "field2"},
	)
	
	// Get audit events
	events := service.auditLogger.GetAuditEvents()
	if len(events) != 1 {
		t.Errorf("Expected 1 audit event, got %d", len(events))
	}
	
	event := events[0]
	if event.EventType != "test_event" {
		t.Errorf("Expected event type 'test_event', got '%s'", event.EventType)
	}
	
	if event.UserID != "user123" {
		t.Errorf("Expected user ID 'user123', got '%s'", event.UserID)
	}
	
	if event.RequestID != "req456" {
		t.Errorf("Expected request ID 'req456', got '%s'", event.RequestID)
	}
	
	if len(event.DataFields) != 2 {
		t.Errorf("Expected 2 data fields, got %d", len(event.DataFields))
	}
}

func TestPrivacyGuaranteesService_GetPrivacyStats(t *testing.T) {
	cfg := &config.Config{}
	service := NewPrivacyGuaranteesService(cfg)
	
	stats := service.GetPrivacyStats()
	
	expectedKeys := []string{
		"service_status",
		"secure_pool_size",
		"audit_events_count",
		"max_retention_time",
		"hash_identifiers",
		"truncate_long_values",
		"mask_sensitive_fields",
	}
	
	for _, key := range expectedKeys {
		if _, exists := stats[key]; !exists {
			t.Errorf("Stats should contain key: %s", key)
		}
	}
	
	if stats["service_status"] != "active" {
		t.Errorf("Expected service_status 'active', got %v", stats["service_status"])
	}
	
	if stats["secure_pool_size"] != 0 {
		t.Errorf("Expected secure_pool_size 0, got %v", stats["secure_pool_size"])
	}
}

func TestPrivacyGuaranteesService_HealthCheck(t *testing.T) {
	cfg := &config.Config{}
	service := NewPrivacyGuaranteesService(cfg)
	
	ctx := context.Background()
	err := service.HealthCheck(ctx)
	
	if err != nil {
		t.Errorf("Health check should not fail: %v", err)
	}
}

func TestPrivacyGuaranteesService_ContainsSensitivePattern(t *testing.T) {
	cfg := &config.Config{}
	service := NewPrivacyGuaranteesService(cfg)
	
	testCases := []struct {
		value     string
		sensitive bool
	}{
		{"password123", true},
		{"secret_key", true},
		{"credit_card", true},
		{"ssn_number", true},
		{"normal_data", false},
		{"test123", false},
		{"", false},
	}
	
	for _, tc := range testCases {
		result := service.containsSensitivePattern(tc.value)
		if result != tc.sensitive {
			t.Errorf("For value '%s', expected sensitive=%t, got %t", tc.value, tc.sensitive, result)
		}
	}
}

func TestPrivacyGuaranteesService_ValidatePattern(t *testing.T) {
	cfg := &config.Config{}
	service := NewPrivacyGuaranteesService(cfg)
	
	testCases := []struct {
		value   string
		pattern string
		valid   bool
	}{
		{"test@example.com", `^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`, true},
		{"invalid-email", `^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`, false},
		{"test123", "unknown_pattern", true}, // Default case
	}
	
	for _, tc := range testCases {
		result := service.validatePattern(tc.value, tc.pattern)
		if result != tc.valid {
			t.Errorf("For value '%s' with pattern '%s', expected valid=%t, got %t", tc.value, tc.pattern, tc.valid, result)
		}
	}
} 