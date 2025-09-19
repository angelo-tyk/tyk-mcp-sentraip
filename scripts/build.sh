#!/bin/bash
set -e

echo "🚀 Starting Tyk MCP SentraIP deployment..."

# Validate required environment variables
if [ -z "$PROJECT_ID" ]; then
    echo "❌ PROJECT_ID environment variable is required"
    exit 1
fi

if [ -z "$CLAUDE_API_KEY" ]; then
    echo "❌ CLAUDE_API_KEY environment variable is required"
    exit 1
fi

if [ -z "$SENTRAIP_CLIENT_ID" ]; then
    echo "❌ SENTRAIP_CLIENT_ID environment variable is required"
    exit 1
fi

if [ -z "$SENTRAIP_CLIENT_SECRET" ]; then
    echo "❌ SENTRAIP_CLIENT_SECRET environment variable is required"
    exit 1
fi

echo "✅ Environment variables validated"

# Create GKE cluster
echo "🏗️ Creating GKE cluster..."
CLUSTER_NAME="tyk-mcp-cluster"
ZONE="us-central1-a"

# Check if cluster already exists
if gcloud container clusters describe $CLUSTER_NAME --zone=$ZONE &>/dev/null; then
    echo "📋 Cluster $CLUSTER_NAME already exists, using existing cluster"
else
    echo "🆕 Creating new GKE cluster: $CLUSTER_NAME"
    gcloud container clusters create $CLUSTER_NAME \
        --zone=$ZONE \
        --num-nodes=3 \
        --machine-type=e2-standard-2 \
        --disk-size=50GB \
        --enable-autorepair \
        --enable-autoupgrade \
        --enable-ip-alias \
        --project=$PROJECT_ID
fi

# Get cluster credentials
echo "🔐 Configuring kubectl for cluster access..."
gcloud container clusters get-credentials $CLUSTER_NAME --zone=$ZONE --project=$PROJECT_ID

# Verify cluster connection
echo "📡 Verifying cluster connection..."
kubectl cluster-info

echo "✅ Cluster ready and configured"

# Build Go plugins
echo "🔨 Building Tyk Go plugins..."
cd src/tyk-plugin
chmod +x build.sh
./build.sh
cd ../..
echo "✅ Go plugins built successfully"

# Build and push Docker images
echo "🐳 Building and pushing Docker images..."

# Configure Docker for GCR
gcloud auth configure-docker --quiet

# Build Tyk Gateway image
docker build -f docker/Dockerfile.gateway -t tyk-gateway-mcp:latest .
docker tag tyk-gateway-mcp:latest gcr.io/$PROJECT_ID/tyk-gateway-mcp:latest
docker push gcr.io/$PROJECT_ID/tyk-gateway-mcp:latest
echo "✅ Tyk Gateway image pushed"

# Build Claude MCP Client image
docker build -f docker/Dockerfile.mcp-client -t claude-mcp-client:latest .
docker tag claude-mcp-client:latest gcr.io/$PROJECT_ID/claude-mcp-client:latest
docker push gcr.io/$PROJECT_ID/claude-mcp-client:latest
echo "✅ Claude MCP Client image pushed"

# Create Kubernetes namespace
echo "☸️ Creating Kubernetes resources..."
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

echo "✅ Secrets created"

# Update manifests with PROJECT_ID
echo "📝 Updating Kubernetes manifests..."
find k8s/ -name "*.yaml" -type f -exec sed -i "s/\${PROJECT_ID}/$PROJECT_ID/g" {} \;

# Deploy ConfigMaps
kubectl apply -f k8s/configmaps/

# Deploy Services
kubectl apply -f k8s/services/

# Deploy applications in order
echo "🚀 Deploying applications..."

# Deploy Redis first
kubectl apply -f k8s/deployments/tyk-redis.yaml
kubectl wait --for=condition=available --timeout=120s deployment/tyk-redis -n tyk

# Deploy OTEL Collector
kubectl apply -f k8s/deployments/otel-collector.yaml
kubectl wait --for=condition=available --timeout=120s deployment/otel-collector -n tyk

# Deploy Claude MCP Client
kubectl apply -f k8s/deployments/claude-mcp-client.yaml
kubectl wait --for=condition=available --timeout=180s deployment/claude-mcp-client -n tyk

# Deploy Tyk Gateway
kubectl apply -f k8s/deployments/tyk-gateway.yaml
kubectl wait --for=condition=available --timeout=300s deployment/tyk-gateway -n tyk

echo "✅ All applications deployed successfully"

# Get service endpoints
echo "🌐 Getting service endpoints..."
sleep 30

TYK_IP=$(kubectl get svc tyk-gateway -n tyk -o jsonpath='{.status.loadBalancer.ingress[0].ip}')
CLAUDE_IP=$(kubectl get svc claude-mcp-client -n tyk -o jsonpath='{.status.loadBalancer.ingress[0].ip}')

echo ""
echo "🎉 DEPLOYMENT COMPLETE!"
echo "========================="
echo "Cluster: $CLUSTER_NAME (zone: $ZONE)"
echo "Tyk Gateway: http://$TYK_IP:8080"
echo "Claude MCP Client: http://$CLAUDE_IP:8080"
echo ""
echo "🧪 Test Commands:"
echo "curl http://$TYK_IP:8080/hello"
echo "curl http://$TYK_IP:8080/mcp/tools"
echo ""
echo "📊 Monitor with:"
echo "kubectl get pods -n tyk"
echo "kubectl logs deployment/tyk-gateway -n tyk"
echo ""
echo "🗑️ To cleanup:"
echo "gcloud container clusters delete $CLUSTER_NAME --zone=$ZONE"
echo ""
