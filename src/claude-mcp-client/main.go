package main

import (
    "encoding/json"
    "fmt"
    "net/http"
    "os"
    "time"

    "github.com/gin-gonic/gin"
    "github.com/sirupsen/logrus"
    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/trace"
)

var (
    logger = logrus.New()
    tracer = otel.Tracer("claude-mcp-client")
)

type ClaudeRequest struct {
    Model     string    `json:"model"`
    MaxTokens int       `json:"max_tokens"`
    Messages  []Message `json:"messages"`
}

type Message struct {
    Role    string `json:"role"`
    Content string `json:"content"`
}

type ClaudeResponse struct {
    ID      string `json:"id"`
    Model   string `json:"model"`
    Content []struct {
        Type string `json:"type"`
        Text string `json:"text"`
    } `json:"content"`
    Usage struct {
        InputTokens  int `json:"input_tokens"`
        OutputTokens int `json:"output_tokens"`
    } `json:"usage"`
}

type MCPChatRequest struct {
    Message string `json:"message"`
}

type MCPChatResponse struct {
    Response    string                 `json:"response"`
    ToolsCalled []string              `json:"tools_called,omitempty"`
    Context     map[string]interface{} `json:"context,omitempty"`
    Timestamp   string                 `json:"timestamp"`
}

func main() {
    // Configure logger
    logger.SetLevel(logrus.InfoLevel)
    logger.SetFormatter(&logrus.JSONFormatter{})

    // Create Gin router
    r := gin.Default()

    // Health check endpoint
    r.GET("/health", func(c *gin.Context) {
        c.JSON(200, gin.H{
            "status":    "healthy",
            "service":   "claude-mcp-client",
            "timestamp": time.Now().Format(time.RFC3339),
        })
    })

    // MCP Chat endpoint
    r.POST("/chat", handleMCPChat)

    // Start server
    port := os.Getenv("PORT")
    if port == "" {
        port = "8080"
    }

    logger.WithField("port", port).Info("Starting Claude MCP Client server")
    r.Run(":" + port)
}

func handleMCPChat(c *gin.Context) {
    ctx, span := tracer.Start(c.Request.Context(), "mcp.chat")
    defer span.End()

    var req MCPChatRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        logger.WithError(err).Error("Failed to parse chat request")
        c.JSON(400, gin.H{"error": "Invalid request format"})
        return
    }

    // Log the incoming request
    logger.WithFields(logrus.Fields{
        "message_length": len(req.Message),
        "trace_id":      span.SpanContext().TraceID().String(),
    }).Info("Processing MCP chat request")

    // Process the message with Claude (mock implementation)
    response, err := processWithClaude(ctx, req.Message)
    if err != nil {
        logger.WithError(err).Error("Failed to process with Claude")
        c.JSON(500, gin.H{"error": "Failed to process request"})
        return
    }

    c.JSON(200, response)
}

func processWithClaude(ctx context.Context, message string) (*MCPChatResponse, error) {
    // This is a mock implementation
    // In a real implementation, this would:
    // 1. Parse the message for tool calls
    // 2. Execute MCP tools as needed
    // 3. Send structured request to Claude API
    // 4. Process Claude's response
    // 5. Return structured response

    response := &MCPChatResponse{
        Response:  "This is a mock response. In a real implementation, this would call Claude AI with MCP tool integration.",
        Timestamp: time.Now().Format(time.RFC3339),
        Context: map[string]interface{}{
            "message_processed": true,
            "tools_available": []string{
                "sentraip_threat_check",
                "tyk_api_analytics",
                "claude_context_search",
            },
        },
    }

    // Mock tool detection
    if containsThreatQuery(message) {
        response.ToolsCalled = append(response.ToolsCalled, "sentraip_threat_check")
        response.Response = "I've analyzed the IP address using SentraIP threat intelligence. The risk score is 2/10 (low risk). No known threats detected."
    }

    if containsAnalyticsQuery(message) {
        response.ToolsCalled = append(response.ToolsCalled, "tyk_api_analytics")
        response.Response = "I've retrieved your API analytics. Total requests: 1,523. Average response time: 245ms. Error rate: 2%."
    }

    return response, nil
}

func containsThreatQuery(message string) bool {
    threatKeywords := []string{"threat", "ip", "security", "malicious", "reputation"}
    for _, keyword := range threatKeywords {
        if contains(message, keyword) {
            return true
        }
    }
    return false
}

func containsAnalyticsQuery(message string) bool {
    analyticsKeywords := []string{"analytics", "performance", "metrics", "requests", "usage"}
    for _, keyword := range analyticsKeywords {
        if contains(message, keyword) {
            return true
        }
    }
    return false
}

func contains(s, substr string) bool {
    return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && (s[:len(substr)] == substr || s[len(s)-len(substr):] == substr || containsSubstring(s, substr)))
}

func containsSubstring(s, substr string) bool {
    for i := 0; i <= len(s)-len(substr); i++ {
        if s[i:i+len(substr)] == substr {
            return true
        }
    }
    return false
}
src/claude-mcp-client/config/config.go
package config

import (
    "os"
    "strconv"
)

type Config struct {
    Port              string
    ClaudeAPIKey      string
    ClaudeAPIURL      string
    SentraIPClientID  string
    SentraIPClientSecret string
    SentraIPAPIURL    string
    TykGatewayURL     string
    LogLevel          string
    OTELEndpoint      string
}

func Load() *Config {
    return &Config{
        Port:                 getEnv("PORT", "8080"),
        ClaudeAPIKey:         getEnv("CLAUDE_API_KEY", ""),
        ClaudeAPIURL:         getEnv("CLAUDE_API_URL", "https://api.anthropic.com/v1/messages"),
        SentraIPClientID:     getEnv("SENTRAIP_CLIENT_ID", ""),
        SentraIPClientSecret: getEnv("SENTRAIP_CLIENT_SECRET", ""),
        SentraIPAPIURL:       getEnv("SENTRAIP_API_URL", "https://api.sentraip.com"),
        TykGatewayURL:        getEnv("TYK_GATEWAY_URL", "http://tyk-gateway:8080"),
        LogLevel:             getEnv("LOG_LEVEL", "info"),
        OTELEndpoint:         getEnv("OTEL_EXPORTER_OTLP_ENDPOINT", "http://otel-collector:4317"),
    }
}

func getEnv(key, defaultValue string) string {
    if value := os.Getenv(key); value != "" {
        return value
    }
    return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
    if value := os.Getenv(key); value != "" {
        if intValue, err := strconv.Atoi(value); err == nil {
            return intValue
        }
    }
    return defaultValue
}

func getEnvBool(key string, defaultValue bool) bool {
    if value := os.Getenv(key); value != "" {
        if boolValue, err := strconv.ParseBool(value); err == nil {
            return boolValue
        }
    }
    return defaultValue
}
