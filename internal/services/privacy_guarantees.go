package services

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/pavilion-trust/core-broker/internal/config"
)

// PrivacyGuaranteesService handles privacy guarantees and secure memory management
type PrivacyGuaranteesService struct {
	config *config.Config
	// Secure memory pool for sensitive data
	securePool *SecureMemoryPool
	// Privacy validation rules
	validationRules map[string]PrivacyRule
	// Audit logger for privacy events
	auditLogger *PrivacyAuditLogger
	// Data minimization settings
	minimizationSettings *DataMinimizationSettings
}

// SecureMemoryPool provides secure memory management for sensitive data
type SecureMemoryPool struct {
	mu    sync.RWMutex
	pools map[string]*SecureBuffer
}

// SecureBuffer represents a secure memory buffer
type SecureBuffer struct {
	data     []byte
	created  time.Time
	accessed time.Time
	// Flag to track if data has been securely wiped
	wiped bool
}

// PrivacyRule defines validation rules for privacy compliance
type PrivacyRule struct {
	FieldName     string   `json:"field_name"`
	AllowedTypes  []string `json:"allowed_types"`
	MaxLength     int      `json:"max_length"`
	MinLength     int      `json:"min_length"`
	Pattern       string   `json:"pattern,omitempty"`
	Required      bool     `json:"required"`
	Sensitive     bool     `json:"sensitive"`
	Minimization  string   `json:"minimization"` // "hash", "truncate", "mask", "none"
}

// PrivacyAuditLogger handles privacy-related audit logging
type PrivacyAuditLogger struct {
	mu     sync.Mutex
	events []PrivacyAuditEvent
}

// PrivacyAuditEvent represents a privacy audit event
type PrivacyAuditEvent struct {
	Timestamp   string            `json:"timestamp"`
	EventType   string            `json:"event_type"`
	Description string            `json:"description"`
	UserID      string            `json:"user_id,omitempty"`
	RequestID   string            `json:"request_id,omitempty"`
	DataFields  []string          `json:"data_fields,omitempty"`
	Metadata    map[string]string `json:"metadata,omitempty"`
}

// DataMinimizationSettings defines data minimization policies
type DataMinimizationSettings struct {
	// Maximum retention time for sensitive data
	MaxRetentionTime time.Duration
	// Whether to hash identifiers
	HashIdentifiers bool
	// Whether to truncate long values
	TruncateLongValues bool
	// Maximum length for truncated values
	MaxTruncatedLength int
	// Whether to mask sensitive fields
	MaskSensitiveFields bool
	// Mask character for sensitive data
	MaskCharacter string
}

// NewPrivacyGuaranteesService creates a new privacy guarantees service
func NewPrivacyGuaranteesService(cfg *config.Config) *PrivacyGuaranteesService {
	service := &PrivacyGuaranteesService{
		config: cfg,
		securePool: &SecureMemoryPool{
			pools: make(map[string]*SecureBuffer),
		},
		validationRules: make(map[string]PrivacyRule),
		auditLogger: &PrivacyAuditLogger{
			events: make([]PrivacyAuditEvent, 0),
		},
		minimizationSettings: &DataMinimizationSettings{
			MaxRetentionTime:     5 * time.Minute,
			HashIdentifiers:       true,
			TruncateLongValues:   true,
			MaxTruncatedLength:   50,
			MaskSensitiveFields:  true,
			MaskCharacter:        "*",
		},
	}

	// Initialize default privacy rules
	service.initializePrivacyRules()

	return service
}

