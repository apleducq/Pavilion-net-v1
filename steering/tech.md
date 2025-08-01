---
title: "Technical Stack & Non-Functional Requirements"
project: "Pavilion Trust Broker"
owner: "Technical Lead"
status: draft
version: 0.1.0
last_updated: 2025-01-08
scope: all
---

# Technical Stack & Non-Functional Requirements

## Technology Stack Overview

### MVP Stack (Local Development)

| Layer | Technology | Purpose | Rationale |
|-------|------------|---------|-----------|
| **Language** | Go 1.22 | Core services | Performance, concurrency, crypto libraries |
| **Runtime** | Docker Compose / KinD | Local deployment | Developer productivity, easy setup |
| **Authentication** | Keycloak (single-realm) | Identity management | Open source, OIDC support |
| **Data Storage** | Postgres (configs), Redis (cache) | Configuration & caching | Reliable, well-understood |
| **Privacy Engine** | Bloom-filter PPRL lib | Record linkage | Simple, effective for MVP |
| **Storage** | Local S3-compatible (MinIO) | File storage | S3 API compatibility |
| **CI/CD** | GitHub Actions | Automation | Free tier, GitHub integration |
| **Observability** | Prometheus + Grafana | Monitoring | Open source, comprehensive |
| **Compliance** | Manual checklist | Audit preparation | MVP simplicity |

### Production Stack (Scaled Deployment)

| Layer | Technology | Purpose | Rationale |
|-------|------------|---------|-----------|
| **Language** | Go 1.22 | Core services | Same as MVP for consistency |
| **Runtime** | Multi-region K8s + Istio | Orchestration & mesh | Scalability, security, traffic management |
| **Authentication** | Keycloak multi-realm + OPA sidecars | Identity & policy | Multi-tenant, externalized policies |
| **Data Storage** | AuroraDB + Polygon blockchain audit | Data & audit | Managed, immutable audit trail |
| **Privacy Engine** | PSI + ZKP circuits (circom) | Advanced privacy | Zero-knowledge proofs, set intersection |
| **Storage** | AWS S3 w/ bucket-level policies | File storage | Enterprise-grade, compliance-ready |
| **CI/CD** | ArgoCD + Terraform Cloud | GitOps + IaC | Declarative, audit-friendly |
| **Observability** | Add Tempo traces, Loki logs | Distributed tracing | Complete observability stack |
| **Compliance** | Automated evidence collection | Regulatory compliance | Continuous compliance monitoring |

## Non-Functional Requirements

### Performance Requirements

#### MVP Performance
- **Response Time**: < 5s (cold path), < 200ms (cache hit)
- **Throughput**: 100 requests/second per instance
- **Latency**: < 800ms end-to-end for verification flow
- **Resource Usage**: < 2GB RAM, < 1 CPU core per service

#### Production Performance
- **Response Time**: < 3s (cold path), < 50ms (cache hit)
- **Throughput**: 10,000 requests/second per region
- **Latency**: < 200ms end-to-end for verification flow
- **Resource Usage**: Auto-scaling based on demand
- **Global Distribution**: < 100ms latency between regions

### Security Requirements

#### Authentication & Authorization
- **MVP**: JWT-based authentication, role-based access control
- **Production**: OIDC federation, fine-grained permissions, OPA policies

#### Data Protection
- **MVP**: TLS 1.3, encrypted data at rest
- **Production**: mTLS, HSM integration, zero-knowledge proofs

#### Privacy Requirements
- **MVP**: Bloom-filter PPRL, minimal data storage
- **Production**: PSI protocols, selective disclosure, ZKPs

### Availability Requirements

#### MVP Availability
- **Uptime**: 95% (development environment)
- **Recovery Time**: < 5 minutes for service restart
- **Backup**: Daily automated backups

#### Production Availability
- **Uptime**: 99.9% (three nines)
- **Recovery Time**: < 30 seconds for failover
- **Backup**: Real-time replication, point-in-time recovery
- **Disaster Recovery**: RTO < 4 hours, RPO < 1 hour

