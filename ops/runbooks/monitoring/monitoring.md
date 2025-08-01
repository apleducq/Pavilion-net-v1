# Monitoring Runbook

This runbook covers monitoring and alerting procedures for the B2B Trust Broker platform.

## Monitoring Stack

- **Prometheus** - Metrics collection
- **Grafana** - Visualization and dashboards
- **AlertManager** - Alert routing and notification
- **ELK Stack** - Log aggregation and analysis
- **Jaeger** - Distributed tracing

## Key Metrics

### Application Metrics

- **Response Time**: API endpoint response times
- **Throughput**: Requests per second
- **Error Rate**: Percentage of failed requests
- **Availability**: Service uptime percentage
- **Resource Usage**: CPU, memory, disk usage

### Business Metrics

- **Trust Score Calculations**: Number of trust assessments
- **Policy Evaluations**: Policy engine processing rate
- **Data Processing**: Data connector throughput
- **User Activity**: Admin UI usage patterns

## Dashboard Access

### Production Dashboards

- **Main Dashboard**: https://grafana.company.com/d/main
- **API Gateway**: https://grafana.company.com/d/api-gateway
- **Core Broker**: https://grafana.company.com/d/core-broker
- **Policy Engine**: https://grafana.company.com/d/policy-engine
- **Database**: https://grafana.company.com/d/database

### Development Dashboards

- **Dev Environment**: https://grafana-dev.company.com/d/dev
- **Staging Environment**: https://grafana-staging.company.com/d/staging

## Alert Configuration

### Critical Alerts

- **Service Down**: Any service unavailable for >5 minutes
- **High Error Rate**: Error rate >5% for >10 minutes
- **High Response Time**: P95 response time >2 seconds
- **Database Issues**: Connection failures or high latency
- **Resource Exhaustion**: CPU >90% or memory >90%

### Warning Alerts

- **High Resource Usage**: CPU >80% or memory >80%
- **Increased Error Rate**: Error rate >2% for >5 minutes
- **Slow Response Time**: P95 response time >1 second
- **Disk Space**: Available disk space <20%

## Alert Response Procedures

### 1. Alert Triage

```bash
# Check service status
kubectl get pods -A
kubectl get svc

# Check logs
kubectl logs -l app=core-broker --tail=100
kubectl logs -l app=api-gateway --tail=100
```

### 2. Service Health Check

```bash
# Check endpoints
curl -k https://api-gateway.example.com/health
curl -k https://admin-ui.example.com/health

# Check database
kubectl exec -it db-pod -- psql -c "SELECT 1;"
```

### 3. Resource Investigation

```bash
# Check resource usage
kubectl top pods
kubectl top nodes

# Check events
kubectl get events --sort-by='.lastTimestamp'
```

## Log Analysis

### Log Locations

- **Application Logs**: `/var/log/app/`
- **System Logs**: `/var/log/syslog`
- **Container Logs**: `kubectl logs <pod-name>`
- **Database Logs**: `/var/log/postgresql/`

### Common Log Patterns

```bash
# Search for errors
kubectl logs -l app=core-broker | grep ERROR

# Search for slow queries
kubectl logs -l app=database | grep "slow query"

# Search for authentication failures
kubectl logs -l app=api-gateway | grep "auth failed"
```

## Performance Monitoring

### Key Performance Indicators

1. **API Response Time**
   - Target: <500ms (P95)
   - Alert: >2 seconds (P95)

2. **Database Query Time**
   - Target: <100ms (average)
   - Alert: >1 second (average)

3. **Memory Usage**
   - Target: <80% of limit
   - Alert: >90% of limit

4. **CPU Usage**
   - Target: <70% of limit
   - Alert: >90% of limit

### Performance Investigation

```bash
# Check slow queries
kubectl exec -it db-pod -- psql -c "
SELECT query, mean_time, calls 
FROM pg_stat_statements 
ORDER BY mean_time DESC 
LIMIT 10;
"

# Check connection pool
kubectl exec -it db-pod -- psql -c "
SELECT count(*) as active_connections 
FROM pg_stat_activity 
WHERE state = 'active';
"
```

## Security Monitoring

### Security Events

- **Authentication Failures**: Multiple failed login attempts
- **Authorization Violations**: Access denied events
- **Data Access Patterns**: Unusual data access patterns
- **Network Anomalies**: Unexpected network connections

### Security Investigation

```bash
# Check authentication logs
kubectl logs -l app=api-gateway | grep "auth"

# Check authorization logs
kubectl logs -l app=policy-engine | grep "access denied"

# Check network policies
kubectl get networkpolicies
kubectl describe networkpolicy default-deny
```

## Incident Response

### Escalation Matrix

1. **Level 1** (0-15 minutes): On-call engineer
2. **Level 2** (15-30 minutes): Senior engineer
3. **Level 3** (30+ minutes): Engineering manager

### Communication Channels

- **Slack**: #platform-alerts
- **Email**: alerts@company.com
- **Phone**: +1-555-0123 (emergency only)

## Maintenance Procedures

### Regular Maintenance

- **Daily**: Review alert history
- **Weekly**: Update dashboards
- **Monthly**: Review and tune alert thresholds
- **Quarterly**: Update monitoring documentation

### Monitoring Updates

```bash
# Update Prometheus configuration
kubectl apply -f k8s/monitoring/prometheus-config.yaml

# Update Grafana dashboards
kubectl apply -f k8s/monitoring/grafana-dashboards.yaml

# Update alert rules
kubectl apply -f k8s/monitoring/alertmanager-config.yaml
```

## Documentation

- Update monitoring runbook based on incidents
- Record new alert patterns
- Document troubleshooting procedures
- Share lessons learned with team 