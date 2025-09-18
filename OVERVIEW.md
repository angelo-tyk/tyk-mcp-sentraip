# Technical Overview - Tyk MCP SentraIP

This document provides a comprehensive technical overview of the Tyk MCP SentraIP system architecture, components, and design decisions.

## Table of Contents

- [System Architecture](#system-architecture)
- [Component Details](#component-details)
- [Data Flow](#data-flow)
- [Technology Stack](#technology-stack)
- [Security Architecture](#security-architecture)
- [Observability Strategy](#observability-strategy)
- [Scalability Design](#scalability-design)
- [Design Decisions](#design-decisions)
- [Integration Patterns](#integration-patterns)

## System Architecture

The Tyk MCP SentraIP system implements a modern, cloud-native architecture that bridges AI capabilities with enterprise API management and threat intelligence. The system follows microservices principles with clear separation of concerns and well-defined interfaces.

```
┌─────────────────────────────────────────────────────────────────────┐
│                           External Layer                            │
├─────────────────┬─────────────────┬─────────────────┬───────────────┤
│   Claude AI     │   SentraIP API  │   Client Apps   │   Monitoring  │
│   (Anthropic)   │  (Threat Intel) │                 │     Tools     │
└─────────────────┴─────────────────┴─────────────────┴───────────────┘
                           │                    │
                           ▼                    ▼
┌─────────────────────────────────────────────────────────────────────┐
│                        Load Balancer                               │
│                    (Kubernetes Service)                           │
└─────────────────────────────────────────────────────────────────────┘
                           │
                           ▼
┌─────────────────────────────────────────────────────────────────────┐
│                      Tyk Gateway Cluster                          │
│  ┌───────────────┐  ┌───────────────┐  ┌───────────────┐        │
│  │ Tyk Gateway   │  │ Tyk Gateway   │  │ Tyk Gateway   │        │
│  │   Instance    │  │   Instance    │  │   Instance    │        │
│  │               │  │               │  │               │        │
│  │ Go Plugins:   │  │ Go Plugins:   │  │ Go Plugins:   │        │
│  │ • OAuth       │  │ • OAuth       │  │ • OAuth       │        │
│  │ • OTEL        │  │ • OTEL        │  │ • OTEL        │        │
│  │ • MCP Tools   │  │ • MCP Tools   │  │ • MCP Tools   │        │
│  └───────────────┘  └───────────────┘  └───────────────┘        │
└─────────────────────────────────────────────────────────────────────┘
                           │
                           ▼
┌─────────────────────────────────────────────────────────────────────┐
│                     Service Mesh Layer                            │
│  ┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐ │
│  │ Claude MCP      │    │ Redis Cache     │    │ OTEL Collector │ │
│  │ Client Service  │    │ (Sessions &     │    │ (Traces &      │ │
│  │                 │    │  Token Cache)   │    │  Metrics)      │ │
│  │ • Chat API      │    │                 │    │                │ │
│  │ • Tool Executor │    │                 │    │                │ │
│  │ • Context Mgmt  │    │                 │    │                │ │
│  └─────────────────┘    └─────────────────┘    └─────────────────┘ │
└─────────────────────────────────────────────────────────────────────┘
```

### Core Components

1. **Tyk Gateway Cluster**: API gateway with custom Go plugins
2. **Claude MCP Client**: Service for AI integration and tool execution
3. **Redis Cache**: Session storage and OAuth token caching
4. **OpenTelemetry Collector**: Observability data collection and export
5. **Configuration Layer**: Kubernetes ConfigMaps and Secrets

## Component Details

### Tyk Gateway

The Tyk Gateway serves as the central API management layer with three custom Go plugins:

#### sentraip_oauth.go
- **Purpose**: Automated OAuth2 client credentials flow for SentraIP
- **Features**:
  - Thread-safe token caching with automatic refresh
  - 60-second buffer for token expiration
  - Error handling and retry logic
  - Request header injection
- **Key Functions**:
  ```go
  func SentraIPOAuthMiddleware(rw http.ResponseWriter, r *http.Request)
  func getValidToken(clientID, clientSecret string) (string, error)
  ```

#### tyk_otel_enhancer.go
- **Purpose**: Rich OpenTelemetry instrumentation and context propagation
- **Features**:
  - Request/response correlation
  - Claude AI request/response analysis
  - SentraIP threat intelligence context
  - Performance categorization
  - Error analysis and classification
- **Key Functions**:
  ```go
  func TykOTELPreMiddleware(rw http.ResponseWriter, r *http.Request)
  func TykOTELPostMiddleware(rw http.ResponseWriter, r *http.Request, res *http.Response)
  ```

#### tyk_mcp_tools.go
- **Purpose**: Model Context Protocol tool registry and execution
- **Features**:
  - Tool discovery and listing
  - Parameterized tool execution
  - Input validation and schema enforcement
  - Response normalization
- **Available Tools**:
  - `sentraip_threat_check`: IP/domain reputation analysis
  - `tyk_api_analytics`: API usage metrics and performance data
  - `claude_context_search`: Conversation history and context retrieval

### Claude MCP Client

A dedicated Go service that bridges Claude AI with the MCP tool ecosystem:

#### Architecture
```go
type ClaudeRequest struct {
    Model     string    `json:"model"`
    MaxTokens int       `json:"max_tokens"`
    Messages  []Message `json:"messages"`
}

type MCPChatRequest struct {
    Message string `json:"message"`
}

type MCPChatResponse struct {
    Response    string                 `json:"response"`
    ToolsCalled []string              `json:"tools_called,omitempty"`
    Context     map[string]interface{} `json:"context,omitempty"`
    Timestamp   string                `json:"timestamp"`
}
```

#### Key Features
- RESTful API for chat interactions
- Tool call detection and routing
- Context management and session handling
- Integration with Tyk Gateway for tool execution

### Redis Cache Layer

Provides high-performance caching for:

#### OAuth Token Management
```
Key Pattern: oauth:token:{client_id}
Value: {
  "access_token": "...",
  "expires_at": "2024-01-15T10:30:00Z",
  "token_type": "Bearer"
}
TTL: Based on token expiration
```

#### Session Storage
```
Key Pattern: session:{session_id}
Value: {
  "user_id": "...",
  "api_key": "...",
  "permissions": [...],
  "created_at": "..."
}
TTL: 3600 seconds (configurable)
```

### OpenTelemetry Collector

Centralized observability pipeline with multiple receivers and exporters:

#### Configuration
```yaml
receivers:
  otlp:           # From Tyk Gateway and MCP Client
  prometheus:     # Metrics scraping
  
processors:
  batch:          # Batching for efficiency
  resource:       # Resource attribution
  attributes:     # Context enrichment

exporters:
  jaeger:         # Distributed tracing
  prometheus:     # Metrics storage
  logging:        # Debug output
```

## Data Flow

### Request Processing Flow

```
1. Client Request
   │
   ▼
2. Load Balancer
   │
   ▼
3. Tyk Gateway (Pre-Middleware)
   ├── OTEL Span Creation
   ├── OAuth Token Injection (if SentraIP)
   └── Request Context Setup
   │
   ▼
4. Upstream Service
   ├── SentraIP API (threat intelligence)
   ├── Claude AI API (chat/completions)
   └── Internal MCP Tools
   │
   ▼
5. Tyk Gateway (Post-Middleware)
   ├── Response Analysis
   ├── Performance Metrics
   ├── OTEL Span Completion
   └── Context Enrichment
   │
   ▼
6. Client Response
```

### MCP Tool Execution Flow

```
1. Client → Claude MCP Client (/chat)
   │
   ▼
2. Message Analysis & Tool Detection
   │
   ▼
3. Tool Execution Request → Tyk Gateway (/mcp/call/{tool})
   │
   ▼
4. Tool Parameter Validation
   │
   ▼
5. Backend Service Integration
   ├── SentraIP API (threat check)
   ├── Tyk Analytics (API metrics)
   └── Context Store (conversation history)
   │
   ▼
6. Response Aggregation & Formatting
   │
   ▼
7. Client Response with Tool Results
```

### OAuth Token Flow

```
1. Request requires SentraIP access
   │
   ▼
2. Check Redis token cache
   │
   ├── Valid Token Found → Inject & Continue
   │
   └── No/Expired Token
       │
       ▼
3. OAuth Token Request
   ├── Client Credentials Grant
   ├── Basic Auth Header
   └── POST /oauth/token
   │
   ▼
4. Cache Token in Redis
   ├── Set TTL (expires_in - 60s)
   └── Return to request flow
```

## Technology Stack

### Core Technologies

| Component | Technology | Version | Purpose |
|-----------|------------|---------|---------|
| **API Gateway** | Tyk Gateway | 5.0+ | API management, routing, plugins |
| **Languages** | Go | 1.21+ | High-performance plugins and services |
| **AI Platform** | Claude AI | API v1 | Large language model integration |
| **Threat Intel** | SentraIP | API v1 | Real-time threat intelligence |
| **Observability** | OpenTelemetry | 1.21+ | Distributed tracing and metrics |
| **Cache/Storage** | Redis | 7+ | Session storage and token caching |
| **Orchestration** | Kubernetes | 1.21+ | Container orchestration |

### Deployment Technologies

| Layer | Technology | Purpose |
|-------|------------|---------|
| **Containerization** | Docker | Application packaging |
| **Registry** | GCR/ECR | Container image storage |
| **Service Mesh** | Native K8s | Service-to-service communication |
| **Load Balancing** | K8s Services | Traffic distribution |
| **Configuration** | ConfigMaps/Secrets | Environment configuration |
| **Monitoring** | Jaeger, Prometheus | Observability backend |

### Development Tools

- **Build System**: Go modules, Docker multi-stage builds
- **CI/CD**: Kubernetes deployment scripts
- **Monitoring**: Structured logging with logrus
- **Testing**: Go testing framework, HTTP client testing

## Security Architecture

### Authentication & Authorization

#### Multi-Layer Security Model

```
┌─────────────────────────────────────────┐
│              Client Layer               │
│  • API Keys                             │
│  • Bearer Tokens                        │
│  • Request Signatures                   │
└─────────────────────────────────────────┘
                    │
                    ▼
┌─────────────────────────────────────────┐
│              Gateway Layer              │
│  • Request Validation                   │
│  • Rate Limiting                        │
│  • OAuth2 Client Credentials            │
└─────────────────────────────────────────┘
                    │
                    ▼
┌─────────────────────────────────────────┐
│             Service Layer               │
│  • Service-to-Service Auth              │
│  • Network Policies                     │
│  • Pod Security Standards               │
└─────────────────────────────────────────┘
```

#### OAuth2 Implementation

The system implements OAuth2 client credentials flow with:

- **Automatic token refresh** with 60-second expiration buffer
- **Thread-safe token caching** to prevent race conditions  
- **Secure credential storage** in Kubernetes Secrets
- **Token injection** into upstream requests

#### Network Security

- **TLS Termination** at load balancer level
- **Inter-service communication** over internal cluster network
- **Network Policies** to restrict pod-to-pod communication
- **Secret Management** with Kubernetes native secrets

### Data Protection

#### Sensitive Data Handling

1. **API Keys**: Stored in Kubernetes Secrets, never logged
2. **OAuth Tokens**: Cached in Redis with appropriate TTL
3. **Request/Response Data**: Filtered in observability pipeline
4. **Personal Information**: Not stored or cached

#### Compliance Considerations

- **Data Retention**: Configurable TTL for cached data
- **Audit Logging**: All API calls traced through OpenTelemetry
- **Access Control**: RBAC for Kubernetes resources
- **Encryption**: TLS for all external communications

## Observability Strategy

### Three Pillars Implementation

#### 1. Distributed Tracing

**OpenTelemetry Integration**:
- Automatic span creation for all API requests
- Context propagation across service boundaries
- Custom attributes for business logic context
- Integration with Jaeger for trace visualization

**Key Trace Attributes**:
```go
// Request Context
attribute.String("tyk.api_id", spec.APIID)
attribute.String("http.method", r.Method)
attribute.String("client.ip", getClientIP(r))

// Business Context
attribute.String("mcp.tool", toolName)
attribute.Int("claude.input_tokens", tokens)
attribute.Int("sentraip.risk_score", score)

// Performance Context
attribute.Int64("request.duration_ms", duration)
attribute.String("performance.category", category)
```

#### 2. Metrics Collection

**Prometheus Integration**:
- Request rate, duration, and error metrics
- Business-specific metrics (threat scores, tool usage)
- Infrastructure metrics (CPU, memory, network)
- Custom metrics for SLA monitoring

**Key Metric Types**:
```
# Counter metrics
tyk_requests_total{api_id="claude-ai", status="200"}
mcp_tool_executions_total{tool="sentraip_threat_check"}

# Histogram metrics  
tyk_request_duration_seconds{api_id="sentraip"}
oauth_token_refresh_duration_seconds

# Gauge metrics
active_sessions_count
cached_tokens_count
```

#### 3. Structured Logging

**Log Structure**:
```json
{
  "timestamp": "2024-01-15T10:30:00Z",
  "level": "info",
  "service": "tyk-gateway",
  "trace_id": "abc123...",
  "span_id": "def456...",
  "message": "OAuth token injected successfully",
  "context": {
    "api_id": "sentraip-threat-intel",
    "client_id": "client_123",
    "token_expires": "2024-01-15T11:30:00Z"
  }
}
```

### Monitoring Stack

```
Client Apps → Tyk Gateway → OpenTelemetry Collector
                    ↓              ↓
            Application Logs → Jaeger (Traces)
                               ↓
                         Prometheus (Metrics)
                               ↓
                         Grafana (Dashboards)
                               ↓
                         AlertManager (Alerts)
```

## Scalability Design

### Horizontal Scaling Patterns

#### Stateless Service Design

All components are designed to be stateless with external state storage:

- **Tyk Gateway**: Stateless with Redis for sessions
- **Claude MCP Client**: Stateless with conversation context in requests
- **OpenTelemetry Collector**: Stateless data pipeline

#### Auto-scaling Configuration

```yaml
apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
metadata:
  name: tyk-gateway-hpa
spec:
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: tyk-gateway
  minReplicas: 3
  maxReplicas: 20
  metrics:
  - type: Resource
    resource:
      name: cpu
      target:
        type: Utilization
        averageUtilization: 70
  - type: Resource
    resource:
      name: memory
      target:
        type: Utilization
        averageUtilization: 80
```

### Performance Optimization

#### Caching Strategy

1. **OAuth Token Caching**: Redis with TTL
2. **API Response Caching**: Tyk built-in cache (optional)
3. **DNS Caching**: Kubernetes CoreDNS
4. **Connection Pooling**: HTTP keep-alive and connection reuse

#### Resource Management

```yaml
resources:
  requests:
    memory: "512Mi"
    cpu: "500m"
  limits:
    memory: "2Gi" 
    cpu: "2000m"
```

### Data Flow Optimization

#### Async Processing

- **Batch Processing**: OpenTelemetry batch processor
- **Background Tasks**: OAuth token refresh
- **Queue Processing**: Redis pub/sub for notifications

#### Load Balancing

- **Service-level**: Kubernetes Services with round-robin
- **Pod-level**: Kubernetes Deployments with replica sets
- **Request-level**: Tyk load balancing algorithms

## Design Decisions

### Architectural Choices

#### 1. Go Plugins vs. Lua Scripts

**Decision**: Use Go plugins for Tyk middleware
**Rationale**:
- Better performance for CPU-intensive operations (OAuth, OTEL)
- Access to full Go ecosystem and libraries
- Type safety and compile-time error checking
- Better integration with observability tools

#### 2. Dedicated MCP Client vs. Gateway Integration

**Decision**: Separate Claude MCP Client service
**Rationale**:
- Separation of concerns (AI logic vs. API management)
- Independent scaling of AI workloads
- Easier testing and development
- Reduced complexity in gateway plugins

#### 3. Redis vs. In-Memory Caching

**Decision**: External Redis cache
**Rationale**:
- Shared state across gateway instances
- Persistence across pod restarts
- Better memory management
- Production-ready high availability

#### 4. OpenTelemetry vs. Custom Logging

**Decision**: OpenTelemetry for observability
**Rationale**:
- Industry standard for observability
- Vendor-neutral approach
- Rich context propagation
- Integration with multiple backends

### Integration Patterns

#### Circuit Breaker Pattern

For external service calls (SentraIP, Claude AI):
```go
// Implemented in Go plugins
func callExternalService(url string, timeout time.Duration) error {
    client := &http.Client{Timeout: timeout}
    
    // Circuit breaker logic
    if circuitOpen {
        return errors.New("circuit breaker open")
    }
    
    // Make request with error handling
    resp, err := client.Do(req)
    if err != nil {
        incrementFailureCount()
        return err
    }
    
    resetFailureCount()
    return nil
}
```

#### Retry Pattern

With exponential backoff for transient failures:
```go
func retryWithBackoff(operation func() error, maxRetries int) error {
    for i := 0; i < maxRetries; i++ {
        if err := operation(); err == nil {
            return nil
        }
        
        backoff := time.Duration(math.Pow(2, float64(i))) * time.Second
        time.Sleep(backoff)
    }
    return errors.New("max retries exceeded")
}
```

#### Bulkhead Pattern

Resource isolation between different API types:
- Separate thread pools for Claude AI vs. SentraIP
- Resource limits per service type
- Circuit breakers per external dependency

### Error Handling Strategy

#### Graceful Degradation

1. **SentraIP Unavailable**: Return cached threat scores or default low-risk
2. **Claude AI Unavailable**: Return static responses or queue requests
3. **Redis Unavailable**: Fall back to in-memory cache with limitations
4. **OTEL Collector Unavailable**: Continue processing with local logging

#### Error Response Format

Standardized error responses across all components:
```json
{
  "error": "oauth_token_failure",
  "message": "Unable to obtain SentraIP OAuth token",
  "timestamp": "2024-01-15T10:30:00Z",
  "trace_id": "abc123...",
  "details": {
    "component": "sentraip_oauth_plugin",
    "retry_after": 60
  }
}
```

## Integration Patterns

### Event-Driven Architecture

While primarily synchronous, the system uses event-driven patterns for:

#### Observability Events

```go
// Trace events
span.AddEvent("oauth.token_refreshed", trace.WithAttributes(
    attribute.String("client_id", clientID),
    attribute.String("expires_at", expiresAt.Format(time.RFC3339)),
))

// Metric events  
tokenRefreshCounter.Inc(ctx, attribute.String("client_id", clientID))
```

#### Background Processing

- **Token Refresh**: Proactive token refresh before expiration
- **Health Checks**: Regular health check events
- **Cache Cleanup**: Periodic cleanup of expired entries

### API Design Patterns

#### RESTful Endpoints

```
GET  /mcp/tools                    # List available tools
POST /mcp/call/{tool_name}         # Execute specific tool
GET  /health                       # Health check
POST /chat                         # Claude chat interface
```

#### Response Envelope Pattern

Consistent response structure:
```json
{
  "success": true,
  "data": { ... },
  "metadata": {
    "timestamp": "2024-01-15T10:30:00Z",
    "version": "v1.0.0",
    "trace_id": "abc123..."
  }
}
```

This technical overview provides the foundation for understanding, maintaining, and extending the Tyk MCP SentraIP system. The architecture prioritizes scalability, observability, and maintainability while providing a robust platform for AI-powered threat intelligence integration.