// initializePrivacyRules sets up default privacy validation rules
func (s *PrivacyGuaranteesService) initializePrivacyRules() {
	s.validationRules = map[string]PrivacyRule{
		"user_id": {
			FieldName:     "user_id",
			AllowedTypes:  []string{"string"},
			MaxLength:     100,
			MinLength:     1,
			Required:      true,
			Sensitive:     true,
			Minimization:  "hash",
		},
		"first_name": {
			FieldName:     "first_name",
			AllowedTypes:  []string{"string"},
			MaxLength:     50,
			MinLength:     1,
			Required:      false,
			Sensitive:     true,
			Minimization:  "hash",
		},
		"last_name": {
			FieldName:     "last_name",
			AllowedTypes:  []string{"string"},
			MaxLength:     50,
			MinLength:     1,
			Required:      false,
			Sensitive:     true,
			Minimization:  "hash",
		},
		"email": {
			FieldName:     "email",
			AllowedTypes:  []string{"string"},
			MaxLength:     255,
			MinLength:     5,
			Pattern:       `^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`,
			Required:      false,
			Sensitive:     true,
			Minimization:  "hash",
		},
		"phone": {
			FieldName:     "phone",
			AllowedTypes:  []string{"string"},
			MaxLength:     20,
			MinLength:     10,
			Required:      false,
			Sensitive:     true,
			Minimization:  "hash",
		},
	}
}

// SecureStore stores sensitive data in secure memory
func (s *PrivacyGuaranteesService) SecureStore(key string, data []byte) error {
	if len(data) == 0 {
		return fmt.Errorf("cannot store empty data")
	}

	// Create a copy of the data to avoid external references
	secureData := make([]byte, len(data))
	copy(secureData, data)

	buffer := &SecureBuffer{
		data:     secureData,
		created:  time.Now(),
		accessed: time.Now(),
		wiped:    false,
	}

	s.securePool.mu.Lock()
	s.securePool.pools[key] = buffer
	s.securePool.mu.Unlock()

	// Log the secure storage event
	s.auditLogger.LogEvent("secure_store", fmt.Sprintf("Securely stored data for key: %s", key), "", "", []string{key})

	return nil
}

// SecureRetrieve retrieves data from secure memory
func (s *PrivacyGuaranteesService) SecureRetrieve(key string) ([]byte, error) {
	s.securePool.mu.RLock()
	buffer, exists := s.securePool.pools[key]
	s.securePool.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("no data found for key: %s", key)
	}

	// Update access time
	buffer.accessed = time.Now()

	// Create a copy to avoid exposing the original buffer
	result := make([]byte, len(buffer.data))
	copy(result, buffer.data)

	// Log the retrieval event
	s.auditLogger.LogEvent("secure_retrieve", fmt.Sprintf("Retrieved data for key: %s", key), "", "", []string{key})

	return result, nil
}

// SecureWipe securely wipes sensitive data from memory
func (s *PrivacyGuaranteesService) SecureWipe(key string) error {
	s.securePool.mu.Lock()
	defer s.securePool.mu.Unlock()

	buffer, exists := s.securePool.pools[key]
	if !exists {
		return fmt.Errorf("no data found for key: %s", key)
	}

	// Securely wipe the data
	s.wipeBuffer(buffer)
	delete(s.securePool.pools, key)

	// Log the wipe event
	s.auditLogger.LogEvent("secure_wipe", fmt.Sprintf("Securely wiped data for key: %s", key), "", "", []string{key})

	return nil
}

// wipeBuffer securely wipes a buffer by overwriting with random data
func (s *PrivacyGuaranteesService) wipeBuffer(buffer *SecureBuffer) {
	if buffer.wiped {
		return
	}

	// Overwrite with random data multiple times
	for i := 0; i < 3; i++ {
		rand.Read(buffer.data)
	}

	// Clear the buffer
	for i := range buffer.data {
		buffer.data[i] = 0
	}

	buffer.wiped = true
}

// CleanupExpiredData removes expired data from secure memory
func (s *PrivacyGuaranteesService) CleanupExpiredData() {
	s.securePool.mu.Lock()
	defer s.securePool.mu.Unlock()

	now := time.Now()
	expiredKeys := make([]string, 0)

	for key, buffer := range s.securePool.pools {
		if now.Sub(buffer.accessed) > s.minimizationSettings.MaxRetentionTime {
			expiredKeys = append(expiredKeys, key)
		}
	}

	for _, key := range expiredKeys {
		buffer := s.securePool.pools[key]
		s.wipeBuffer(buffer)
		delete(s.securePool.pools, key)
		s.auditLogger.LogEvent("auto_cleanup", fmt.Sprintf("Automatically cleaned up expired data for key: %s", key), "", "", []string{key})
	}
}

