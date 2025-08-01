---
title: "Core Broker Requirements - MVP"
project: "Pavilion Trust Broker"
owner: "Technical Lead"
status: draft
version: 0.1.0
last_updated: 2025-01-08
scope: mvp
---

# Core Broker Requirements - MVP

## Overview

The Core Broker is the central orchestrator of the Pavilion Trust Broker MVP. It handles the end-to-end flow of verification requests from Relying Parties (RPs) to Data Providers (DPs), manages policy enforcement, and ensures privacy-preserving operations.

## Functional Requirements

### FR-001: Request Processing
**Priority**: High  
**Epic**: E-001: Core Verification Flow

The system must process verification requests from RPs and return appropriate responses.

**Acceptance Criteria**:
- [ ] Accept POST requests to `/api/v1/verify`
- [ ] Validate JWT authentication tokens
- [ ] Parse and validate request payload
- [ ] Return structured JSON responses
- [ ] Handle malformed requests gracefully

**User Story**: US-001: As an RP developer, I want to submit verification requests so that I can verify user eligibility.

### FR-002: Policy Enforcement
**Priority**: High  
**Epic**: E-001: Core Verification Flow

The system must enforce access policies before processing verification requests.

**Acceptance Criteria**:
- [ ] Query OPA policy service for authorization decisions
- [ ] Enforce RP permissions for specific claim types
- [ ] Validate DP access permissions
- [ ] Log policy decisions for audit
- [ ] Return appropriate error codes for policy violations

**User Story**: US-002: As a compliance officer, I want policy enforcement so that unauthorized access is prevented.

### FR-003: Privacy-Preserving Record Linkage
**Priority**: High  
**Epic**: E-002: Privacy Protection

The system must perform record linkage without exposing raw PII.

**Acceptance Criteria**:
- [ ] Implement Bloom-filter PPRL algorithm
- [ ] Hash identifiers before transmission to DPs
- [ ] Support fuzzy matching for names/addresses
- [ ] Maintain privacy guarantees (no raw PII exposure)
- [ ] Log linkage attempts for audit purposes

**User Story**: US-003: As a privacy engineer, I want PPRL so that user data remains protected during verification.

### FR-004: DP Communication
**Priority**: High  
**Epic**: E-001: Core Verification Flow

The system must communicate with DP Connector to retrieve verification data.

**Acceptance Criteria**:
- [ ] Send pull-job requests to DP Connector
- [ ] Handle DP response timeouts (30s default)
- [ ] Parse DP responses and extract verification results
- [ ] Cache DP responses for performance
- [ ] Handle DP unavailability gracefully

**User Story**: US-004: As a DP admin, I want reliable communication so that verification requests are processed correctly.

### FR-005: Response Generation
**Priority**: High  
**Epic**: E-001: Core Verification Flow

The system must generate appropriate responses to RP requests.

**Acceptance Criteria**:
- [ ] Format responses according to API specification
- [ ] Include verification status and confidence scores
- [ ] Add JWS attestation to responses
- [ ] Include audit trail references
- [ ] Handle multiple DP responses when applicable

**User Story**: US-005: As an RP developer, I want structured responses so that I can process verification results.

### FR-006: Audit Logging
**Priority**: High  
**Epic**: E-003: Audit & Compliance

The system must log all verification activities for audit purposes.

**Acceptance Criteria**:
- [ ] Log all verification requests and responses
- [ ] Include cryptographic hashes for tamper detection
- [ ] Store audit entries in append-only format
- [ ] Generate Merkle proofs for audit entries
- [ ] Support audit log querying and verification

**User Story**: US-006: As a compliance officer, I want comprehensive audit logging so that I can demonstrate compliance.

### FR-007: Caching
**Priority**: Medium  
**Epic**: E-004: Performance Optimization

The system must cache verification results for performance.

**Acceptance Criteria**:
- [ ] Cache successful verification results (TTL: 90 days)
- [ ] Cache DP public keys and configuration
- [ ] Implement cache invalidation on policy changes
- [ ] Monitor cache hit rates and performance
- [ ] Handle cache failures gracefully

