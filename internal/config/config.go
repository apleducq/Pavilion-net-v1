package config

import (
	"os"
	"strconv"
	"strings"
	"time"
)

// RedisConfig holds Redis-specific configuration
type RedisConfig struct {
	Host     string
	Port     int
	Password string
	DB       int
	TTL      int // Time to live in seconds
}

// Config holds all configuration for the Core Broker service
type Config struct {
	// Service Configuration
	Port string
	Env  string

	// API Gateway Configuration
	APIGatewayPort string
	TLSCertFile    string
	TLSKeyFile     string
	CoreBrokerURL  string

	// Authentication
	KeycloakURL  string
	KeycloakRealm string

	// Policy Service
	OPAURL     string
	OPATimeout time.Duration

	// DP Communication
	DPConnectorURL string
	DPConnectorToken string
	DPTimeout      time.Duration

	// Cache Configuration
	RedisURL   string
	CacheTTL   time.Duration
	Redis      RedisConfig

	// Audit Configuration
	AuditDBURL     string
	AuditBatchSize int

	// Privacy/PPRL Configuration
	BloomFilterSize     int
	BloomFilterHashCount int
	BloomFilterFalsePositiveRate float64
	PhoneticEncodingEnabled bool

	// Logging
	LogLevel string
}

// Load loads configuration from environment variables
func Load() (*Config, error) {
	cfg := &Config{
		// Service Configuration
		Port: getEnv("PAVILION_PORT", "8080"),
		Env:  getEnv("PAVILION_ENV", "development"),

		// API Gateway Configuration
		APIGatewayPort: getEnv("API_GATEWAY_PORT", "8443"),
		TLSCertFile:    getEnv("TLS_CERT_FILE", "certs/server.crt"),
		TLSKeyFile:     getEnv("TLS_KEY_FILE", "certs/server.key"),
		CoreBrokerURL:  getEnv("CORE_BROKER_URL", "http://core-broker:8080"),

		// Authentication
		KeycloakURL:   getEnv("KEYCLOAK_URL", "http://keycloak:8080"),
		KeycloakRealm: getEnv("KEYCLOAK_REALM", "pavilion"),

		// Policy Service
		OPAURL:     getEnv("OPA_URL", "http://opa:8181"),
		OPATimeout: getDurationEnv("OPA_TIMEOUT", 5*time.Second),

		// DP Communication
		DPConnectorURL: getEnv("DP_CONNECTOR_URL", "http://dp-connector:8080"),
		DPConnectorToken: getEnv("DP_CONNECTOR_TOKEN", ""), // Default empty string
		DPTimeout:      getDurationEnv("DP_TIMEOUT", 30*time.Second),

		// Cache Configuration
		RedisURL: getEnv("REDIS_URL", "redis://redis:6379"),
		CacheTTL: getDurationEnv("CACHE_TTL", 90*24*time.Hour), // 90 days
		Redis: RedisConfig{
			Host:     getEnv("REDIS_HOST", "redis"),
			Port:     getIntEnv("REDIS_PORT", 6379),
			Password: getEnv("REDIS_PASSWORD", ""),
			DB:       getIntEnv("REDIS_DB", 0),
			TTL:      getIntEnv("REDIS_TTL", int(90*24*3600)), // 90 days in seconds
		},

		// Audit Configuration
		AuditDBURL:     getEnv("AUDIT_DB_URL", "postgres://audit:5432"),
		AuditBatchSize: getIntEnv("AUDIT_BATCH_SIZE", 100),

		// Privacy/PPRL Configuration
		BloomFilterSize:     getIntEnv("BLOOM_FILTER_SIZE", 1000000),
		BloomFilterHashCount: getIntEnv("BLOOM_FILTER_HASH_COUNT", 7),
		BloomFilterFalsePositiveRate: getFloat64Env("BLOOM_FILTER_FALSE_POSITIVE_RATE", 0.01),
		PhoneticEncodingEnabled: getBoolEnv("PHONETIC_ENCODING_ENABLED", false),

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

// getFloat64Env gets a float64 environment variable or returns a default value
func getFloat64Env(key string, defaultValue float64) float64 {
	if value := os.Getenv(key); value != "" {
		if floatValue, err := strconv.ParseFloat(value, 64); err == nil {
			return floatValue
		}
	}
	return defaultValue
}

// getBoolEnv gets a boolean environment variable or returns a default value
func getBoolEnv(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if boolValue, err := strconv.ParseBool(value); err == nil {
			return boolValue
		}
	}
	return defaultValue
} 