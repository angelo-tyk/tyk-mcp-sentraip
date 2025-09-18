# Installation Guide - Tyk MCP SentraIP

This guide provides detailed step-by-step instructions for installing and configuring the Tyk MCP SentraIP system.

## Table of Contents

- [Prerequisites](#prerequisites)
- [Pre-Installation Setup](#pre-installation-setup)
- [Installation Methods](#installation-methods)
- [Configuration](#configuration)
- [Deployment](#deployment)
- [Verification](#verification)
- [Post-Installation](#post-installation)
- [Troubleshooting](#troubleshooting)
- [Upgrade Instructions](#upgrade-instructions)

## Prerequisites

### System Requirements

- **Kubernetes Cluster**: Version 1.21+
  - GKE, EKS, AKS, or local (minikube/kind)
  - Minimum 4 CPU cores, 8GB RAM
  - LoadBalancer support (for cloud deployments)

- **Local Development Tools**:
  - Docker 20.10+
  - kubectl 1.21+
  - Go 1.21+ (for building plugins)
  - Git

### API Access Requirements

- **Claude AI API Key**
  - Anthropic account with API access
  - Sufficient credits for your usage
  - API key with appropriate permissions

- **SentraIP Credentials**
  - SentraIP account with OAuth2 client credentials
  - Client ID and Client Secret
  - Access to threat intelligence APIs

### Cloud Platform Setup (if applicable)

- **Google Cloud Platform**:
  - Project with Container Registry enabled
  - GKE cluster or compute instances
  - Service account with appropriate permissions

- **AWS**:
  - EKS cluster or EC2 instances
  - ECR repository access
  - IAM roles configured

## Pre-Installation Setup

### 1. Clone Repository

```bash
git clone https://github.com/angelo-tyk/tyk-mcp-sentraip.git
cd tyk-mcp-sentraip
```

### 2. Environment Configuration

Create your environment file from the template:

```bash
cp .env.template .env
```

Edit `.env` with your specific values:

```bash
# Google Cloud Project (if using GCP you can find it via  "gcloud config get-value project")
PROJECT_ID=your-gcp-project-id

# Claude API Configuration
CLAUDE_API_KEY=your-claude-api-key

# SentraIP Configuration  
SENTRAIP_CLIENT_ID=your-sentraip-client-id
SENTRAIP_CLIENT_SECRET=your-sentraip-client-secret

# Optional: Custom endpoints
SENTRAIP_API_URL=https://api.sentraip.com
CLAUDE_API_URL=https://api.anthropic.com/v1/messages
```

Source the environment:

```bash
source .env
```

### 3. Verify Prerequisites

```bash
# Check kubectl connection (if in GCP make sure you are authenticated already - gcloud auth login)
kubectl cluster-info

# Check Docker
docker --version

# Check Go installation
go version

# Verify environment variables
echo "Project: $PROJECT_ID"
echo "Claude API Key: ${CLAUDE_API_KEY:0:10}..."
echo "SentraIP Client ID: $SENTRAIP_CLIENT_ID"
```

## Installation Methods

### Method 1: Automated Installation (Recommended)

Use the provided build script for a complete automated deployment:

```bash
# Make script executable
chmod +x scripts/build.sh

# Run automated installation
./scripts/build.sh
```

This script will:
- Build Go plugins
- Build and push Docker images
- Create Kubernetes resources
- Deploy all services
- Verify deployment

### Method 2: Manual Step-by-Step Installation

If you prefer to install components manually or need to customize the process:

#### Step 1: Build Go Plugins

```bash
cd src/tyk-plugin

# Install dependencies
go mod download

# Build plugins
chmod +x build.sh
./build.sh

# Verify plugins were created
ls -la *.so

cd ../..
```

#### Step 2: Build Docker Images

```bash
# Build Tyk Gateway image
docker build -f docker/Dockerfile.gateway -t tyk-gateway-mcp:latest .

# Build Claude MCP Client image  
docker build -f docker/Dockerfile.mcp-client -t claude-mcp-client:latest .
```

#### Step 3: Push to Registry (Cloud Deployment)

For GCP:
```bash
# Tag images
docker tag tyk-gateway-mcp:latest gcr.io/$PROJECT_ID/tyk-gateway-mcp:latest
docker tag claude-mcp-client:latest gcr.io/$PROJECT_ID/claude-mcp-client:latest

# Push images
docker push gcr.io/$PROJECT_ID/tyk-gateway-mcp:latest
docker push gcr.io/$PROJECT_ID/claude-mcp-client:latest
```

For AWS ECR:
```bash
# Get ECR login
aws ecr get-login-password --region your-region | docker login --username AWS --password-stdin your-account.dkr.ecr.your-region.amazonaws.com

# Tag and push
docker tag tyk-gateway-mcp:latest your-account.dkr.ecr.your-region.amazonaws.com/tyk-gateway-mcp:latest
docker push your-account.dkr.ecr.your-region.amazonaws.com/tyk-gateway-mcp:latest
```

#### Step 4: Deploy to Kubernetes

```bash
# Create namespace
kubectl apply -f k8s/namespace.yaml

# Create secrets
kubectl create secret generic claude-api-secret \
  --from-literal=api-key="$CLAUDE_API_KEY" \
  --namespace=tyk \
  --dry-run=client -o yaml | kubectl apply -f -

kubectl create secret generic sentraip-oauth-secret \
  --from-literal=client-id="$SENTRAIP_CLIENT_ID" \
  --from-literal=client-secret="$SENTRAIP_CLIENT_SECRET" \
  --namespace=tyk \
  --dry-run=client -o yaml | kubectl apply -f -

# Update image references in manifests
find k8s/ -name "*.yaml" -type f -exec sed -i "s/\${PROJECT_ID}/$PROJECT_ID/g" {} \;

# Deploy ConfigMaps and Services
kubectl apply -f k8s/configmaps/
kubectl apply -f k8s/services/

# Deploy applications in dependency order
kubectl apply -f k8s/deployments/tyk-redis.yaml
kubectl wait --for=condition=available --timeout=120s deployment/tyk-redis -n tyk

kubectl apply -f k8s/deployments/otel-collector.yaml
kubectl wait --for=condition=available --timeout=120s deployment/otel-collector -n tyk

kubectl apply -f k8s/deployments/claude-mcp-client.yaml
kubectl wait --for=condition=available --timeout=180s deployment/claude-mcp-client -n tyk

kubectl apply -f k8s/deployments/tyk-gateway.yaml
kubectl wait --for=condition=available --timeout=300s deployment/tyk-gateway -n tyk
```

## Configuration

### Tyk Gateway Configuration

The main Tyk configuration is in `k8s/configmaps/tyk-gateway-config.yaml`. Key settings:

```yaml
# API Gateway settings
listen_port: 8080
secret: "change-me-in-production"

# Redis storage
storage:
  type: "redis"
  host: "tyk-redis"
  port: 6379

# OpenTelemetry integration
opentelemetry:
  enabled: true
  endpoint: "http://otel-collector:4317"
```

### Claude MCP Client Configuration

Environment variables for the Claude MCP client:

```bash
CLAUDE_API_KEY=your-claude-api-key
TYK_GATEWAY_URL=http://tyk-gateway:8080
OTEL_EXPORTER_OTLP_ENDPOINT=http://otel-collector:4317
```

### OpenTelemetry Configuration

OTEL collector configuration in `k8s/configmaps/otel-collector-config.yaml`:

```yaml
receivers:
  otlp:
    protocols:
      grpc:
        endpoint: 0.0.0.0:4317
      http:
        endpoint: 0.0.0.0:4318

exporters:
  jaeger:
    endpoint: jaeger-collector:14250
  prometheus:
    endpoint: "0.0.0.0:8889"
```

## Deployment

### Production Deployment Considerations

#### Resource Allocation

Recommended resource limits for production:

```yaml
# Tyk Gateway
resources:
  requests:
    memory: "1Gi"
    cpu: "1000m"
  limits:
    memory: "4Gi" 
    cpu: "4000m"

# Claude MCP Client
resources:
  requests:
    memory: "512Mi"
    cpu: "500m"
  limits:
    memory: "2Gi"
    cpu: "2000m"
```

#### High Availability Setup

For production deployment, configure multiple replicas:

```yaml
spec:
  replicas: 3  # Tyk Gateway
  replicas: 2  # Claude MCP Client
```

#### Security Hardening

1. **Network Policies**:
```bash
kubectl apply -f k8s/network-policies/
```

2. **Pod Security Standards**:
```yaml
securityContext:
  runAsNonRoot: true
  runAsUser: 1000
  fsGroup: 1000
```

3. **Secret Management**:
```bash
# Use external secret management (e.g., Google Secret Manager)
kubectl create secret generic claude-api-secret \
  --from-literal=api-key="$(gcloud secrets versions access latest --secret=claude-api-key)"
```

### Environment-Specific Configurations

#### Development Environment

```bash
export ENVIRONMENT=development
export REPLICAS=1
export RESOURCE_LIMITS_ENABLED=false
```

#### Staging Environment

```bash
export ENVIRONMENT=staging
export REPLICAS=2
export RESOURCE_LIMITS_ENABLED=true
```

#### Production Environment

```bash
export ENVIRONMENT=production
export REPLICAS=3
export RESOURCE_LIMITS_ENABLED=true
export MONITORING_ENABLED=true
```

## Verification

### 1. Check Pod Status

```bash
kubectl get pods -n tyk
```

Expected output:
```
NAME                                READY   STATUS    RESTARTS   AGE
claude-mcp-client-xxx              1/1     Running   0          5m
otel-collector-xxx                 1/1     Running   0          5m
tyk-gateway-xxx                    1/1     Running   0          5m
tyk-redis-xxx                      1/1     Running   0          5m
```

### 2. Test Gateway Health

```bash
# Get gateway IP
TYK_IP=$(kubectl get svc tyk-gateway -n tyk -o jsonpath='{.status.loadBalancer.ingress[0].ip}')

# Test health endpoint
curl http://$TYK_IP:8080/hello
```

Expected response:
```json
{"status": "pass", "version": "v5.0", "description": "Tyk GW"}
```

### 3. Test MCP Tools

```bash
# List available MCP tools
curl http://$TYK_IP:8080/mcp/tools

# Test threat intelligence tool
curl -X POST http://$TYK_IP:8080/mcp/call/sentraip_threat_check \
  -H "Content-Type: application/json" \
  -d '{"target": "8.8.8.8", "type": "ip"}'
```

### 4. Test Claude Integration

```bash
# Get Claude MCP client IP
CLAUDE_IP=$(kubectl get svc claude-mcp-client -n tyk -o jsonpath='{.status.loadBalancer.ingress[0].ip}')

# Test chat endpoint
curl -X POST http://$CLAUDE_IP:8080/chat \
  -H "Content-Type: application/json" \
  -d '{"message": "Hello, can you check the threat level for IP 1.1.1.1?"}'
```

### 5. Verify Observability

```bash
# Check OpenTelemetry collector
kubectl logs deployment/otel-collector -n tyk

# Check traces are being generated
curl http://jaeger-ui:16686
```

## Post-Installation

### 1. Configure Monitoring

Set up monitoring dashboards:

```bash
# Install Grafana (optional)
helm repo add grafana https://grafana.github.io/helm-charts
helm install grafana grafana/grafana -n monitoring --create-namespace

# Import pre-built dashboards
kubectl apply -f monitoring/dashboards/
```

### 2. Set Up Alerts

Configure alerting rules:

```yaml
# Create PrometheusRule for alerts
apiVersion: monitoring.coreos.com/v1
kind: PrometheusRule
metadata:
  name: tyk-mcp-alerts
spec:
  groups:
  - name: tyk.rules
    rules:
    - alert: TykGatewayDown
      expr: up{job="tyk-gateway"} == 0
      for: 5m
```

### 3. Backup Configuration

```bash
# Backup ConfigMaps and Secrets
kubectl get configmaps -n tyk -o yaml > backup/configmaps.yaml
kubectl get secrets -n tyk -o yaml > backup/secrets.yaml
```

### 4. Performance Tuning

Monitor and adjust based on your workload:

```bash
# Monitor resource usage
kubectl top pods -n tyk

# Adjust HPA if needed
kubectl autoscale deployment tyk-gateway --min=3 --max=10 --cpu-percent=70 -n tyk
```

## Troubleshooting

### Common Issues

#### 1. Pods Not Starting

**Symptom**: Pods stuck in `Pending` or `CrashLoopBackOff`

**Solution**:
```bash
# Check pod events
kubectl describe pod <pod-name> -n tyk

# Check logs
kubectl logs <pod-name> -n tyk

# Common fixes:
# - Verify image pull secrets
# - Check resource quotas
# - Verify node capacity
```

#### 2. OAuth Token Failures

**Symptom**: `401 Unauthorized` errors from SentraIP

**Solution**:
```bash
# Verify SentraIP credentials
kubectl get secret sentraip-oauth-secret -n tyk -o jsonpath='{.data.client-id}' | base64 -d
kubectl get secret sentraip-oauth-secret -n tyk -o jsonpath='{.data.client-secret}' | base64 -d

# Check plugin logs
kubectl logs deployment/tyk-gateway -n tyk | grep -i oauth

# Test OAuth flow manually
curl -X POST https://auth.sentraip.com/oauth/token \
  -H "Content-Type: application/x-www-form-urlencoded" \
  -H "Authorization: Basic $(echo -n "$SENTRAIP_CLIENT_ID:$SENTRAIP_CLIENT_SECRET" | base64)" \
  -d "grant_type=client_credentials&scope=threat-intelligence"
```

#### 3. Claude API Connection Issues

**Symptom**: Claude MCP client returning errors

**Solution**:
```bash
# Verify Claude API key
kubectl get secret claude-api-secret -n tyk -o jsonpath='{.data.api-key}' | base64 -d

# Test Claude API directly
curl -X POST https://api.anthropic.com/v1/messages \
  -H "x-api-key: $CLAUDE_API_KEY" \
  -H "anthropic-version: 2023-06-01" \
  -H "content-type: application/json" \
  -d '{"model": "claude-3-haiku-20240307", "max_tokens": 10, "messages": [{"role": "user", "content": "Hi"}]}'
```

#### 4. Go Plugin Loading Errors

**Symptom**: Tyk gateway fails to load custom plugins

**Solution**:
```bash
# Check plugin compilation
cd src/tyk-plugin
go build -buildmode=plugin -o test.so sentraip_oauth.go

# Verify plugin symbols
nm -D sentraip_oauth.so | grep -E "(SentraIPOAuthMiddleware|init)"

# Check plugin path in container
kubectl exec deployment/tyk-gateway -n tyk -- ls -la /opt/tyk-gateway/plugins/
```

#### 5. Network Connectivity Issues

**Symptom**: Services can't reach each other

**Solution**:
```bash
# Test DNS resolution
kubectl exec deployment/tyk-gateway -n tyk -- nslookup tyk-redis
kubectl exec deployment/tyk-gateway -n tyk -- nslookup claude-mcp-client

# Check NetworkPolicies
kubectl get networkpolicies -n tyk

# Test connectivity
kubectl exec deployment/tyk-gateway -n tyk -- curl http://claude-mcp-client:8080/health
```

### Debug Mode

Enable debug logging for troubleshooting:

```bash
# Enable debug mode
kubectl set env deployment/tyk-gateway LOG_LEVEL=debug -n tyk
kubectl set env deployment/claude-mcp-client LOG_LEVEL=debug -n tyk

# Check debug logs
kubectl logs deployment/tyk-gateway -n tyk --follow
kubectl logs deployment/claude-mcp-client -n tyk --follow
```

### Log Analysis

Common log patterns to look for:

```bash
# OAuth token issues
kubectl logs deployment/tyk-gateway -n tyk | grep -i "oauth"

# Plugin loading issues  
kubectl logs deployment/tyk-gateway -n tyk | grep -i "plugin"

# MCP tool execution
kubectl logs deployment/claude-mcp-client -n tyk | grep -i "mcp"

# OpenTelemetry traces
kubectl logs deployment/otel-collector -n tyk | grep -i "trace"
```

## Upgrade Instructions

### 1. Backup Current Deployment

```bash
# Create backup directory
mkdir -p backup/$(date +%Y-%m-%d)

# Backup configurations
kubectl get all -n tyk -o yaml > backup/$(date +%Y-%m-%d)/all-resources.yaml
kubectl get configmaps -n tyk -o yaml > backup/$(date +%Y-%m-%d)/configmaps.yaml
kubectl get secrets -n tyk -o yaml > backup/$(date +%Y-%m-%d)/secrets.yaml
```

### 2. Update Source Code

```bash
git fetch origin
git checkout main
git pull origin main
```

### 3. Build New Images

```bash
# Rebuild with new version tag
export VERSION=v1.1.0

docker build -f docker/Dockerfile.gateway -t gcr.io/$PROJECT_ID/tyk-gateway-mcp:$VERSION .
docker build -f docker/Dockerfile.mcp-client -t gcr.io/$PROJECT_ID/claude-mcp-client:$VERSION .

docker push gcr.io/$PROJECT_ID/tyk-gateway-mcp:$VERSION
docker push gcr.io/$PROJECT_ID/claude-mcp-client:$VERSION
```

### 4. Rolling Update

```bash
# Update deployments with new image tags
kubectl set image deployment/tyk-gateway tyk-gateway=gcr.io/$PROJECT_ID/tyk-gateway-mcp:$VERSION -n tyk
kubectl set image deployment/claude-mcp-client claude-mcp-client=gcr.io/$PROJECT_ID/claude-mcp-client:$VERSION -n tyk

# Monitor rollout
kubectl rollout status deployment/tyk-gateway -n tyk
kubectl rollout status deployment/claude-mcp-client -n tyk
```

### 5. Verify Upgrade

```bash
# Check pod versions
kubectl get pods -n tyk -o jsonpath='{range .items[*]}{.metadata.name}{"\t"}{.spec.containers[0].image}{"\n"}{end}'

# Test functionality
curl http://$TYK_IP:8080/hello
curl http://$TYK_IP:8080/mcp/tools
```

### 6. Rollback (if needed)

```bash
# Rollback to previous version
kubectl rollout undo deployment/tyk-gateway -n tyk
kubectl rollout undo deployment/claude-mcp-client -n tyk

# Check rollback status
kubectl rollout status deployment/tyk-gateway -n tyk
```

## Cleanup

To completely remove the deployment:

```bash
# Use cleanup script
chmod +x scripts/cleanup.sh
./scripts/cleanup.sh

# Or manual cleanup
kubectl delete namespace tyk
```

---

For additional support, see the main [README.md](README.md) or create an issue in the repository.
