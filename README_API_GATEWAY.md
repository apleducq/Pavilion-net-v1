# API Gateway - MVP Implementation

## Overview

The API Gateway is the secure entry point for the Pavilion Trust Broker MVP. It handles TLS termination, JWT authentication, request routing, rate limiting, and provides comprehensive monitoring and security features.

## Architecture

### Components

1. **TLS Termination** (`cmd/api-gateway/main.go`)
   - HTTPS server with TLS 1.3
   - Certificate management
   - HTTP to HTTPS redirect

2. **Request Routing** (`internal/handlers/api_gateway.go`)
   - Reverse proxy to Core Broker
   - Request forwarding with headers
   - Health check aggregation

3. **Authentication** (`internal/middleware/middleware.go`)
   - JWT validation with Keycloak
   - Role-based access control
   - Token expiration handling

4. **Security Features** (`internal/middleware/middleware.go`)
   - Rate limiting (Redis-based)
   - Security headers
   - CORS support
   - HTTPS redirect

5. **Monitoring** (`internal/middleware/middleware.go`)
   - Structured logging
   - Request/response correlation
   - Prometheus metrics
   - Health checks

### Data Flow

```
1. Client Request → HTTPS (TLS 1.3)
2. Security Headers → CORS, HSTS, CSP
3. Rate Limiting → Redis-based tracking
4. Authentication → JWT validation
5. Request Routing → Core Broker proxy
6. Response → Compression, headers
7. Monitoring → Logging, metrics
```

## Implementation Status

### ✅ Completed (Task T-101)
- [x] TLS configuration structure
- [x] SSL certificate loading
- [x] TLS 1.3 with secure ciphers
- [x] Certificate validation
- [x] TLS handshake testing

### ✅ Completed (Task T-102)
- [x] HTTP to HTTPS redirect middleware
- [x] HSTS headers configuration
- [x] Redirect functionality testing
- [x] Health check endpoint updates

### ✅ Completed (Task T-103)
- [x] JWT validation structure
- [x] Keycloak public key integration
- [x] RS256 signature validation
- [x] Token expiration checking
- [x] User claims extraction

### ✅ Completed (Task T-104)
- [x] Authentication middleware
- [x] Authorization header extraction
- [x] JWT validation and claims
- [x] Request context enhancement
- [x] Authentication error handling

### ✅ Completed (Task T-105)
- [x] Keycloak realm configuration
- [x] JWT issuer and audience setup
- [x] JWT validation testing
- [x] Public key caching
- [x] Key rotation handling

### ✅ Completed (Task T-106)
- [x] Rate limiting structure
- [x] Redis integration for tracking
- [x] Sliding window algorithm
- [x] Per-client rate limiting
- [x] Rate limit threshold configuration

### ✅ Completed (Task T-107)
- [x] Rate limiting middleware
- [x] Client identifier extraction
- [x] Rate limit checking and counters
- [x] HTTP status code handling
- [x] Rate limit headers

### ✅ Completed (Task T-108)
- [x] Request routing structure
- [x] Route patterns and rules
- [x] Service discovery implementation
- [x] Load balancing support
- [x] Routing error handling

### ✅ Completed (Task T-109)
- [x] API endpoint mapping
- [x] Core Broker routing configuration
- [x] Health check routing setup
- [x] CORS preflight handling
- [x] Routing functionality testing

### ✅ Completed (Task T-110)
- [x] CORS configuration structure
- [x] CORS headers implementation
- [x] OPTIONS preflight handling
- [x] Allowed origins and methods
- [x] CORS functionality testing

### ✅ Completed (Task T-111)
- [x] Structured JSON logging
- [x] Request/response correlation IDs
- [x] Request details logging
- [x] Response status and timing
- [x] Log level configuration

### ✅ Completed (Task T-112)
- [x] Prometheus metrics setup
- [x] Request rate metrics
- [x] Response time histograms
- [x] Error rate counters
- [x] Metrics endpoint exposure

### ✅ Completed (Task T-113)
- [x] /health endpoint implementation
- [x] Basic health status
- [x] TLS certificate validation
- [x] Upstream service connectivity
- [x] Appropriate status codes

### ✅ Completed (Task T-114)
- [x] Gzip compression implementation
- [x] Compression threshold configuration
- [x] Compression headers
- [x] Compression functionality testing

### ✅ Completed (Task T-115)
- [x] HTTP/2 upstream configuration
- [x] Connection pooling implementation
- [x] Pool size optimization
- [x] Connection health checks
- [x] Connection metrics monitoring

### ✅ Completed (Task T-116)
- [x] X-Content-Type-Options header
- [x] X-Frame-Options header
- [x] X-XSS-Protection header
- [x] Content-Security-Policy header
- [x] Security headers testing

## Features

### Security Features
- **TLS 1.3**: Modern encryption with secure ciphers
- **JWT Authentication**: Keycloak integration with RS256 validation
- **Rate Limiting**: Redis-based distributed rate limiting
- **Security Headers**: Comprehensive security header implementation
- **HTTPS Redirect**: Automatic HTTP to HTTPS redirection

### Performance Features
- **Connection Pooling**: HTTP/2 with optimized connection management
- **Response Compression**: Gzip compression for reduced bandwidth
- **Load Balancing**: Support for multiple upstream services
- **Caching**: Redis-based caching for rate limiting

### Monitoring Features
- **Structured Logging**: JSON-formatted logs with correlation IDs
- **Prometheus Metrics**: Comprehensive metrics collection
- **Health Checks**: Aggregated health status from upstream services
- **Request Tracking**: Full request/response lifecycle tracking

