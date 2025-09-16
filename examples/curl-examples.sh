#!/bin/bash

# Set your service IPs
TYK_IP="your-tyk-gateway-ip"
CLAUDE_IP="your-claude-mcp-client-ip"

echo "=== Tyk MCP Gateway Examples ==="

# List available MCP tools
echo "1. List MCP Tools:"
curl -X GET "http://$TYK_IP:8080/mcp/tools"

# Check single IP reputation  
echo -e "\n2. Check IP Reputation:"
curl -X POST "http://$TYK_IP:8080/mcp/execute" \
  -H "Content-Type: application/json" \
  -d '{
    "tool": "check_ip_reputation",
    "parameters": {"ip": "185.220.101.45"}
  }'

# Bulk IP analysis
echo -e "\n3. Bulk IP Analysis:"
curl -X POST "http://$TYK_IP:8080/mcp/execute" \
  -H "Content-Type: application/json" \
  -d '{
    "tool": "bulk_ip_analysis",
    "parameters": {
      "ips": ["185.220.101.45", "192.168.1.100", "8.8.8.8"]
    }
  }'

echo -e "\n=== Claude Conversational Examples ==="

# Simple threat check
echo -e "\n4. Conversational Threat Check:"
curl -X POST "http://$CLAUDE_IP:8080/chat" \
  -H "Content-Type: application/json" \
  -d '{
    "message": "Is IP 185.220.101.45 dangerous?"
  }'

# Pattern analysis
echo -e "\n5. Pattern Analysis Query:"
curl -X POST "http://$CLAUDE_IP:8080/chat" \
  -H "Content-Type: application/json" \
  -d '{
    "message": "Analyze these IPs for attack patterns and recommend blocking: 185.220.101.45, 192.168.1.100"
  }'

# Geographic threat analysis
echo -e "\n6. Geographic Analysis:"
curl -X POST "http://$CLAUDE_IP:8080/chat" \
  -H "Content-Type: application/json" \
  -d '{
    "message": "Show me threat patterns from Russian IP ranges in the last hour"
  }'
