# Deployment Runbook

This runbook covers the deployment procedures for the B2B Trust Broker platform.

## Prerequisites

- Access to AWS console and CLI
- kubectl configured for target cluster
- Docker images built and pushed to registry
- Terraform infrastructure deployed
- Database migrations ready

## Pre-Deployment Checklist

- [ ] All tests passing in CI/CD pipeline
- [ ] Security scan completed
- [ ] Performance tests passed
- [ ] Database backup completed
- [ ] Rollback plan prepared
- [ ] Team notified of deployment

## Deployment Process

### 1. Environment Preparation

```bash
# Verify infrastructure status
terraform plan -var-file=environments/production.tfvars

# Check Kubernetes cluster status
kubectl get nodes
kubectl get pods -A
```

### 2. Database Migration

```bash
# Run database migrations
kubectl apply -f k8s/migrations/

# Verify migration status
kubectl logs -l app=migration -f
```

### 3. Application Deployment

```bash
# Deploy core components
kubectl apply -f k8s/core-broker/
kubectl apply -f k8s/api-gateway/
kubectl apply -f k8s/policy-engine/
kubectl apply -f k8s/dp-connector/

# Deploy admin UI
kubectl apply -f k8s/admin-ui/

# Verify deployment
kubectl get pods -l app=core-broker
kubectl get pods -l app=api-gateway
kubectl get pods -l app=policy-engine
kubectl get pods -l app=dp-connector
kubectl get pods -l app=admin-ui
```

### 4. Health Checks

```bash
# Check service health
kubectl get svc
kubectl describe svc core-broker-service

# Verify endpoints
curl -k https://api-gateway.example.com/health
curl -k https://admin-ui.example.com/health
```

### 5. Load Balancer Configuration

```bash
# Update ingress rules
kubectl apply -f k8s/ingress/

# Verify ingress
kubectl get ingress
kubectl describe ingress b2b-trust-broker-ingress
```

## Post-Deployment Verification

### 1. Functional Testing

- [ ] API endpoints responding correctly
- [ ] Admin UI accessible and functional
- [ ] Database connections working
- [ ] Authentication/authorization working
- [ ] Policy engine processing requests

### 2. Performance Monitoring

- [ ] Response times within SLA
- [ ] Resource usage within limits
- [ ] Error rates acceptable
- [ ] Logs showing normal operation

### 3. Security Verification

- [ ] TLS certificates valid
- [ ] Secrets properly mounted
- [ ] Network policies applied
- [ ] RBAC configured correctly

## Rollback Procedure

### Quick Rollback

```bash
# Rollback to previous version
kubectl rollout undo deployment/core-broker
kubectl rollout undo deployment/api-gateway
kubectl rollout undo deployment/policy-engine
kubectl rollout undo deployment/dp-connector
kubectl rollout undo deployment/admin-ui

# Verify rollback
kubectl get pods -A
kubectl get svc
```

### Database Rollback

```bash
# If database changes need rollback
kubectl apply -f k8s/migrations/rollback/

# Verify database state
kubectl exec -it db-pod -- psql -c "SELECT version();"
```

## Troubleshooting

### Common Issues

1. **Pods not starting**
   - Check resource limits
   - Verify image availability
   - Check configuration maps

2. **Service not accessible**
   - Verify service configuration
   - Check ingress rules
   - Validate DNS resolution

3. **Database connection issues**
   - Check database credentials
   - Verify network policies
   - Check database status

### Emergency Contacts

- **Platform Team**: platform@company.com
- **DevOps Team**: devops@company.com
- **On-Call Engineer**: +1-555-0123

## Monitoring and Alerts

- Monitor deployment metrics in Grafana
- Set up alerts for critical failures
- Track performance metrics
- Monitor security events

## Documentation

- Update deployment logs
- Record any issues encountered
- Update runbook based on learnings
- Share lessons learned with team 