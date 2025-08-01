---
title: "Pavilion Trust Broker - Specification Workspace"
project: "Pavilion Trust Broker"
status: draft
version: 0.1.0
last_updated: 2025-01-08
---

# Pavilion Trust Broker - Specification Workspace

## Project Overview

**Privacy-first eligibility verification at web-scale.**

Pavilion is a B2B trust broker that enables privacy‑first, consent‑driven verification between organisations. It lets Relying Parties (RPs) verify eligibility with Data Providers (DPs) using Verifiable Credentials (VCs), Privacy-Preserving Record Linkage (PPRL), Zero-Knowledge Proofs (ZKPs) and Private Set Intersection (PSI).

## Repository Structure

```
/steering/                    # Project governance & strategy
├── product.md               # Vision, metrics, personas
├── structure.md             # Roles, cadence, RACI, ADR flow
└── tech.md                  # Stack & NFR summary

/specs/                      # Technical specifications
├── mvp/                     # MVP scope (local development)
│   ├── core-broker/         # Central orchestrator
│   ├── policy-engine/       # OPA-based authorization
│   ├── dp-connector/        # Data provider integration
│   ├── api-gateway/         # TLS termination & routing
│   └── admin-ui/            # Web interface
└── production/              # Production scope (scaled deployment)
    ├── core-broker/         # Multi-tenant, compliance
    ├── policy-engine/       # Advanced policies, federation
    ├── dp-connector/        # Multi-cloud, advanced protocols
    ├── api-gateway/         # Service mesh, advanced security
    └── admin-ui/            # Enterprise features

/adr/                        # Architecture Decision Records
├── ADR-0001.md             # Microservices Architecture
└── ADR-0002.md             # Privacy-Preserving Technology Stack

/testing/                    # Test plans & cases
├── mvp/                     # MVP testing strategy
└── production/              # Production testing strategy

/ops/                        # Operations & deployment
├── runbooks/               # Operational procedures
└── terraform/              # Infrastructure as Code

/notes/                      # Raw input files (to be mined)
```

## MVP Scope

### Core Features
- **End-to-end verification flow**: RP → API Gateway → Core Broker → DP Connector
- **Privacy-preserving record linkage**: Bloom-filter PPRL algorithm
- **JWT authentication**: Keycloak integration
- **Policy enforcement**: OPA-based authorization
- **Audit logging**: Merkle-verifiable audit trail
- **Local deployment**: Docker Compose environment

### Technology Stack
- **Language**: Go 1.22
- **Runtime**: Docker Compose / KinD
- **Authentication**: Keycloak (single-realm)
- **Data Storage**: Postgres (configs), Redis (cache)
- **Privacy Engine**: Bloom-filter PPRL library
- **Storage**: Local S3-compatible (MinIO)
- **CI/CD**: GitHub Actions
- **Observability**: Prometheus + Grafana

### Success Metrics
- **Response Time**: < 800ms end-to-end (cold), < 200ms (cache hit)
- **Install Time**: < 20 minutes on clean machine
- **Privacy**: Zero raw PII exposure
- **Auditability**: Merkle-verifiable audit trail
- **Throughput**: 100 requests/second

## Production Scope

### Advanced Features
- **Multi-region deployment**: Kubernetes + Istio
- **Advanced privacy**: PSI + ZKP circuits
- **Multi-tenancy**: Isolated tenant environments
- **Compliance**: Automated evidence collection
- **Blockchain audit**: Public audit trail anchoring

### Technology Enhancements
- **Runtime**: Multi-region K8s + Istio
- **Authentication**: Keycloak multi-realm + OPA sidecars
- **Data Storage**: AuroraDB + Polygon blockchain audit
- **Privacy Engine**: PSI + ZKP circuits (circom)
- **Storage**: AWS S3 w/ bucket-level policies
- **CI/CD**: ArgoCD + Terraform Cloud
- **Observability**: Add Tempo traces, Loki logs

