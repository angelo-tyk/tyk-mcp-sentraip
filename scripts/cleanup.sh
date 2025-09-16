echo
if [[ ! $REPLY =~ ^[Yy]$ ]]; then
    echo "Cleanup cancelled."
    exit 0
fi

# Delete the namespace (this removes all resources within it)
echo "Deleting Kubernetes resources..."
kubectl delete namespace tyk --ignore-not-found=true

# Wait for namespace deletion
echo "Waiting for namespace deletion to complete..."
kubectl wait --for=delete namespace/tyk --timeout=120s 2>/dev/null || echo "Namespace deletion completed or timed out"

# Optional: Clean up Docker images (uncomment if needed)
# echo "Removing Docker images..."
# if [[ -n "$PROJECT_ID" ]]; then
#     docker rmi gcr.io/$PROJECT_ID/tyk-gateway-mcp:latest 2>/dev/null || echo "Tyk Gateway image not found locally"
#     docker rmi gcr.io/$PROJECT_ID/claude-mcp-client:latest 2>/dev/null || echo "Claude MCP Client image not found locally"
# else
#     echo "PROJECT_ID not set, skipping Docker image cleanup"
# fi

# Optional: Clean up any local build artifacts
echo "Cleaning up local build artifacts..."
rm -f src/tyk-plugin/*.so 2>/dev/null || true

echo "Cleanup complete!"
echo "All Kubernetes resources in the 'tyk' namespace have been removed."