### Scalability Requirements

#### MVP Scalability
- **Concurrent Users**: 100 simultaneous users
- **Data Volume**: 1,000 verifications per day
- **Storage**: 10GB total storage

#### Production Scalability
- **Concurrent Users**: 100,000 simultaneous users
- **Data Volume**: 1,000,000 verifications per day
- **Storage**: Auto-scaling based on usage
- **Global Distribution**: Multi-region deployment

### Compliance Requirements

#### Data Residency
- **MVP**: Local storage only
- **Production**: Regional data residency, cross-border restrictions

#### Audit Requirements
- **MVP**: Local Merkle-verifiable audit log
- **Production**: Public blockchain anchoring, tamper-evident logs

#### Regulatory Compliance
- **MVP**: Manual compliance checklist
- **Production**: Automated compliance monitoring (GDPR, CCPA, etc.)

## Architecture Patterns

### MVP Architecture
```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   RP Client     │    │   API Gateway   │    │   Orchestrator  │
│   (curl/UI)     │───▶│   (TLS/JWT)     │───▶│   (Core Logic)  │
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

### Production Architecture
```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   RP Client     │    │   API Gateway   │    │   Orchestrator  │
│   (Global)      │───▶│   (Istio/Envoy) │───▶│   (K8s Service) │
└─────────────────┘    └─────────────────┘    └─────────────────┘
                                                        │
                       ┌─────────────────┐    ┌─────────────────┐
                       │   Policy Svc    │    │   Audit Svc     │
                       │   (OPA Sidecar) │◀───│   (Blockchain)  │
                       └─────────────────┘    └─────────────────┘
                                                        │
                       ┌─────────────────┐    ┌─────────────────┐
                       │   Privacy Svc   │    │   DP Connector  │
                       │   (PSI/ZKP)     │◀───│   (Multi-Cloud) │
                       └─────────────────┘    └─────────────────┘
```

## Technology Decisions

### Why Go?
- **Performance**: Excellent for concurrent operations
- **Crypto Libraries**: Strong cryptographic support
- **Deployment**: Single binary, easy containerization
- **Community**: Active ecosystem for privacy-preserving tech

### Why Kubernetes + Istio?
- **Scalability**: Horizontal scaling, auto-scaling
- **Security**: mTLS, service mesh security
- **Observability**: Built-in tracing, metrics, logging
- **Multi-region**: Native support for global deployment

### Why OPA?
- **Policy as Code**: Version-controlled policies
- **Flexibility**: Supports complex authorization rules
- **Performance**: Fast policy evaluation
- **Auditability**: Policy decisions are logged

### Why Bloom-filter PPRL?
- **Privacy**: No raw PII exchanged
- **Simplicity**: Easy to implement and understand
- **Performance**: Fast matching algorithms
- **MVP Friendly**: Good starting point for privacy

## Migration Path

### MVP to Production
1. **Container Orchestration**: Docker Compose → Kubernetes
2. **Service Mesh**: Direct calls → Istio mTLS
3. **Privacy Engine**: Bloom-filter → PSI + ZKP
4. **Audit Log**: Local Merkle → Blockchain anchoring
5. **Multi-tenancy**: Single tenant → Multi-tenant
6. **Compliance**: Manual → Automated monitoring

### Risk Mitigation
- **Gradual Migration**: Feature flags for gradual rollout
- **A/B Testing**: Compare old vs new implementations
- **Rollback Strategy**: Quick rollback to previous version
- **Monitoring**: Comprehensive observability during migration

## Future Considerations

### Emerging Technologies
- **Zero-Knowledge Proofs**: Circom, snarkJS integration
- **Decentralized Identity**: DIDComm, Verifiable Credentials
- **Confidential Computing**: Intel SGX, AMD SEV
- **Quantum Resistance**: Post-quantum cryptography

### Scalability Enhancements
- **Edge Computing**: Regional processing nodes
- **CDN Integration**: Global content delivery
- **Database Sharding**: Horizontal data partitioning
- **Event Streaming**: Kafka for high-throughput scenarios 