---
title: "Core Broker Design - MVP"
project: "Pavilion Trust Broker"
owner: "Technical Lead"
status: draft
version: 0.1.0
last_updated: 2025-01-08
scope: mvp
---

# Core Broker Design - MVP

## Architecture Overview

The Core Broker is designed as a stateless microservice that orchestrates the verification flow between Relying Parties (RPs) and Data Providers (DPs). It implements privacy-preserving record linkage and maintains an immutable audit trail.

## System Context

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   RP Client     │    │   API Gateway   │    │   Core Broker   │
│   (External)    │───▶│   (TLS/JWT)     │───▶│   (Orchestrator)│
└─────────────────┘    └─────────────────┘    └─────────────────┘
                                                        │
                       ┌─────────────────┐    ┌─────────────────┐
                       │   Policy Svc    │    │   Audit Svc     │
                       │   (OPA)         │◀───│   (Postgres)    │
                       └─────────────────┘    └─────────────────┘
                                                        │
                       ┌─────────────────┐    ┌─────────────────┐
                       │   Redis Cache   │    │   DP Connector  │
                       │   (Hot Data)    │◀───│   (Pull Jobs)   │
                       └─────────────────┘    └─────────────────┘
```

## Component Architecture

### Core Components

#### 1. Request Handler
**Purpose**: Process incoming verification requests from RPs

**Responsibilities**:
- Validate JWT authentication tokens
- Parse and validate request payload
- Route requests to appropriate handlers
- Generate structured responses

**Interfaces**:
- `POST /api/v1/verify` - Main verification endpoint
- `GET /health` - Health check endpoint

#### 2. Policy Enforcer
**Purpose**: Enforce access policies before processing requests

**Responsibilities**:
- Query OPA policy service for authorization decisions
- Validate RP permissions for specific claim types
- Check DP access permissions
- Log policy decisions for audit

**Dependencies**:
- OPA Policy Service (HTTP API)
- Policy configuration cache

#### 3. Privacy Engine
**Purpose**: Perform privacy-preserving record linkage

**Responsibilities**:
- Implement Bloom-filter PPRL algorithm
- Hash identifiers before transmission
- Support fuzzy matching for names/addresses
- Maintain privacy guarantees

**Algorithms**:
- Bloom-filter PPRL with configurable parameters
- SHA-256 hashing for identifiers
- Fuzzy matching using phonetic encoding

#### 4. DP Communicator
**Purpose**: Communicate with DP Connector for data retrieval

**Responsibilities**:
- Send pull-job requests to DP Connector
- Handle response timeouts and retries
- Parse DP responses and extract verification results
- Cache DP responses for performance

**Protocol**:
- HTTP/HTTPS communication
- JSON payload format
- Configurable timeouts (default: 30s)

#### 5. Response Generator
**Purpose**: Generate appropriate responses to RP requests

**Responsibilities**:
- Format responses according to API specification
- Include verification status and confidence scores
- Add JWS attestation to responses
- Include audit trail references

**Response Format**:
```json
{
  "verification_id": "uuid",
  "status": "verified|not_found|error",
  "confidence_score": 0.95,
  "attestation": "JWS token",
  "audit_reference": "merkle_proof",
  "timestamp": "ISO8601",
  "expires_at": "ISO8601"
}
```

#### 6. Audit Logger
**Purpose**: Log all verification activities for audit purposes

**Responsibilities**:
- Log all verification requests and responses
- Include cryptographic hashes for tamper detection
- Store audit entries in append-only format
- Generate Merkle proofs for audit entries

**Audit Entry Format**:
```json
{
  "timestamp": "ISO8601",
  "request_id": "uuid",
  "rp_id": "string",
  "dp_id": "string",
  "claim_type": "string",
  "privacy_hash": "sha256",
  "merkle_proof": "base64",
  "policy_decision": "allow|deny"
}
```

#### 7. Cache Manager
**Purpose**: Cache verification results for performance

**Responsibilities**:
- Cache successful verification results (TTL: 90 days)
- Cache DP public keys and configuration
- Implement cache invalidation on policy changes
- Monitor cache hit rates and performance

**Cache Keys**:
- `verification:{rp_id}:{user_hash}:{claim_type}`
- `dp_config:{dp_id}`
- `policy:{policy_id}`

## Data Flows

### 1. Verification Request Flow

```
1. RP sends POST /api/v1/verify
   ├── JWT authentication
   ├── Request validation
   └── Route to Core Broker

