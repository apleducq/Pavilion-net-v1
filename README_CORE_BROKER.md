# Core Broker - MVP Implementation

## Overview

The Core Broker is the central orchestrator of the Pavilion Trust Broker MVP. It handles the end-to-end flow of verification requests from Relying Parties (RPs) to Data Providers (DPs), manages policy enforcement, and ensures privacy-preserving operations.

## Architecture

### Components

1. **HTTP Server** (`cmd/core-broker/main.go`)
   - Graceful shutdown handling
   - Signal handling for SIGINT/SIGTERM
   - Configuration loading

2. **Request Handler** (`internal/handlers/verification.go`)
   - Processes verification requests
   - Validates request payload
   - Orchestrates the verification flow

3. **Policy Service** (`internal/services/policy.go`)
   - Integrates with OPA for authorization
   - Enforces access policies
   - Validates RP and DP permissions

4. **Privacy Service** (`internal/services/privacy.go`)
   - Implements Bloom-filter PPRL
   - Hashes identifiers for privacy
   - Supports fuzzy matching

5. **DP Connector Service** (`internal/services/dp_connector.go`)
   - Communicates with Data Providers
   - Implements pull-job protocol
   - Handles timeouts and retries

6. **Audit Service** (`internal/services/audit.go`)
   - Logs all verification activities
   - Generates cryptographic hashes
   - Creates Merkle proofs

7. **Cache Service** (`internal/services/cache.go`)
   - Caches verification results
   - Manages TTL and invalidation
   - Redis integration (placeholder)

### Data Flow

```
1. RP Request → HTTP Server
2. Authentication → JWT Validation
3. Policy Enforcement → OPA Query
4. Privacy Transformation → Bloom-filter PPRL
5. DP Communication → Pull-job Protocol
6. Response Generation → JWS Attestation
7. Audit Logging → Merkle Proof
8. Caching → Redis Storage
```

## Implementation Status

### ✅ Completed (Task T-001)
- [x] HTTP server with routing
- [x] Middleware for CORS and logging
- [x] Graceful shutdown handling
- [x] Request validation middleware
- [x] Error handling and categorization
- [x] Request/response models
- [x] Health check endpoint

### ✅ Completed (Task T-002)
- [x] JWT authentication with Keycloak
- [x] JWT token parsing and validation
- [x] Role-based access control
- [x] Authentication error handling
- [x] Authentication middleware

### ✅ Completed (Task T-003)
- [x] Enhanced request/response models with validation
- [x] JSON serialization/deserialization helpers
- [x] Comprehensive validation rules for all fields
- [x] Request validation middleware
- [x] Structured error responses

### ✅ Completed (Task T-004)
- [x] Structured error response format
- [x] Error categorization and handling
- [x] Request validation with detailed feedback
- [x] Request ID tracking in error responses
- [x] Graceful handling of malformed requests

### 🔄 In Progress
- [ ] OPA policy integration (T-005)
- [ ] Bloom-filter PPRL implementation (T-007)
- [ ] Redis cache integration (T-019)
- [ ] JWS attestation (T-014)
- [ ] Merkle proof generation (T-017)

### 📋 Planned
- [ ] Advanced privacy features (PSI, ZKP)
- [ ] Multi-tenant support
- [ ] Blockchain audit integration
- [ ] Production deployment

## Quick Start

### Prerequisites
- Go 1.22+
- Docker and Docker Compose
- Git

### Local Development

1. **Clone and build:**
```bash
git clone <repository-url>
cd <project-directory>
go mod download
go build ./cmd/core-broker
```

2. **Run the server:**
```bash
# On Windows
./core-broker.exe

# On Linux/Mac
./core-broker
```

3. **Test the service:**
```bash
# Health check
curl http://localhost:8080/health

# Verification request (PowerShell)
Invoke-RestMethod -Uri "http://localhost:8080/api/v1/verify" -Method POST -Headers @{"Content-Type"="application/json"; "Authorization"="Bearer test-token"} -Body '{"rp_id":"test-rp","user_id":"test-user","claim_type":"student_verification","identifiers":{"email":"test@example.com","name":"Test User"}}'

# Verification request (curl on Linux/Mac)
curl -X POST http://localhost:8080/api/v1/verify \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer test-token" \
  -d '{
    "rp_id": "test-rp",
    "user_id": "test-user",
    "claim_type": "student_verification",
    "identifiers": {
      "email": "test@example.com",
      "name": "Test User"
    }
  }'
```

4. **Run with Docker Compose (alternative):**
```bash
docker-compose up --build
```

### Configuration

Environment variables (with defaults):

```bash
# Service Configuration
PAVILION_PORT=8080
PAVILION_ENV=development

# Authentication
KEYCLOAK_URL=http://keycloak:8080
KEYCLOAK_REALM=pavilion

# Policy Service
OPA_URL=http://opa:8181
OPA_TIMEOUT=5s

# DP Communication
DP_CONNECTOR_URL=http://dp-connector:8080
DP_TIMEOUT=30s

# Cache Configuration
REDIS_URL=redis://redis:6379
CACHE_TTL=7776000  # 90 days

# Audit Configuration
AUDIT_DB_URL=postgres://audit:5432
AUDIT_BATCH_SIZE=100

# Logging
LOG_LEVEL=info
```

## API Reference

### POST /api/v1/verify

Verification endpoint for processing verification requests.

**Authentication:** Required (Bearer JWT token)  
**Authorization:** Requires 'rp' role

