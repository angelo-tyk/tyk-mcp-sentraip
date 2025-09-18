#!/bin/bash
set -e

echo "🧹 Starting cleanup of Tyk MCP SentraIP deployment..."

# Delete Kubernetes resources
echo "☸️ Deleting Kubernetes resources..."
kubectl delete namespace tyk --ignore-not-found=true

# Wait for namespace deletion
echo "⏳ Waiting for namespace deletion..."
kubectl wait --for=delete namespace/tyk --timeout=120s || true

# Delete Docker images from GCR (optional)
read -p "Delete Docker images from GCR? (y/N): " -n 1 -r
echo
if [[ $REPLY =~ ^[Yy]$ ]]; then
    if [ -z "$PROJECT_ID" ]; then
        echo "❌ PROJECT_ID environment variable is required for image cleanup"
        exit 1
    fi
    
    echo "🐳 Deleting Docker images..."
    gcloud container images delete gcr.io/$PROJECT_ID/tyk-gateway-mcp:latest --quiet --force-delete-tags || true
    gcloud container images delete gcr.io/$PROJECT_ID/claude-mcp-client:latest --quiet --force-delete-tags || true
    echo "✅ Docker images deleted"
fi

# Clean local artifacts
echo "🧽 Cleaning local artifacts..."
rm -f src/tyk-plugin/*.so
rm -f deployment-info.txt

echo ""
echo "✅ Cleanup complete!"
echo "All Tyk MCP SentraIP resources have been removed."
