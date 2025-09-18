package main

import (
    "encoding/json"
    "fmt"
    "io"
    "net/http"
    "os"
    "strconv"
    "strings"
    "time"

    "github.com/TykTechnologies/tyk/ctx"
    "github.com/TykTechnologies/tyk/log"
    "github.com/TykTechnologies/tyk/user"
    "github.com/sirupsen/logrus"
)

// MCPTool represents an MCP tool definition
type MCPTool struct {
    Name        string      `json:"name"`
    Description string      `json:"description"`
    InputSchema InputSchema `json:"inputSchema"`
}

// InputSchema represents the JSON schema for tool input
type InputSchema struct {
    Type       string              `json:"type"`
    Properties map[string]Property `json:"properties"`
    Required   []string            `json:"required"`
}

// Property represents a schema property
type Property struct {
    Type        string   `json:"type"`
    Description string   `json:"description"`
    Enum        []string `json:"enum,omitempty"`
}

// MCPToolsRegistry holds all available MCP tools
var MCPToolsRegistry = map[string]MCPTool{
    "sentraip_threat_check": {
        Name:        "sentraip_threat_check",
        Description: "Check IP address or domain reputation using SentraIP",
        InputSchema: InputSchema{
            Type: "object",
            Properties: map[string]Property{
                "target": {
                    Type:        "string",
                    Description: "IP address or domain to check",
                },
                "type": {
                    Type:        "string",
                    Description: "Type of target to check",
                    Enum:        []string{"ip", "domain"},
                },
            },
            Required: []string{"target", "type"},
        },
    },
    "tyk_api_analytics": {
        Name:        "tyk_api_analytics",
        Description: "Get API usage analytics from Tyk Gateway",
        InputSchema: InputSchema{
            Type: "object",
            Properties: map[string]Property{
                "api_id": {
                    Type:        "string",
                    Description: "API ID to analyze",
                },
                "time_range": {
                    Type:        "string",
                    Description: "Time range (24h, 7d, 30d)",
                },
            },
            Required: []string{"api_id"},
        },
    },
    "claude_context_search": {
        Name:        "claude_context_search",
        Description: "Search previous conversations and context",
        InputSchema: InputSchema{
            Type: "object",
            Properties: map[string]Property{
                "query": {
                    Type:        "string",
                    Description: "Search query",
                },
                "limit": {
                    Type:        "string",
                    Description: "Number of results",
                },
            },
            Required: []string{"query"},
        },
    },
}

// MCPToolsMiddleware handles MCP tools listing and execution
func MCPToolsMiddleware(rw http.ResponseWriter, r *http.Request) {
    session := ctx.GetSession(r)
    spec := ctx.GetDefinition(r)
    
    logger.WithFields(logrus.Fields{
        "method": r.Method,
        "path":   r.RequestURI,
        "api_id": spec.APIID,
    }).Info("MCP tools request received")
    
    // Handle MCP tools listing
    if r.RequestURI == "/mcp/tools" && r.Method == "GET" {
        handleToolsList(rw, r, session)
        return
    }
    
    // Handle MCP tool execution
    if strings.HasPrefix(r.RequestURI, "/mcp/call/") && r.Method == "POST" {
        toolName := strings.TrimPrefix(r.RequestURI, "/mcp/call/")
        handleToolExecution(rw, r, session, toolName)
        return
    }
    
    // If no MCP endpoint matched, continue to next middleware
}

func handleToolsList(rw http.ResponseWriter, r *http.Request, session *user.SessionState) {
    tools := make([]MCPTool, 0, len(MCPToolsRegistry))
    for _, tool := range MCPToolsRegistry {
        tools = append(tools, tool)
    }
    
    response := map[string]interface{}{
        "tools":     tools,
        "count":     len(tools),
        "timestamp": time.Now().Format(time.RFC3339),
    }
    
    rw.Header().Set("Content-Type", "application/json")
    rw.WriteHeader(http.StatusOK)
    json.NewEncoder(rw).Encode(response)
    
    logger.WithFields(logrus.Fields{
        "tools_count": len(tools),
        "session_id":  getSessionID(session),
    }).Info("MCP tools list served")
}

