Important Adaptations Needed
Before deploying, you'll need to customize several files:

Replace placeholders in all YAML files:

${PROJECT_ID} → your actual GCP project ID
your-gcp-project → your project name
yourusername → your GitHub username


Update Go modules in go.mod files:

go   module github.com/yourusername/tyk-mcp-sentraip/claude-mcp-client

Configure secrets properly:

bash   cp .env.template .env
   # Edit .env with your actual API keys

Test locally first:

bash   # Build the plugin locally
   cd src/tyk-plugin && ./build.sh
   
   # Test the Go client
   cd ../claude-mcp-client && go run main.go
Security Considerations

Never commit actual API keys to Git
Use Kubernetes secrets or external secret management
Rotate credentials regularly
Monitor API usage and costs
Review the OAuth2 configuration for production


# Installation Guide

This guide walks you through setting up the Tyk MCP SentraIP integration on Google Kubernetes Engine (GKE).

## Prerequisites

Before you begin, ensure you have:

- Google Cloud account with billing enabled
- GKE cluster running Kubernetes 1.24+
- `kubectl` configured to access your cluster
- Docker installed locally
- Go 1.21+ for building the plugin
- Claude API key from Anthropic
- SentraIP API credentials

## Environment Setup

1. **Set required environment variables:**
```bash
export PROJECT_ID="your-gcp-project-id"
export CLAUDE_API_KEY="your-claude-api-key"
export SENTRAIP_CLIENT_ID="your-sentraip-client-id"
export SENTRAIP_CLIENT_SECRET="your-sentraip-client-secret"
```

2. **Configure Google Cloud:**
```bash
gcloud auth login
gcloud config set project $PROJECT_ID
gcloud auth configure-docker gcr.io
```

3. **Get GKE credentials:**
```bash
gcloud container clusters get-credentials your-cluster-name --zone your-zone
```

## Quick Installation

### Option 1: Automated Script

```bash
git clone https://github.com/yourusername/tyk-mcp-sentraip.git
cd tyk-mcp-sentraip
chmod +x scripts/build-and-deploy.sh
./scripts/build-and-deploy.sh
```

### Option 2: Manual Step-by-Step

Follow the detailed steps below for more control over the installation process.

## Manual Installation

### Step 1: Build and Push Images

1. **Build the Tyk Gateway with MCP plugin:**
```bash
docker build -f docker/Dockerfile.gateway -t tyk-gateway-mcp:latest .
docker tag tyk-gateway-mcp:latest gcr.io/$PROJECT_ID/tyk-gateway-mcp:latest
docker push gcr.io/$PROJECT_ID/tyk-gateway-mcp:latest
```

2. **Build the Claude MCP client:**
```bash
docker build -f docker/Dockerfile.mcp-client -t claude-mcp-client:latest .
docker tag claude-mcp-client:latest gcr.io/$PROJECT_ID/claude-mcp-client:latest
docker push gcr.io/$PROJECT_ID/claude-mcp-client:latest
```

### Step 2: Create Kubernetes Namespace

```bash
kubectl apply -f k8s/namespace.yaml
```

### Step 3: Configure Secrets

**Create environment-specific secrets:**
```bash
# Claude API secret
kubectl create secret generic claude-api-secret \
  --from-literal=api-key="$CLAUDE_API_KEY" \
  --namespace=tyk

# SentraIP OAuth secret
kubectl create secret generic sentraip-oauth-secret \
  --from-literal=client-id="$SENTRAIP_CLIENT_ID" \
  --from-literal=client-secret="$SENTRAIP_CLIENT_SECRET" \
  --namespace=tyk
```

### Step 4: Apply Configuration Maps

```bash
kubectl apply -f k8s/configmaps/
```

### Step 5: Deploy Services

```bash
kubectl apply -f k8s/services/
```

### Step 6: Deploy Applications

```bash
# Deploy Redis (required for Tyk)
kubectl apply -f k8s/deployments/tyk-redis.yaml

# Deploy OpenTelemetry Collector  
kubectl apply -f k8s/deployments/otel-collector.yaml

# Deploy Tyk Gateway with MCP plugin
kubectl apply -f k8s/deployments/tyk-gateway.yaml

# Deploy Claude MCP Client
kubectl apply -f k8s/deployments/claude-mcp-client.yaml
```