// ValidatePrivacyCompliance validates data against privacy rules
func (s *PrivacyGuaranteesService) ValidatePrivacyCompliance(fieldName string, value string) error {
	rule, exists := s.validationRules[fieldName]
	if !exists {
		// If no specific rule exists, apply default validation
		return s.validateDefault(fieldName, value)
	}

	// Check required fields
	if rule.Required && value == "" {
		return fmt.Errorf("field '%s' is required", fieldName)
	}

	// Check length constraints
	if rule.MaxLength > 0 && len(value) > rule.MaxLength {
		return fmt.Errorf("field '%s' exceeds maximum length of %d", fieldName, rule.MaxLength)
	}

	if rule.MinLength > 0 && len(value) < rule.MinLength {
		return fmt.Errorf("field '%s' is shorter than minimum length of %d", fieldName, rule.MinLength)
	}

	// Check pattern if specified
	if rule.Pattern != "" {
		// Simple pattern validation (in production, use regex)
		if !s.validatePattern(value, rule.Pattern) {
			return fmt.Errorf("field '%s' does not match required pattern", fieldName)
		}
	}

	return nil
}

// validateDefault applies default validation for fields without specific rules
func (s *PrivacyGuaranteesService) validateDefault(fieldName string, value string) error {
	// Default maximum length
	if len(value) > 1000 {
		return fmt.Errorf("field '%s' exceeds default maximum length", fieldName)
	}

	// Check for potentially sensitive patterns
	if s.containsSensitivePattern(value) {
		s.auditLogger.LogEvent("sensitive_pattern_detected", 
			fmt.Sprintf("Potentially sensitive pattern detected in field: %s", fieldName), 
			"", "", []string{fieldName})
	}

	return nil
}

// validatePattern validates a value against a pattern
func (s *PrivacyGuaranteesService) validatePattern(value, pattern string) bool {
	// Simple pattern validation - in production, use proper regex
	switch pattern {
	case `^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`:
		return strings.Contains(value, "@") && strings.Contains(value, ".")
	default:
		return true
	}
}

// containsSensitivePattern checks if a value contains potentially sensitive patterns
func (s *PrivacyGuaranteesService) containsSensitivePattern(value string) bool {
	sensitivePatterns := []string{
		"password", "secret", "key", "token", "credential",
		"ssn", "social", "security", "number",
		"credit", "card", "cc", "cvv",
		"passport", "license", "id",
	}

	valueLower := strings.ToLower(value)
	for _, pattern := range sensitivePatterns {
		if strings.Contains(valueLower, pattern) {
			return true
		}
	}

	return false
}

// MinimizeData applies data minimization techniques
func (s *PrivacyGuaranteesService) MinimizeData(fieldName string, value string) (string, error) {
	rule, exists := s.validationRules[fieldName]
	if !exists {
		// Apply default minimization
		return s.applyDefaultMinimization(value), nil
	}

	switch rule.Minimization {
	case "hash":
		return s.hashValue(value), nil
	case "truncate":
		return s.truncateValue(value, s.minimizationSettings.MaxTruncatedLength), nil
	case "mask":
		return s.maskValue(value), nil
	case "none":
		return value, nil
	default:
		return s.applyDefaultMinimization(value), nil
	}
}

// applyDefaultMinimization applies default data minimization
func (s *PrivacyGuaranteesService) applyDefaultMinimization(value string) string {
	if s.minimizationSettings.HashIdentifiers {
		return s.hashValue(value)
	}
	if s.minimizationSettings.TruncateLongValues && len(value) > s.minimizationSettings.MaxTruncatedLength {
		return s.truncateValue(value, s.minimizationSettings.MaxTruncatedLength)
	}
	if s.minimizationSettings.MaskSensitiveFields {
		return s.maskValue(value)
	}
	return value
}

