# tyk-mcp-sentraip
Experimental project to implement API security with Tyk Gateway + SentraIP threat intel + Claude MCP integration. Features conversational threat analysis, real-time blocking, OpenTelemetry observability, and automated incident response.

This example was created on Google Cloud GKE but can be easily adapted to any Cloud platform.


# Tyk MCP SentraIP

An AI-ready API gateway that integrates Tyk Gateway with Claude AI through the Model Context Protocol (MCP), featuring automated threat intelligence from SentraIP and comprehensive OpenTelemetry observability.

## Overview

This project creates a sophisticated API gateway that bridges AI capabilities with threat intelligence and API management. It enables Claude AI to access real-time threat data, API analytics, and contextual information through standardized MCP tools while maintaining enterprise-grade security and observability.

## Key Features

### AI Integration
- **Model Context Protocol (MCP)** support for Claude AI
- Custom MCP tools for threat intelligence, API analytics, and context search
- Seamless tool calling and response handling

### Threat Intelligence
- **SentraIP Integration** with automated OAuth2 client credentials flow
- Real-time IP and domain reputation checking
- Threat score analysis and risk assessment
- Cached token management for optimal performance

### API Management
- **Tyk Gateway** with custom Go plugins
- Rate limiting, authentication, and request transformation
- API versioning and routing
- Load balancing and health checking

### Observability
- **OpenTelemetry** integration with detailed tracing
- Request/response correlation and context propagation
- Performance metrics and error tracking
- Jaeger tracing and Prometheus metrics

### Cloud-Native
- **Kubernetes** deployment with full orchestration
- Horizontal pod autoscaling
- ConfigMaps and Secrets management
- Health checks and rolling updates

## Quick Start

### Prerequisites
- Kubernetes cluster (GKE, EKS, or local)
- Docker and kubectl installed
- Claude API key from Anthropic
- SentraIP OAuth credentials

### Installation

1. **Clone the repository**
```bash
git clone https://github.com/yourusername/tyk-mcp-sentraip.git
cd tyk-mcp-sentraip
```

2. **Set environment variables**
```bash
export PROJECT_ID=your-gcp-project-id
export CLAUDE_API_KEY=your-claude-api-key
export SENTRAIP_CLIENT_ID=your-sentraip-client-id
export SENTRAIP_CLIENT_SECRET=your-sentraip-client-secret
```

3. **Deploy to Kubernetes**
```bash
chmod +x scripts/build.sh
./scripts/build.sh
```

4. **Verify deployment**
```bash
kubectl get pods -n tyk
```

See [INSTALL.md](INSTALL.md) for detailed installation instructions.

## Usage Examples

### Check Threat Intelligence
```bash
# Get threat intelligence for an IP address
curl -X POST http://your-gateway-ip:8080/mcp/call/sentraip_threat_check \
  -H "Content-Type: application/json" \
  -d '{"target": "192.168.1.1", "type": "ip"}'
```

### Get API Analytics
```bash
# Retrieve API usage analytics
curl -X POST http://your-gateway-ip:8080/mcp/call/tyk_api_analytics \
  -H "Content-Type: application/json" \
  -d '{"api_id": "claude-ai-mcp", "time_range": "24h"}'
```

### Chat with Claude via MCP
```bash
# Send a message to Claude with MCP tool access
curl -X POST http://your-claude-client-ip:8080/chat \
  -H "Content-Type: application/json" \
  -d '{"message": "Check the threat level for IP 203.0.113.1"}'
```

### List Available MCP Tools
```bash
# Get list of available MCP tools
curl http://your-gateway-ip:8080/mcp/tools
```

## Architecture

The system consists of several interconnected components:

- **Tyk Gateway** - API gateway with custom Go plugins
- **Claude MCP Client** - Service bridging Claude AI and MCP tools
- **SentraIP API** - Threat intelligence service with OAuth2
- **OpenTelemetry Collector** - Observability and tracing
- **Redis** - Session storage and caching
- **Kubernetes** - Container orchestration

See [OVERVIEW.md](OVERVIEW.md) for detailed architecture information.

## Project Structure