func handleToolExecution(rw http.ResponseWriter, r *http.Request, session *user.SessionState, toolName string) {
    tool, exists := MCPToolsRegistry[toolName]
    if !exists {
        logger.WithField("tool_name", toolName).Error("MCP tool not found")
        http.Error(rw, fmt.Sprintf(`{"error":"tool_not_found","message":"Tool not found: %s"}`, toolName), http.StatusNotFound)
        return
    }
    
    // Parse request body
    body, err := io.ReadAll(r.Body)
    if err != nil {
        logger.WithError(err).Error("Failed to read MCP tool request body")
        http.Error(rw, `{"error":"invalid_request","message":"Failed to read request body"}`, http.StatusBadRequest)
        return
    }
    
    var params map[string]interface{}
    if err := json.Unmarshal(body, &params); err != nil {
        logger.WithError(err).Error("Failed to parse MCP tool request JSON")
        http.Error(rw, `{"error":"invalid_json","message":"Invalid JSON in request body"}`, http.StatusBadRequest)
        return
    }
    
    // Execute the tool
    result, err := executeMCPTool(toolName, params, session)
    if err != nil {
        logger.WithError(err).WithField("tool_name", toolName).Error("MCP tool execution failed")
        http.Error(rw, fmt.Sprintf(`{"error":"execution_failed","message":"%s"}`, err.Error()), http.StatusInternalServerError)
        return
    }
    
    rw.Header().Set("Content-Type", "application/json")
    rw.WriteHeader(http.StatusOK)
    json.NewEncoder(rw).Encode(result)
    
    logger.WithFields(logrus.Fields{
        "tool_name":     toolName,
        "session_id":    getSessionID(session),
        "params_count":  len(params),
    }).Info("MCP tool executed successfully")
}

func executeMCPTool(toolName string, params map[string]interface{}, session *user.SessionState) (map[string]interface{}, error) {
    switch toolName {
    case "sentraip_threat_check":
        return callSentraIPAPI(params, session)
    case "tyk_api_analytics":
        return getTykAnalytics(params, session)
    case "claude_context_search":
        return searchClaudeContext(params, session)
    default:
        return nil, fmt.Errorf("unknown tool: %s", toolName)
    }
}

func callSentraIPAPI(params map[string]interface{}, session *user.SessionState) (map[string]interface{}, error) {
    target, ok := params["target"].(string)
    if !ok {
        return nil, fmt.Errorf("missing or invalid 'target' parameter")
    }
    
    targetType, ok := params["type"].(string)
    if !ok {
        return nil, fmt.Errorf("missing or invalid 'type' parameter")
    }
    
    var endpoint string
    switch targetType {
    case "ip":
        endpoint = fmt.Sprintf("/threat-intel/ip/%s", target)
    case "domain":
        endpoint = fmt.Sprintf("/threat-intel/domain/%s", target)
    default:
        return nil, fmt.Errorf("invalid type: %s (must be 'ip' or 'domain')", targetType)
    }
    
    // Get OAuth token (this would use the token from the OAuth middleware)
    token := os.Getenv("SENTRAIP_OAUTH_TOKEN") // In real implementation, get from token cache
    
    // Create HTTP request to SentraIP via Tyk Gateway
    client := &http.Client{Timeout: 10 * time.Second}
    req, err := http.NewRequest("GET", "http://tyk-gateway:8080/sentraip"+endpoint, nil)
    if err != nil {
        return nil, fmt.Errorf("failed to create SentraIP request: %w", err)
    }
    
    req.Header.Set("Authorization", "Bearer "+token)
    req.Header.Set("User-Agent", "Tyk-MCP-Gateway/1.0")
    req.Header.Set("X-MCP-Tool", "sentraip_threat_check")
    
    resp, err := client.Do(req)
    if err != nil {
        return nil, fmt.Errorf("SentraIP API request failed: %w", err)
    }
    defer resp.Body.Close()
    
    if resp.StatusCode != http.StatusOK {
        return map[string]interface{}{
            "success": false,
            "error":   fmt.Sprintf("SentraIP API error: %d", resp.StatusCode),
            "target":  target,
            "type":    targetType,
        }, nil
    }
    
    var sentraipResponse map[string]interface{}
    if err := json.NewDecoder(resp.Body).Decode(&sentraipResponse); err != nil {
        return nil, fmt.Errorf("failed to parse SentraIP response: %w", err)
    }
    
    return map[string]interface{}{
        "success":   true,
        "data":      sentraipResponse,
        "target":    target,
        "type":      targetType,
        "timestamp": time.Now().Format(time.RFC3339),
    }, nil
}

