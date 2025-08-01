---
title: "API Gateway Design - MVP"
project: "Pavilion Trust Broker"
owner: "Technical Lead"
status: draft
version: 0.1.0
last_updated: 2025-01-08
scope: mvp
---

# API Gateway Design - MVP

## Architecture Overview
The API Gateway serves as the single entry point for all external traffic to the Pavilion Trust Broker. It handles TLS termination, authentication, rate limiting, and request routing to internal services.

## System Context
```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   RP Client     │    │   API Gateway   │    │   Core Broker   │
│   (External)    │───▶│   (TLS/JWT)     │───▶│   (Orchestrator)│
└─────────────────┘    └─────────────────┘    └─────────────────┘
                              │
                              ▼
                       ┌─────────────────┐
                       │   Policy Engine │
                       │   (Rules)       │
                       └─────────────────┘
```

## Component Architecture

### 1. Request Handler
**Purpose**: Process incoming HTTP requests
**Responsibilities**:
- Parse HTTP headers and body
- Extract client information
- Route to appropriate internal service
- Handle CORS preflight requests

**Design**:
```go
type RequestHandler struct {
    router     *mux.Router
    middleware []Middleware
    services   ServiceRegistry
}
```

### 2. TLS Manager
**Purpose**: Handle SSL/TLS termination
**Responsibilities**:
- Load SSL certificates
- Terminate TLS connections
- Redirect HTTP to HTTPS
- Validate certificate chains

**Design**:
```go
type TLSManager struct {
    certFile   string
    keyFile    string
    caCertFile string
    config     *tls.Config
}
```

### 3. JWT Validator
**Purpose**: Validate JWT tokens from Keycloak
**Responsibilities**:
- Parse JWT tokens
- Validate signatures
- Check expiration
- Extract user claims

**Design**:
```go
type JWTValidator struct {
    publicKey  *rsa.PublicKey
    issuer     string
    audience   string
    clockSkew  time.Duration
}
```

### 4. Rate Limiter
**Purpose**: Prevent abuse and ensure fair usage
**Responsibilities**:
- Track request counts per client
- Enforce rate limits
- Return appropriate HTTP status codes
- Log rate limit violations

**Design**:
```go
type RateLimiter struct {
    redis      *redis.Client
    limits     map[string]Limit
    window     time.Duration
}
```

### 5. Request Router
**Purpose**: Route requests to appropriate internal services
**Responsibilities**:
- Match URL patterns
- Apply routing rules
- Handle service discovery
- Manage load balancing

**Design**:
```go
type RequestRouter struct {
    routes     []Route
    services   ServiceRegistry
    balancer   LoadBalancer
}
```

### 6. Response Handler
**Purpose**: Process and return HTTP responses
**Responsibilities**:
- Add security headers
- Handle CORS headers
- Compress responses
- Log response metrics

**Design**:
```go
type ResponseHandler struct {
    corsConfig CORSConfig
    compress   bool
    logger     Logger
}
```

## Data Flows

### 1. Verification Request Flow
```
1. Client → Gateway (HTTPS)
2. Gateway validates TLS certificate
3. Gateway extracts JWT from Authorization header
4. Gateway validates JWT with Keycloak
5. Gateway applies rate limiting
6. Gateway routes to Core Broker
7. Core Broker processes request
8. Gateway returns response to client
```

### 2. Health Check Flow
```
1. Load balancer → Gateway (/health)
2. Gateway returns 200 OK
3. No authentication required
```

### 3. CORS Preflight Flow
```
1. Browser → Gateway (OPTIONS)
2. Gateway returns CORS headers
3. No authentication required
```

## Security Design

### TLS Configuration
- **Protocol**: TLS 1.3
- **Ciphers**: ECDHE-RSA-AES256-GCM-SHA384
- **Certificate**: Self-signed for MVP
- **HSTS**: Enabled with max-age=31536000

### JWT Validation
- **Algorithm**: RS256
- **Issuer**: Keycloak realm
- **Audience**: Pavilion API
- **Clock Skew**: 30 seconds

### Rate Limiting
- **Window**: 1 minute sliding window
- **Limits**: 100 requests/minute per client
- **Storage**: Redis for distributed tracking

### Security Headers
```
X-Content-Type-Options: nosniff
X-Frame-Options: DENY
X-XSS-Protection: 1; mode=block
Strict-Transport-Security: max-age=31536000
```

## Performance Design

### Caching Strategy
- **JWT Validation**: Cache public keys for 1 hour
- **Rate Limiting**: Redis with 1-minute TTL
- **Routing**: In-memory route table

### Connection Pooling
- **Upstream**: HTTP/2 connections to internal services
- **Downstream**: HTTP/1.1 for client compatibility
- **Pool Size**: 100 connections per service

### Monitoring
- **Metrics**: Request rate, latency, error rate
- **Logging**: Structured JSON logs
- **Health Checks**: /health endpoint

## Error Handling

### HTTP Status Codes
- **200**: Success
- **400**: Bad Request (invalid JWT, malformed request)
- **401**: Unauthorized (missing/invalid JWT)
- **429**: Too Many Requests (rate limit exceeded)
- **500**: Internal Server Error
- **502**: Bad Gateway (upstream service unavailable)

### Error Response Format
```json
{
  "error": "rate_limit_exceeded",
  "message": "Too many requests",
  "retry_after": 60
}
```

## Configuration Management

### Environment Variables
```bash
GATEWAY_PORT=8080
GATEWAY_TLS_CERT=/certs/server.crt
GATEWAY_TLS_KEY=/certs/server.key
KEYCLOAK_URL=http://keycloak:8080
KEYCLOAK_REALM=pavilion
REDIS_URL=redis://redis:6379
CORE_BROKER_URL=http://core-broker:8080
```

### Configuration File
```yaml
gateway:
  port: 8080
  tls:
    cert: /certs/server.crt
    key: /certs/server.key
  
  jwt:
    issuer: http://keycloak:8080/realms/pavilion
    audience: pavilion-api
    clock_skew: 30s
  
  rate_limiting:
    window: 1m
    limit: 100
  
  cors:
    allowed_origins: ["*"]
    allowed_methods: ["GET", "POST", "OPTIONS"]
    allowed_headers: ["*"]
```

## Deployment Considerations

### Docker Configuration
```dockerfile
FROM golang:1.22-alpine
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o gateway ./cmd/gateway
EXPOSE 8080
CMD ["./gateway"]
```

### Docker Compose
```yaml
api-gateway:
  build: .
  ports:
    - "8080:8080"
  volumes:
    - ./certs:/certs
  environment:
    - GATEWAY_PORT=8080
    - KEYCLOAK_URL=http://keycloak:8080
  depends_on:
    - keycloak
    - redis
    - core-broker
```

### Health Checks
- **Readiness**: /health endpoint returns 200
- **Liveness**: Process responds to SIGTERM
- **Startup**: TLS certificate loaded successfully

## Observability

### Metrics
- Request rate per endpoint
- Response time percentiles
- Error rate by status code
- Rate limit violations
- JWT validation failures

### Logging
- Structured JSON logs
- Request/response correlation
- Client IP and user agent
- Upstream service responses

### Tracing
- Distributed tracing headers
- Span correlation across services
- Performance bottleneck identification 