**User Story**: US-007: As an RP developer, I want fast response times so that my application remains responsive.

### FR-008: Health Monitoring
**Priority**: Medium  
**Epic**: E-005: Operations

The system must provide health and status information.

**Acceptance Criteria**:
- [ ] Expose `/health` endpoint for health checks
- [ ] Report service status and dependencies
- [ ] Include performance metrics (response times, error rates)
- [ ] Support graceful shutdown
- [ ] Log startup and shutdown events

**User Story**: US-008: As a DevOps engineer, I want health monitoring so that I can ensure service availability.

## Non-Functional Requirements

### NFR-001: Performance
**Priority**: High

- **Response Time**: < 800ms end-to-end for verification flow
- **Throughput**: 100 requests/second per instance
- **Resource Usage**: < 2GB RAM, < 1 CPU core
- **Cache Hit Rate**: > 80% for repeated requests

### NFR-002: Security
**Priority**: High

- **Authentication**: JWT-based authentication required
- **Encryption**: TLS 1.3 for all external communications
- **Data Protection**: No raw PII stored in memory
- **Audit Trail**: All actions logged with cryptographic integrity

### NFR-003: Reliability
**Priority**: High

- **Availability**: 95% uptime during development
- **Error Handling**: Graceful degradation on component failures
- **Recovery**: < 5 minutes for service restart
- **Data Integrity**: Cryptographic verification of all data

### NFR-004: Privacy
**Priority**: High

- **Data Minimization**: Only process necessary data
- **PPRL**: Bloom-filter implementation for record linkage
- **No Raw PII**: Never store or transmit raw personal data
- **Audit Privacy**: Log hashes only, not raw data

### NFR-005: Scalability
**Priority**: Medium

- **Horizontal Scaling**: Support multiple instances
- **Stateless Design**: No local state storage
- **Configuration**: External configuration management
- **Resource Efficiency**: Optimize for container deployment

## Technical Constraints

### MVP Constraints
- **Deployment**: Local Docker Compose environment
- **Database**: Postgres for configuration, Redis for caching
- **Authentication**: Keycloak single-realm
- **Privacy**: Bloom-filter PPRL only (no PSI/ZKP)
- **Audit**: Local Merkle batching (no blockchain)

### Dependencies
- **API Gateway**: For TLS termination and routing
- **Policy Service**: OPA for authorization decisions
- **DP Connector**: For data provider communication
- **Audit Service**: For logging and Merkle proof generation

## Risk Assessment

### High Risk
- **RK-001**: PPRL implementation complexity
- **RK-002**: Performance under load
- **RK-003**: Privacy guarantee verification

### Medium Risk
- **RK-004**: DP communication reliability
- **RK-005**: Cache consistency across instances
- **RK-006**: Audit log performance

### Low Risk
- **RK-007**: Configuration management
- **RK-008**: Health monitoring overhead

## Success Criteria

### MVP Success Metrics
- [ ] End-to-end verification flow works in < 800ms
- [ ] Zero raw PII exposure during operations
- [ ] All verification attempts logged with Merkle proofs
- [ ] Policy enforcement prevents unauthorized access
- [ ] Cache hit rate > 80% for repeated requests
- [ ] Service can handle 100 requests/second
- [ ] Health checks pass consistently
- [ ] Graceful handling of DP unavailability

## TBD Items

### TBD-001: Advanced Privacy Features
**Description**: PSI and ZKP implementation for production
**Impact**: High - Required for production privacy guarantees
**Timeline**: Post-MVP

### TBD-002: Multi-Tenant Support
**Description**: Support for multiple organizations
**Impact**: Medium - Required for production scaling
**Timeline**: Production phase

### TBD-003: Blockchain Integration
**Description**: Public blockchain audit log anchoring
**Impact**: Medium - Required for production audit
**Timeline**: Production phase 