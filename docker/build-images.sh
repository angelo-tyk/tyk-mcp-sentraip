#!/bin/bash
set -e

PROJECT_ID=${PROJECT_ID:-"your-gcp-project"}

echo "Building Tyk Gateway with MCP plugin..."
docker build -f docker/Dockerfile.gateway -t tyk-gateway-mcp:latest .
docker tag tyk-gateway-mcp:latest gcr.io/${PROJECT_ID}/tyk-gateway-mcp:latest

echo "Building Claude MCP Client..."
docker build -f docker/Dockerfile.mcp-client -t claude-mcp-client:latest .
docker tag claude-mcp-client:latest gcr.io/${PROJECT_ID}/claude-mcp-client:latest