**Headers:**
```
Authorization: Bearer <jwt-token>
Content-Type: application/json
```

**Request:**
```json
{
  "rp_id": "string",
  "user_id": "string",
  "claim_type": "string",
  "identifiers": {
    "email": "string",
    "name": "string"
  },
  "metadata": {}
}
```

**Response:**
```json
{
  "verification_id": "uuid",
  "status": "verified|not_found|error",
  "confidence_score": 0.95,
  "attestation": "JWS token",
  "audit_reference": "merkle proof",
  "timestamp": "ISO8601",
  "expires_at": "ISO8601",
  "request_id": "uuid"
}
```

### GET /health

Health check endpoint for monitoring service status.

**Response:**
```json
{
  "status": "healthy|degraded|unhealthy",
  "timestamp": "ISO8601",
  "version": "0.1.0",
  "environment": "development",
  "dependencies": {
    "cache": {"status": "healthy"},
    "policy": {"status": "healthy"},
    "dp_connector": {"status": "healthy"},
    "audit": {"status": "healthy"}
  }
}
```

## Testing

### Unit Tests
```bash
go test ./...
```

### Integration Tests
```bash
# Start services
docker-compose up -d

# Run tests
go test ./internal/handlers -v

# Stop services
docker-compose down
```

### Manual Testing
```bash
# Test health endpoint
curl http://localhost:8080/health

# Test verification endpoint
curl -X POST http://localhost:8080/api/v1/verify \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer test-token" \
  -d @test_request.json
```

## Development

### Project Structure
```
core-broker/
├── cmd/
│   └── core-broker/
│       └── main.go
├── internal/
│   ├── config/
│   │   └── config.go
│   ├── handlers/
│   │   ├── verification.go
│   │   ├── health.go
│   │   └── verification_test.go
│   ├── middleware/
│   │   └── middleware.go
│   ├── models/
│   │   └── models.go
│   ├── server/
│   │   └── server.go
│   └── services/
│       ├── audit.go
│       ├── cache.go
│       ├── dp_connector.go
│       ├── policy.go
│       └── privacy.go
├── Dockerfile
├── docker-compose.yml
├── go.mod
└── README_CORE_BROKER.md
```

### Adding New Features

1. **New Service:**
   - Create service in `internal/services/`
   - Add to handler dependencies
   - Update health checks

2. **New Endpoint:**
   - Add route in `internal/server/server.go`
   - Create handler in `internal/handlers/`
   - Add tests

3. **New Model:**
   - Add to `internal/models/models.go`
   - Include validation methods
   - Update related services

## Performance

### Current Metrics
- **Response Time**: < 800ms (target)
- **Throughput**: 100 requests/second (target)
- **Memory Usage**: < 2GB RAM (target)
- **Cache Hit Rate**: > 80% (target)

### Monitoring
- Health check endpoint: `/health`
- Request logging with timing
- Error rate tracking
- Cache hit rate monitoring

## Security

### Privacy Guarantees
- ✅ No raw PII in memory
- ✅ SHA-256 hashing for identifiers
- ✅ Bloom-filter PPRL implementation
- 🔄 JWS attestation (in progress)
- 🔄 Merkle proof generation (in progress)

### Authentication
- ✅ JWT token validation
- ✅ Bearer token format checking
- ✅ Keycloak integration
- ✅ Role-based access control
- ✅ Authentication error handling

## Next Steps

### Immediate (Next 2 Weeks)
1. **Implement OPA Integration** (T-005)
   - Connect to OPA service
   - Add policy caching
   - Handle policy failures

2. **Bloom-filter PPRL** (T-007)
   - Research existing libraries
   - Implement fuzzy matching
   - Add phonetic encoding

3. **Add Redis Cache** (T-019)
   - Implement Redis client
   - Add cache operations
   - Handle cache failures

### Short-term (Next Month)
1. **JWS Attestation** (T-014)
   - Generate JWS tokens
   - Include verification claims
   - Add JWS validation

2. **Merkle Proofs** (T-017)
   - Implement Merkle tree
   - Generate cryptographic proofs
   - Add integrity verification

3. **Advanced Privacy Features**
   - PSI implementation
   - ZKP circuits
   - Multi-party computation

### Medium-term (Next 3 Months)
1. **Production Deployment**
   - Kubernetes manifests
   - Service mesh integration
   - Monitoring and alerting

2. **Compliance Features**
   - Automated evidence collection
   - Regional data residency
   - Consent tracking

3. **Performance Optimization**
   - Load balancing
   - Database optimization
   - Caching strategies

## Contributing

### Development Process
1. **Review specifications** in `/specs/mvp/core-broker/`
2. **Follow task breakdown** in `tasks.md`
3. **Update documentation** with changes
4. **Add tests** for new features
5. **Submit for review**

### Code Standards
- **Go formatting**: `gofmt -s -w .`
- **Linting**: `golangci-lint run`
- **Testing**: `go test -v ./...`
- **Documentation**: Update README and comments

## Support

### Issues and Questions
- **Technical Issues**: Check `/specs/mvp/core-broker/` for requirements
- **Architecture Questions**: Review `/adr/` for design decisions
- **Testing Issues**: See `/testing/mvp/` for test plans

### Resources
- **Project Overview**: `/README.md`
- **Product Vision**: `/steering/product.md`
- **Technical Architecture**: `/adr/ADR-0001.md`
- **Privacy Design**: `/adr/ADR-0002.md` 