func getTykAnalytics(params map[string]interface{}, session *user.SessionState) (map[string]interface{}, error) {
    apiID, ok := params["api_id"].(string)
    if !ok {
        return nil, fmt.Errorf("missing or invalid 'api_id' parameter")
    }
    
    timeRange := "24h"
    if tr, exists := params["time_range"]; exists {
        if trStr, ok := tr.(string); ok {
            timeRange = trStr
        }
    }
    
    // In a real implementation, this would query actual Tyk analytics data
    // For demo purposes, returning mock data
    mockAnalytics := map[string]interface{}{
        "api_id":              apiID,
        "time_range":         timeRange,
        "total_requests":     1523,
        "successful_requests": 1492,
        "error_requests":     31,
        "avg_response_time":  245.7,
        "error_rate":         0.02,
        "top_endpoints": []map[string]interface{}{
            {"path": "/v1/messages", "requests": 756},
            {"path": "/threat-intel/ip", "requests": 432},
            {"path": "/threat-intel/domain", "requests": 335},
        },
        "status_codes": map[string]int{
            "200": 1492,
            "400": 15,
            "401": 8,
            "500": 8,
        },
        "timestamp": time.Now().Format(time.RFC3339),
    }
    
    logger.WithFields(logrus.Fields{
        "api_id":     apiID,
        "time_range": timeRange,
        "session_id": getSessionID(session),
    }).Info("Tyk analytics data served")
    
    return mockAnalytics, nil
}

func searchClaudeContext(params map[string]interface{}, session *user.SessionState) (map[string]interface{}, error) {
    query, ok := params["query"].(string)
    if !ok {
        return nil, fmt.Errorf("missing or invalid 'query' parameter")
    }
    
    limit := 10
    if l, exists := params["limit"]; exists {
        if lStr, ok := l.(string); ok {
            if parsedLimit, err := strconv.Atoi(lStr); err == nil && parsedLimit > 0 {
                limit = parsedLimit
            }
        }
    }
    
    // In a real implementation, this would search actual conversation history
    // For demo purposes, returning mock search results
    mockResults := map[string]interface{}{
        "query": query,
        "limit": limit,
        "results": []map[string]interface{}{
            {
                "conversation_id": "conv_123abc",
                "snippet":        fmt.Sprintf("Previous discussion about %s and API security best practices...", query),
                "timestamp":      "2024-01-15T10:30:00Z",
                "relevance":      0.95,
            },
            {
                "conversation_id": "conv_456def",
                "snippet":        fmt.Sprintf("User asked about %s implementation details...", query),
                "timestamp":      "2024-01-14T15:45:00Z",
                "relevance":      0.87,
            },
        },
        "total_matches": 2,
        "search_time_ms": 45,
        "timestamp": time.Now().Format(time.RFC3339),
    }
    
    logger.WithFields(logrus.Fields{
        "query":      query,
        "limit":      limit,
        "session_id": getSessionID(session),
    }).Info("Claude context search completed")
    
    return mockResults, nil
}

func init() {
    logger.WithField("tools_count", len(MCPToolsRegistry)).Info("Tyk MCP Tools middleware loaded")
}