2. Core Broker processes request
   ├── Policy enforcement (OPA)
   ├── Privacy-preserving linkage
   ├── DP communication
   ├── Response generation
   └── Audit logging

3. Response returned to RP
   ├── Verification result
   ├── JWS attestation
   └── Audit reference
```

### 2. Privacy-Preserving Record Linkage

```
1. Receive user identifiers from RP
   ├── Hash identifiers (SHA-256)
   ├── Apply Bloom-filter encoding
   └── Prepare for DP transmission

2. Send to DP Connector
   ├── Hashed identifiers only
   ├── No raw PII transmitted
   └── Configurable timeout

3. Process DP response
   ├── Parse verification result
   ├── Apply confidence scoring
   └── Cache successful results
```

### 3. Audit Log Flow

```
1. Log verification request
   ├── Hash request payload
   ├── Include privacy hash
   └── Generate Merkle proof

2. Log verification response
   ├── Hash response payload
   ├── Link to request entry
   └── Update Merkle tree

3. Store in audit database
   ├── Append-only storage
   ├── Cryptographic integrity
   └── Tamper-evident design
```

## Security Design

### Authentication & Authorization
- **JWT Authentication**: Validate tokens from Keycloak
- **Policy Enforcement**: OPA-based authorization decisions
- **Role-Based Access**: RP and DP permission management

### Data Protection
- **TLS 1.3**: All external communications encrypted
- **No Raw PII**: Only hashed identifiers processed
- **Memory Safety**: No sensitive data stored in memory
- **Audit Integrity**: Cryptographic hashing of all operations

### Privacy Guarantees
- **Bloom-filter PPRL**: Privacy-preserving record linkage
- **Data Minimization**: Only necessary data processed
- **Audit Privacy**: Log hashes only, not raw data
- **Consent Tracking**: User consent validation (future)

## Performance Design

### Caching Strategy
- **Verification Results**: 90-day TTL for successful verifications
- **DP Configuration**: Cache DP public keys and settings
- **Policy Rules**: Cache OPA policy decisions
- **Cache Invalidation**: On policy changes or DP updates

### Resource Management
- **Memory Usage**: < 2GB RAM per instance
- **CPU Usage**: < 1 CPU core per instance
- **Connection Pooling**: HTTP client connection reuse
- **Goroutine Management**: Controlled concurrency limits

### Monitoring & Observability
- **Health Checks**: `/health` endpoint with dependency status
- **Metrics**: Request rates, response times, error rates
- **Logging**: Structured JSON logging
- **Tracing**: Distributed tracing for request flows

## Error Handling

### Error Categories
- **Authentication Errors**: Invalid JWT tokens
- **Authorization Errors**: Policy violations
- **Privacy Errors**: PPRL algorithm failures
- **Communication Errors**: DP Connector timeouts
- **System Errors**: Internal service failures

### Error Responses
```json
{
  "error": {
    "code": "AUTHENTICATION_FAILED",
    "message": "Invalid JWT token",
    "timestamp": "ISO8601",
    "request_id": "uuid"
  }
}
```

### Retry Strategy
- **DP Communication**: Exponential backoff (max 3 retries)
- **Policy Queries**: Immediate retry on OPA failures
- **Cache Operations**: Fail gracefully, continue without cache

## Configuration Management

### Environment Variables
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
CACHE_TTL=7776000  # 90 days in seconds

# Audit Configuration
AUDIT_DB_URL=postgres://audit:5432
AUDIT_BATCH_SIZE=100
```

### Configuration Validation
- **Startup Validation**: Verify all required configuration
- **Runtime Validation**: Validate configuration changes
- **Default Values**: Sensible defaults for development
- **Environment-Specific**: Different configs per environment

## Deployment Considerations

### Container Design
- **Single Binary**: Go application with all dependencies
- **Health Checks**: HTTP health check endpoint
- **Graceful Shutdown**: Handle SIGTERM signals
- **Resource Limits**: Memory and CPU constraints

### Local Development
- **Docker Compose**: All services in single environment
- **Hot Reload**: Development mode with file watching
- **Debug Logging**: Verbose logging for troubleshooting
- **Mock Services**: Optional mock DP Connector

### Production Readiness
- **Stateless Design**: No local state storage
- **Horizontal Scaling**: Multiple instances support
- **Configuration**: External configuration management
- **Monitoring**: Comprehensive observability 