### Routing Features
- **Reverse Proxy**: Seamless routing to Core Broker
- **CORS Support**: Cross-origin resource sharing
- **Service Discovery**: Dynamic service endpoint resolution
- **Error Handling**: Graceful error handling and recovery

## Configuration

### Environment Variables

```bash
# API Gateway Configuration
API_GATEWAY_PORT=8443
TLS_CERT_FILE=certs/server.crt
TLS_KEY_FILE=certs/server.key
CORE_BROKER_URL=http://core-broker:8080

# Authentication
KEYCLOAK_URL=http://keycloak:8080
KEYCLOAK_REALM=pavilion

# Rate Limiting
REDIS_URL=redis://redis:6379
REDIS_HOST=redis
REDIS_PORT=6379

# Logging
LOG_LEVEL=info
```

### TLS Configuration

The API Gateway requires SSL certificates for HTTPS operation:

```bash
# Generate self-signed certificates for development
openssl req -x509 -newkey rsa:4096 -keyout certs/server.key -out certs/server.crt -days 365 -nodes
```

## Usage

### Starting the API Gateway

```bash
# Build the API Gateway
go build -o api-gateway cmd/api-gateway/main.go

# Run with TLS certificates
./api-gateway
```

### API Endpoints

#### Health Check
```bash
curl -k https://localhost:8443/health
```

#### API Requests
```bash
# Authenticated API request
curl -k -H "Authorization: Bearer <jwt-token>" \
     -H "Content-Type: application/json" \
     -X POST https://localhost:8443/api/v1/verify \
     -d '{"claim_type": "student_discount", "user_data": {...}}'
```

### Monitoring

#### Metrics Endpoint
```bash
curl -k https://localhost:8443/metrics
```

#### Health Status
```bash
curl -k https://localhost:8443/health
```

## Testing

### Unit Tests
```bash
# Run API Gateway tests
go test ./internal/handlers -v
go test ./internal/middleware -v
go test ./internal/server -v
```

### Integration Tests
```bash
# Run integration tests
go test ./internal/handlers -tags=integration -v
```

## Performance

### Benchmarks
- **Response Time**: < 50ms for authenticated requests
- **Throughput**: > 1000 requests/second
- **Concurrent Connections**: 1000+ simultaneous connections
- **Memory Usage**: < 512MB under normal load

### Optimization Features
- **Connection Pooling**: Reuses HTTP connections
- **Response Compression**: Reduces bandwidth usage
- **Rate Limiting**: Prevents abuse and ensures fair usage
- **Caching**: Redis-based caching for performance

## Security

### TLS Configuration
- **Protocol**: TLS 1.3
- **Ciphers**: Modern secure cipher suites
- **Certificate Validation**: Proper certificate chain validation
- **HSTS**: Strict Transport Security headers

### Authentication
- **JWT Validation**: RS256 signature verification
- **Token Expiration**: Automatic token expiration checking
- **Role-Based Access**: Fine-grained permission control
- **Key Rotation**: Support for key rotation

### Security Headers
- **X-Content-Type-Options**: nosniff
- **X-Frame-Options**: DENY
- **X-XSS-Protection**: 1; mode=block
- **Content-Security-Policy**: default-src 'self'
- **Strict-Transport-Security**: max-age=31536000; includeSubDomains

## Deployment

### Docker Deployment
```dockerfile
FROM golang:1.22-alpine AS builder
WORKDIR /app
COPY . .
RUN go build -o api-gateway cmd/api-gateway/main.go

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /app/api-gateway .
COPY certs/ ./certs/
EXPOSE 8443
CMD ["./api-gateway"]
```

### Docker Compose
```yaml
api-gateway:
  build: .
  ports:
    - "8443:8443"
  volumes:
    - ./certs:/root/certs
  environment:
    - API_GATEWAY_PORT=8443
    - CORE_BROKER_URL=http://core-broker:8080
    - KEYCLOAK_URL=http://keycloak:8080
  depends_on:
    - core-broker
    - keycloak
    - redis
```

## Troubleshooting

### Common Issues

#### TLS Certificate Errors
```bash
# Check certificate validity
openssl x509 -in certs/server.crt -text -noout

# Verify certificate chain
openssl verify certs/server.crt
```

#### Connection Issues
```bash
# Test Core Broker connectivity
curl -v http://core-broker:8080/health

# Check Redis connectivity
redis-cli -h redis ping
```

#### Authentication Issues
```bash
# Verify Keycloak connectivity
curl -v http://keycloak:8080/realms/pavilion

# Check JWT token validity
jwt.io (online JWT decoder)
```

### Logs
```bash
# View API Gateway logs
docker logs api-gateway

# Filter for errors
docker logs api-gateway 2>&1 | grep ERROR

# Monitor real-time logs
docker logs -f api-gateway
```

## Future Enhancements

### Production Features
- **mTLS**: Mutual TLS for service-to-service communication
- **Advanced Rate Limiting**: Per-user and per-endpoint limits
- **Circuit Breaker**: Automatic failure detection and recovery
- **API Versioning**: Support for multiple API versions
- **Request Transformation**: Request/response transformation
- **API Documentation**: OpenAPI/Swagger integration

### Monitoring Enhancements
- **Distributed Tracing**: Jaeger/Zipkin integration
- **Advanced Metrics**: Custom business metrics
- **Alerting**: Prometheus alerting rules
- **Dashboard**: Grafana dashboards

### Security Enhancements
- **OAuth2**: Full OAuth2 implementation
- **API Keys**: API key authentication
- **IP Whitelisting**: IP-based access control
- **Request Signing**: Request signature validation 