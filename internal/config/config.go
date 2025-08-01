package config

import (
	"os"
	"strconv"
	"time"
)

// Config holds all configuration for the Core Broker service
type Config struct {
	// Service Configuration
	Port string
	Env  string

	// Authentication
	KeycloakURL  string
	KeycloakRealm string

	// Policy Service
	OPAURL     string
	OPATimeout time.Duration

	// DP Communication
	DPConnectorURL string
	DPTimeout      time.Duration

	// Cache Configuration
	RedisURL   string
	CacheTTL   time.Duration

	// Audit Configuration
	AuditDBURL     string
	AuditBatchSize int

	// Logging
	LogLevel string
}

// Load loads configuration from environment variables
func Load() (*Config, error) {
	cfg := &Config{
		// Service Configuration
		Port: getEnv("PAVILION_PORT", "8080"),
		Env:  getEnv("PAVILION_ENV", "development"),

		// Authentication
		KeycloakURL:   getEnv("KEYCLOAK_URL", "http://keycloak:8080"),
		KeycloakRealm: getEnv("KEYCLOAK_REALM", "pavilion"),

		// Policy Service
		OPAURL:     getEnv("OPA_URL", "http://opa:8181"),
		OPATimeout: getDurationEnv("OPA_TIMEOUT", 5*time.Second),

		// DP Communication
		DPConnectorURL: getEnv("DP_CONNECTOR_URL", "http://dp-connector:8080"),
		DPTimeout:      getDurationEnv("DP_TIMEOUT", 30*time.Second),

		// Cache Configuration
		RedisURL: getEnv("REDIS_URL", "redis://redis:6379"),
		CacheTTL: getDurationEnv("CACHE_TTL", 90*24*time.Hour), // 90 days

		// Audit Configuration
		AuditDBURL:     getEnv("AUDIT_DB_URL", "postgres://audit:5432"),
		AuditBatchSize: getIntEnv("AUDIT_BATCH_SIZE", 100),

		// Logging
		LogLevel: getEnv("LOG_LEVEL", "info"),
	}

	return cfg, nil
}

// getEnv gets an environment variable or returns a default value
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// getIntEnv gets an integer environment variable or returns a default value
func getIntEnv(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

// getDurationEnv gets a duration environment variable or returns a default value
func getDurationEnv(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if duration, err := time.ParseDuration(value); err == nil {
			return duration
		}
	}
	return defaultValue
} 