// hashValue creates a hash of the value
func (s *PrivacyGuaranteesService) hashValue(value string) string {
	// Use SHA-256 for hashing
	hash := sha256.Sum256([]byte(value))
	return hex.EncodeToString(hash[:])
}

// truncateValue truncates a value to the specified length
func (s *PrivacyGuaranteesService) truncateValue(value string, maxLength int) string {
	if len(value) <= maxLength {
		return value
	}
	return value[:maxLength] + "..."
}

// maskValue masks sensitive data
func (s *PrivacyGuaranteesService) maskValue(value string) string {
	if len(value) <= 2 {
		return strings.Repeat(s.minimizationSettings.MaskCharacter, len(value))
	}
	
	// Keep first and last character, mask the rest
	first := value[:1]
	last := value[len(value)-1:]
	middle := strings.Repeat(s.minimizationSettings.MaskCharacter, len(value)-2)
	
	return first + middle + last
}

// LogEvent logs a privacy audit event
func (pl *PrivacyAuditLogger) LogEvent(eventType, description, userID, requestID string, dataFields []string) {
	pl.mu.Lock()
	defer pl.mu.Unlock()

	event := PrivacyAuditEvent{
		Timestamp:   time.Now().Format(time.RFC3339),
		EventType:   eventType,
		Description: description,
		UserID:      userID,
		RequestID:   requestID,
		DataFields:  dataFields,
		Metadata: map[string]string{
			"service": "privacy_guarantees",
		},
	}

	pl.events = append(pl.events, event)

	// Keep only the last 1000 events to prevent memory issues
	if len(pl.events) > 1000 {
		pl.events = pl.events[len(pl.events)-1000:]
	}
}

// GetAuditEvents returns privacy audit events
func (pl *PrivacyAuditLogger) GetAuditEvents() []PrivacyAuditEvent {
	pl.mu.Lock()
	defer pl.mu.Unlock()

	events := make([]PrivacyAuditEvent, len(pl.events))
	copy(events, pl.events)
	return events
}

// GetPrivacyStats returns statistics about privacy guarantees
func (s *PrivacyGuaranteesService) GetPrivacyStats() map[string]interface{} {
	s.securePool.mu.RLock()
	poolSize := len(s.securePool.pools)
	s.securePool.mu.RUnlock()

	s.auditLogger.mu.Lock()
	eventCount := len(s.auditLogger.events)
	s.auditLogger.mu.Unlock()

	return map[string]interface{}{
		"service_status": "active",
		"secure_pool_size": poolSize,
		"audit_events_count": eventCount,
		"max_retention_time": s.minimizationSettings.MaxRetentionTime.String(),
		"hash_identifiers": s.minimizationSettings.HashIdentifiers,
		"truncate_long_values": s.minimizationSettings.TruncateLongValues,
		"mask_sensitive_fields": s.minimizationSettings.MaskSensitiveFields,
	}
}

// HealthCheck checks if the privacy guarantees service is healthy
func (s *PrivacyGuaranteesService) HealthCheck(ctx context.Context) error {
	// Test secure memory operations
	testData := []byte("test")
	err := s.SecureStore("health_test", testData)
	if err != nil {
		return fmt.Errorf("privacy guarantees health check failed: %w", err)
	}

	retrieved, err := s.SecureRetrieve("health_test")
	if err != nil {
		return fmt.Errorf("privacy guarantees health check failed: %w", err)
	}

	if string(retrieved) != "test" {
		return fmt.Errorf("privacy guarantees health check failed: data corruption detected")
	}

	err = s.SecureWipe("health_test")
	if err != nil {
		return fmt.Errorf("privacy guarantees health check failed: %w", err)
	}

	// Test privacy validation
	err = s.ValidatePrivacyCompliance("user_id", "test123")
	if err != nil {
		return fmt.Errorf("privacy guarantees health check failed: %w", err)
	}

	return nil
} 