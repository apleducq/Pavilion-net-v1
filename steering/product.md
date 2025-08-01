---
title: "Product Vision & Strategy"
project: "Pavilion Trust Broker"
owner: "Product Manager"
status: draft
version: 0.1.0
last_updated: 2025-01-08
scope: all
---

# Pavilion Trust Broker - Product Vision

## Mission Statement

**Privacy-first eligibility verification at web-scale.**

Enable privacy‑first, consent‑driven verification between organisations so people can prove facts about themselves **without exposing their data**, and businesses can collaborate **without brittle one‑off integrations**.

## What Pavilion Is

A **B2B trust broker**: one hub integration for Relying Parties (RPs) to verify eligibility/credentials with Data Providers (DPs), using verifiable credentials (VCs), privacy‑preserving record linkage (PPRL), and—where it adds value—**zero‑knowledge** proofs (ZK) and private set intersection (PSI). Pavilion **minimises data centralisation**; it orchestrates requests, policies, and proofs, with a tamper‑evident audit log.

## Primary Outcomes

- **Reduce integration cost/time** by 80%+ versus bespoke point‑to‑point
- **Lower fraud & abuse** with adaptive trust signals
- **Unlock cross‑sell** via reusable, consented credentials across ecosystems

## Core Personas

### Data Provider Admin
- **Needs**: Easy onboarding, secure data sharing, compliance reporting
- **Pain Points**: Complex integrations, data exposure risks, audit overhead
- **Success Metrics**: Onboarding time < 10 minutes, zero data breaches

### Relying Party Developer
- **Needs**: Simple API integration, fast response times, reliable verification
- **Pain Points**: Multiple DP integrations, inconsistent APIs, slow responses
- **Success Metrics**: Integration time < 1 week, response time < 3s

### Compliance Officer
- **Needs**: Audit trails, policy enforcement, regulatory compliance
- **Pain Points**: Manual compliance checks, data residency issues, consent tracking
- **Success Metrics**: Automated compliance, regional data residency, consent audit trails

### End User
- **Needs**: Privacy protection, seamless verification, control over data sharing
- **Pain Points**: Repeated data sharing, lack of transparency, privacy concerns
- **Success Metrics**: Zero raw PII exposure, user consent control, transparency

## Success Metrics

### MVP Success Metrics
- **Time-to-yes**: < 5s (cold path), < 200ms (cache hit)
- **DP onboarding**: < 10 minutes (manual script)
- **Install & run time**: < 20 minutes on clean machine
- **E2E demo**: claim from RP → policy check → DP decision → response ≤ 800ms
- **Auditability**: Every decision produces Merkle-verifiable proof locally

### Production Success Metrics
- **Time-to-yes**: < 3s (cold path), < 50ms (cache hit)
- **DP onboarding**: < 10 minutes (automated)
- **False-positive match rate**: < 0.01%
- **Availability**: 99.9% uptime
- **Regional compliance**: 100% data residency adherence

## MVP vs Production Scope

| Feature | MVP | Production |
|---------|-----|------------|
| **Deployment** | Local Docker Desktop | Multi-region K8s + Istio |
| **Authentication** | Keycloak single-realm | Keycloak multi-realm + OPA sidecars |
| **Data Storage** | Postgres (configs), Redis (cache) | AuroraDB + Polygon blockchain audit |
| **Privacy Engine** | Bloom-filter PPRL lib | PSI + ZKP circuits (circom) |
| **Storage** | Local S3-compatible (MinIO) | AWS S3 w/ bucket-level policies |
| **CI/CD** | GitHub Actions | ArgoCD + Terraform Cloud |
| **Observability** | Prometheus + Grafana | Add Tempo traces, Loki logs |
| **Compliance** | Manual checklist | Automated evidence collection |
| **Multi-tenancy** | Single tenant | Multi-tenant with isolation |
| **VC Issuance** | Self-signed | External KMS integration |
| **Audit Log** | Local Merkle batching | Public blockchain anchoring |

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

## Risk Assessment

| Risk Level | Description | Mitigation |
|------------|-------------|------------|
| **High** | Data breach, privacy violation | Zero-knowledge proofs, minimal data storage |
| **Medium** | Service downtime, compliance failure | Multi-region deployment, automated compliance |
| **Low** | Performance degradation, integration complexity | Caching, standardized APIs |

## Revenue Model

- **API Usage**: Per-verification pricing
- **Credential Reuse**: Revenue sharing on reused credentials
- **Enterprise Features**: Advanced compliance, custom integrations
- **Regional Expansion**: Localized deployments and compliance 