#!/bin/bash
set -e

echo "Deploying Tyk MCP SentraIP Integration..."

# Check required environment variables
required_vars=("PROJECT_ID" "CLAUDE_API_KEY" "SENTRAIP_CLIENT_ID" "SENTRAIP_CLIENT_SECRET")
for var in "${required_vars[@]}"; do
    if [[ -z "${!var}" ]]; then
        echo "Error: $var environment variable is not set"
        exit 1
    fi
done

# Create namespace
kubectl apply -f k8s/namespace.yaml

# Apply secrets
envsubst < k8s/secrets/claude-api-secret.yaml | kubectl apply -f -
envsubst < k8s/secrets/sentraip-oauth-secret.yaml | kubectl apply -f -

# Apply configmaps
kubectl apply -f k8s/configmaps/

# Apply services
kubectl apply -f k8s/services/

# Apply deployments
kubectl apply -f k8s/deployments/

# Wait for deployments
echo "Waiting for deployments to be ready..."
kubectl wait --for=condition=available --timeout=300s deployment/tyk-gateway -n tyk
kubectl wait --for=condition=available --timeout=300s deployment/otel-collector -n tyk
kubectl wait --for=condition=available --timeout=300s deployment/claude-mcp-client -n tyk

echo "Deployment complete!"
echo "Tyk Gateway: $(kubectl get svc tyk-gateway -n tyk -o jsonpath='{.status.loadBalancer.ingress[0].ip}'):8080"
echo "Claude MCP Client: $(kubectl get svc claude-mcp-client -n tyk -o jsonpath='{.status.loadBalancer.ingress[0].ip}'):8080"
scripts/test-integration.sh
#!/bin/bash
set -e

TYK_IP=$(kubectl get svc tyk-gateway -n tyk -o jsonpath='{.status.loadBalancer.ingress[0].ip}')
CLAUDE_IP=$(kubectl get svc claude-mcp-client -n tyk -o jsonpath='{.status.loadBalancer.ingress[0].ip}')

if [[ -z "$TYK_IP" ]] || [[ -z "$CLAUDE_IP" ]]; then
    echo "Error: Could not get service IPs. Check if services are deployed."
    exit 1
fi

echo "Testing Tyk Gateway MCP endpoints..."

# Test MCP tools endpoint
echo "Testing /mcp/tools endpoint..."
curl -s -X GET "http://$TYK_IP:8080/mcp/tools" | jq .

# Test direct IP reputation check
echo -e "\nTesting direct SentraIP integration..."
curl -s -X POST "http://$TYK_IP:8080/mcp/execute" \
  -H "Content-Type: application/json" \
  -d '{
    "tool": "check_ip_reputation",
    "parameters": {"ip": "8.8.8.8"},
    "session_id": "test-session"
  }' | jq .

echo -e "\nTesting Claude MCP Client..."

# Test conversational interface
curl -s -X POST "http://$CLAUDE_IP:8080/chat" \
  -H "Content-Type: application/json" \
  -d '{
    "message": "Check the reputation of IP address 1.1.1.1"
  }' | jq .

echo -e "\nTesting complex query..."
curl -s -X POST "http://$CLAUDE_IP:8080/chat" \
  -H "Content-Type: application/json" \
  -d '{
    "message": "Show me threat analysis for these IPs: 185.220.101.45, 192.168.1.100"
  }' | jq .

echo -e "\nIntegration tests complete!"
