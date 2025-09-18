#!/bin/bash
set -e

echo "Building Tyk Go Plugins..."

# Set Go build environment
export CGO_ENABLED=1
export GOOS=linux
export GOARCH=amd64

# Build plugins as shared libraries
echo "Building SentraIP OAuth plugin..."
go build -buildmode=plugin -o sentraip_oauth.so sentraip_oauth.go

echo "Building OTEL enhancer plugin..."
go build -buildmode=plugin -o tyk_otel_enhancer.so tyk_otel_enhancer.go

echo "Building MCP tools plugin..."
go build -buildmode=plugin -o tyk_mcp_tools.so tyk_mcp_tools.go

echo "Go plugins built successfully!"
ls -la *.so