### Step 7: Wait for Deployments

```bash
kubectl wait --for=condition=available --timeout=300s deployment/tyk-gateway -n tyk
kubectl wait --for=condition=available --timeout=300s deployment/otel-collector -n tyk  
kubectl wait --for=condition=available --timeout=300s deployment/claude-mcp-client -n tyk
```

## Verification

### Check Deployment Status

```bash
kubectl get pods -n tyk
kubectl get services -n tyk
```

### Test MCP Endpoints

1. **Get service IPs:**
```bash
export TYK_IP=$(kubectl get svc tyk-gateway -n tyk -o jsonpath='{.status.loadBalancer.ingress[0].ip}')
export CLAUDE_IP=$(kubectl get svc claude-mcp-client -n tyk -o jsonpath='{.status.loadBalancer.ingress[0].ip}')
```

2. **Test MCP tools endpoint:**
```bash
curl -X GET "http://$TYK_IP:8080/mcp/tools"
```

3. **Test threat intelligence query:**
```bash
curl -X POST "http://$CLAUDE_IP:8080/chat" \
  -H "Content-Type: application/json" \
  -d '{"message": "Check the reputation of IP address 8.8.8.8"}'
```

## Troubleshooting

### Common Issues

**1. Image pull errors:**
```bash
# Ensure Docker is authenticated with GCR
gcloud auth configure-docker gcr.io
```

**2. Secret creation failures:**
```bash
# Check if namespace exists
kubectl get namespace tyk
```

**3. Plugin loading errors:**
```bash
# Check Tyk Gateway logs
kubectl logs deployment/tyk-gateway -n tyk
```

**4. MCP connection issues:**
```bash
# Verify Claude API key is valid
curl -H "x-api-key: $CLAUDE_API_KEY" https://api.anthropic.com/v1/messages
```

### View Logs

```bash
# Tyk Gateway logs
kubectl logs -f deployment/tyk-gateway -n tyk

# Claude MCP Client logs  
kubectl logs -f deployment/claude-mcp-client -n tyk

# OpenTelemetry Collector logs
kubectl logs -f deployment/otel-collector -n tyk
```

### Service Discovery

```bash
# Check service endpoints
kubectl get endpoints -n tyk

# Test internal connectivity
kubectl exec -it deployment/claude-mcp-client -n tyk -- curl http://tyk-gateway:8080/health
```

## Configuration

### Environment Variables

You can customize the installation by setting these environment variables before deployment:

- `PROJECT_ID` - GCP project ID
- `CLAUDE_API_KEY` - Anthropic Claude API key
- `SENTRAIP_CLIENT_ID` - SentraIP OAuth client ID  
- `SENTRAIP_CLIENT_SECRET` - SentraIP OAuth client secret
- `TYK_GATEWAY_IMAGE` - Custom Tyk Gateway image (optional)
- `CLAUDE_CLIENT_IMAGE` - Custom Claude client image (optional)

### Custom Configuration

To modify the configuration:

1. **Edit ConfigMaps:**
```bash
kubectl edit configmap tyk-gateway-config -n tyk
```

2. **Update API definitions:**
```bash
kubectl edit configmap sentraip-api-definition -n tyk  
```

3. **Restart deployments:**
```bash
kubectl rollout restart deployment/tyk-gateway -n tyk
```

## Uninstallation

To remove the installation:

```bash
./scripts/cleanup.sh
```

Or manually:

```bash
kubectl delete namespace tyk
docker rmi gcr.io/$PROJECT_ID/tyk-gateway-mcp:latest
docker rmi gcr.io/$PROJECT_ID/claude-mcp-client:latest
```

## Next Steps

After installation, see:

- [examples/curl-examples.sh](examples/curl-examples.sh) for API testing
- [docs/TROUBLESHOOTING.md](docs/TROUBLESHOOTING.md) for detailed debugging
- [examples/test-queries.md](examples/test-queries.md) for conversation examples

## Support

This is experimental code. For questions or issues:

1. Check the troubleshooting section above
2. Review the GitHub Issues page
3. For production implementations, consult with Tyk's professional services team