```
tyk-mcp-sentraip/
├── README.md                   # This file
├── INSTALL.md                  # Detailed installation guide
├── OVERVIEW.md                 # Technical architecture overview
├── .env.template               # Environment variables template
├── src/                        # Source code
│   ├── tyk-plugin/            # Tyk Gateway Go plugins
│   └── claude-mcp-client/     # Claude MCP client service
├── k8s/                       # Kubernetes manifests
│   ├── deployments/           # Application deployments
│   ├── services/              # Service definitions
│   ├── configmaps/            # Configuration files
│   └── secrets/               # Secret templates
├── docker/                    # Docker build files
└── scripts/                   # Build and deployment scripts
```

## Custom MCP Tools

### sentraip_threat_check
Check IP addresses or domains against SentraIP threat intelligence database.

**Input:**
- `target` (string): IP address or domain to check
- `type` (string): "ip" or "domain"

**Output:**
- Threat score (0-10)
- Risk assessment
- Threat categories
- Last updated timestamp

### tyk_api_analytics
Retrieve API usage analytics and performance metrics from Tyk Gateway.

**Input:**
- `api_id` (string): API identifier to analyze
- `time_range` (string): Time range (24h, 7d, 30d)

**Output:**
- Request counts and error rates
- Response times and performance metrics
- Top endpoints and status codes
- Traffic patterns

### claude_context_search
Search previous conversations and contextual information.

**Input:**
- `query` (string): Search query
- `limit` (string): Number of results to return

**Output:**
- Matching conversation snippets
- Relevance scores
- Timestamps and conversation IDs

## Monitoring and Observability

The system provides comprehensive monitoring through:

- **Distributed Tracing** with OpenTelemetry and Jaeger
- **Metrics Collection** with Prometheus
- **Log Aggregation** with structured JSON logging
- **Health Checks** for all services
- **Performance Monitoring** with request/response timing

Access monitoring dashboards at:
- Jaeger UI: `http://jaeger-ui:16686`
- Prometheus: `http://prometheus:9090`

## Configuration

### Environment Variables

Key configuration options:

- `PROJECT_ID` - Google Cloud Project ID
- `CLAUDE_API_KEY` - Anthropic Claude API key
- `SENTRAIP_CLIENT_ID` - SentraIP OAuth client ID
- `SENTRAIP_CLIENT_SECRET` - SentraIP OAuth client secret
- `TYK_GATEWAY_URL` - Internal Tyk Gateway URL
- `OTEL_EXPORTER_OTLP_ENDPOINT` - OpenTelemetry collector endpoint

### Kubernetes Configuration

Configuration is managed through Kubernetes ConfigMaps and Secrets:

- `tyk-gateway-config` - Tyk Gateway configuration
- `otel-collector-config` - OpenTelemetry collector settings
- `claude-api-secret` - Claude API credentials
- `sentraip-oauth-secret` - SentraIP OAuth credentials

## Security Considerations

- **OAuth2 Token Management** - Automated token refresh and secure storage
- **API Key Protection** - Kubernetes Secrets for sensitive data
- **Network Policies** - Restricted inter-service communication
- **TLS Encryption** - HTTPS for all external communications
- **Rate Limiting** - Protection against abuse and DoS attacks

## Troubleshooting

### Common Issues

**Gateway not starting:**
```bash
kubectl logs deployment/tyk-gateway -n tyk
```

**OAuth token failures:**
```bash
# Check SentraIP credentials
kubectl get secret sentraip-oauth-secret -n tyk -o yaml
```

**MCP tools not responding:**
```bash
# Verify Claude MCP client health
curl http://your-claude-client-ip:8080/health
```

### Debug Mode

Enable debug logging by setting environment variables:
```bash
export LOG_LEVEL=debug
export TYK_LOGLEVEL=debug
```

## Contributing

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

### Development Setup

```bash
# Install Go dependencies
cd src/tyk-plugin && go mod download
cd ../claude-mcp-client && go mod download

# Build plugins locally
cd src/tyk-plugin && ./build.sh

# Run tests
go test ./...
```

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## Support

For support and questions:

- Create an issue in this repository
- Check the [troubleshooting guide](INSTALL.md#troubleshooting)
- Review the [architecture documentation](OVERVIEW.md)

## Roadmap

- [ ] Support for additional AI models (GPT-4, Gemini)
- [ ] Enhanced threat intelligence sources
- [ ] GraphQL API support
- [ ] Advanced analytics and reporting
- [ ] Multi-tenant support
- [ ] Webhook integrations

---

**Built with** Tyk Gateway, Claude AI, SentraIP, OpenTelemetry, and Kubernetes.