## Key Use Cases

### MVP Use Cases
1. **Student Discount Verification** - RP: Event Cinemas; DP: University/MoE
2. **Age-Based Check-in** - RP: Hotels/hostels; Claims: age>=18/21
3. **Club Membership Access** - RP: Event venue; DP: Club CRM
4. **SMB Customer List Verification** - RP: Local merchants; DP: SMB uploaded CSV

### Production Use Cases
1. **Public Transport Concessions** - Student/Senior/Disability verification
2. **KYC Reuse Across Fintechs** - Cross-platform identity verification
3. **Healthcare Provider Verification** - Medical credential verification
4. **Government Benefit Verification** - Social service eligibility checks

## Development Roadmap

### Phase 1: MVP Development (Months 1-3)
- **Week 1-2**: Core Broker implementation
- **Week 3-4**: API Gateway and authentication
- **Week 5-6**: Policy engine and DP Connector
- **Week 7-8**: Audit logging and privacy features
- **Week 9-10**: Admin UI and testing
- **Week 11-12**: Integration testing and documentation

### Phase 2: Production Development (Months 4-12)
- **Months 4-6**: Multi-tenant architecture
- **Months 7-9**: Advanced privacy features (PSI, ZKP)
- **Months 10-12**: Compliance and production deployment

## Getting Started

### Prerequisites
- **OS**: Windows 10/11, macOS 12+, Ubuntu 22.04+
- **Docker**: Docker Desktop with 4GB+ RAM
- **Tools**: curl, Postman, browser for testing

### Quick Start
1. **Clone repository**: `git clone <repo-url>`
2. **Review specifications**: Start with `/steering/product.md`
3. **Understand architecture**: Read `/adr/ADR-0001.md`
4. **Review MVP scope**: Check `/specs/mvp/` directories
5. **Plan development**: Use `/specs/mvp/*/tasks.md` files

## Contributing

### Development Process
1. **Review ADRs**: Understand architectural decisions
2. **Follow specifications**: Implement according to `/specs/` documents
3. **Update documentation**: Keep specs current with implementation
4. **Test thoroughly**: Follow `/testing/` test plans
5. **Submit for review**: Code review and specification review
6. **Update task status**: Mark completed tasks in `/specs/*/tasks.md` files

### Specification Updates
- **Requirements**: Update `/specs/*/requirements.md`
- **Design**: Update `/specs/*/design.md`
- **Tasks**: Update `/specs/*/tasks.md`
- **Architecture**: Create new ADRs in `/adr/`
- **Testing**: Update `/testing/` test plans

## Next Steps

### Immediate Actions (Next 2 Weeks)
1. **Review specifications**: Ensure all documents are complete and accurate
2. **Set up development environment**: Docker Compose for local development
3. **Start Core Broker implementation**: Begin with T-001 (HTTP server)
4. **Establish CI/CD**: GitHub Actions for automated testing
5. **Create test data**: Sample RP and DP data for development

### Short-term Goals (Next Month)
1. **Complete MVP architecture**: All services implemented and integrated
2. **End-to-end testing**: Verify complete verification flow works
3. **Performance validation**: Meet MVP performance requirements
4. **Security review**: Validate privacy and security guarantees
5. **Documentation**: Complete user and developer documentation

### Medium-term Goals (Next 3 Months)
1. **MVP release**: Production-ready MVP for pilot customers
2. **Production planning**: Design production architecture
3. **Compliance preparation**: Begin compliance framework implementation
4. **Team scaling**: Add specialized roles (privacy engineer, security engineer)
5. **Customer feedback**: Gather feedback from pilot implementations

## Contact & Support

- **Product Manager**: For product vision and requirements
- **Technical Lead**: For architecture and technical decisions
- **Security Engineer**: For security and compliance questions
- **Privacy Engineer**: For privacy-preserving technology questions

## License

[License information to be added] 