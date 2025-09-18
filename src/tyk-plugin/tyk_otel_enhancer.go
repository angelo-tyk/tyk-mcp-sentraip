package main

import (
    "context"
    "encoding/json"
    "fmt"
    "io"
    "net"
    "net/http"
    "regexp"
    "strconv"
    "strings"
    "time"

    "github.com/TykTechnologies/tyk/ctx"
    "github.com/TykTechnologies/tyk/log"
    "github.com/TykTechnologies/tyk/user"
    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/attribute"
    "go.opentelemetry.io/otel/codes"
    "go.opentelemetry.io/otel/trace"
)

// ClaudeRequest represents Claude API request structure
type ClaudeRequest struct {
    Model     string    `json:"model"`
    MaxTokens int       `json:"max_tokens"`
    Messages  []Message `json:"messages"`
}

type Message struct {
    Role    string `json:"role"`
    Content string `json:"content"`
}

// ClaudeResponse represents Claude API response structure
type ClaudeResponse struct {
    ID      string `json:"id"`
    Model   string `json:"model"`
    Content []struct {
        Type  string                 `json:"type"`
        Name  string                 `json:"name,omitempty"`
        Input map[string]interface{} `json:"input,omitempty"`
        Text  string                 `json:"text,omitempty"`
    } `json:"content"`
    Usage struct {
        InputTokens  int `json:"input_tokens"`
        OutputTokens int `json:"output_tokens"`
    } `json:"usage"`
}

var (
    tracer    = otel.Tracer("tyk-mcp-gateway")
    toolRegex = regexp.MustCompile(`tool:\s*(\w+)`)
)

// TykOTELPreMiddleware handles request tracing setup
func TykOTELPreMiddleware(rw http.ResponseWriter, r *http.Request) {
    // Start timing
    ctx.SetContext(r, "request.start_time", strconv.FormatInt(time.Now().UnixNano(), 10))
    
    session := ctx.GetSession(r)
    spec := ctx.GetDefinition(r)
    
    // Create span context
    spanCtx, span := tracer.Start(r.Context(), fmt.Sprintf("tyk.%s", spec.Name))
    r = r.WithContext(spanCtx)
    
    // Base trace attributes
    attrs := []attribute.KeyValue{
        attribute.String("tyk.gateway_id", spec.OrgID),
        attribute.String("tyk.api_name", spec.Name),
        attribute.String("tyk.api_id", spec.APIID),
        attribute.String("http.method", r.Method),
        attribute.String("http.url", r.RequestURI),
        attribute.String("http.user_agent", r.UserAgent()),
        attribute.String("client.ip", getClientIP(r)),
        attribute.String("request.timestamp", time.Now().Format(time.RFC3339)),
    }
    
    // MCP-specific tracing
    if mcpTool := r.Header.Get("X-MCP-Tool"); mcpTool != "" {
        attrs = append(attrs,
            attribute.String("mcp.tool", mcpTool),
            attribute.String("mcp.tool_version", r.Header.Get("X-MCP-Tool-Version")),
            attribute.String("mcp.session_id", getSessionID(session)),
        )
        
        // Set context for downstream processing
        ctx.SetContext(r, "mcp.tool", mcpTool)
    }
    
    // Claude AI specific tracing
    if strings.Contains(r.RequestURI, "/claude/") {
        if body, err := io.ReadAll(r.Body); err == nil {
            // Restore body for downstream processing
            r.Body = io.NopCloser(strings.NewReader(string(body)))
            
            var claudeReq ClaudeRequest
            if json.Unmarshal(body, &claudeReq) == nil {
                attrs = append(attrs,
                    attribute.String("claude.model", claudeReq.Model),
                    attribute.Int("claude.max_tokens", claudeReq.MaxTokens),
                    attribute.Int("claude.message_count", len(claudeReq.Messages)),
                    attribute.Int("claude.request_size", len(body)),
                )
                
                // Extract MCP context from Claude messages
                if len(claudeReq.Messages) > 0 {
                    lastMessage := claudeReq.Messages[len(claudeReq.Messages)-1]
                    if matches := toolRegex.FindStringSubmatch(lastMessage.Content); len(matches) > 1 {
                        attrs = append(attrs, attribute.String("mcp.referenced_tool", matches[1]))
                        ctx.SetContext(r, "mcp.referenced_tool", matches[1])
                    }
                }
            } else {
                attrs = append(attrs, attribute.String("claude.parse_error", err.Error()))
            }
        }
    }
    
    // SentraIP specific tracing
    if strings.Contains(r.RequestURI, "/threat-intel/") {
        urlParts := strings.Split(strings.TrimPrefix(r.RequestURI, "/threat-intel/"), "/")
        if len(urlParts) >= 2 {
            attrs = append(attrs,
                attribute.String("sentraip.endpoint_type", urlParts[0]),
                attribute.String("sentraip.query_target", urlParts[1]),
                attribute.String("sentraip.client_version", r.Header.Get("X-Client-Version")),
            )
        }
    }
    
    // OAuth context
    if r.Header.Get("Authorization") != "" {
        attrs = append(attrs, attribute.Bool("oauth.token_present", true))
        if tokenExpires := ctx.GetContextData(r, "oauth.token_expires"); tokenExpires != nil {
            attrs = append(attrs, attribute.String("oauth.token_expires", tokenExpires.(string)))
        }
    }
    
    // Set all attributes on span
    span.SetAttributes(attrs...)
    
    // Store span in context for response middleware
    ctx.SetContext(r, "otel.span", span)
}

// TykOTELPostMiddleware handles response tracing
func TykOTELPostMiddleware(rw http.ResponseWriter, r *http.Request, res *http.Response) {
    // Retrieve span from context
    spanInterface := ctx.GetContextData(r, "otel.span")
    if spanInterface == nil {
        return
    }
    
    span, ok := spanInterface.(trace.Span)
    if !ok {
        return
    }
    defer span.End()
    
    // Calculate request duration
    startTimeStr := ctx.GetContextData(r, "request.start_time")
    var duration int64
    if startTimeStr != nil {
        if startTime, err := strconv.ParseInt(startTimeStr.(string), 10, 64); err == nil {
            duration = (time.Now().UnixNano() - startTime) / int64(time.Millisecond)
        }
    }
    
    // Response attributes
    responseAttrs := []attribute.KeyValue{
        attribute.Int64("request.duration_ms", duration),
        attribute.Int("response.status_code", res.StatusCode),
        attribute.Int("response.size_bytes", int(res.ContentLength)),
    }
    
    // Process SentraIP intelligence response
    if riskScore := res.Header.Get("X-SentraIP-Risk-Score"); riskScore != "" {
        if score, err := strconv.Atoi(riskScore); err == nil {
            responseAttrs = append(responseAttrs,
                attribute.Int("sentraip.risk_score", score),
                attribute.String("sentraip.threat_types", res.Header.Get("X-SentraIP-Threats")),
                attribute.String("sentraip.last_updated", res.Header.Get("X-Last-Updated")),
                attribute.String("sentraip.data_source", res.Header.Get("X-Data-Source")),
            )
        }
    }
    
    // Process Claude AI response
    if strings.Contains(r.RequestURI, "/claude/") && res.StatusCode == 200 {
        if body, err := io.ReadAll(res.Body); err == nil {
            // Restore body for client
            res.Body = io.NopCloser(strings.NewReader(string(body)))
            
            var claudeResp ClaudeResponse
            if json.Unmarshal(body, &claudeResp) == nil {
                responseAttrs = append(responseAttrs,
                    attribute.String("claude.response_id", claudeResp.ID),
                    attribute.String("claude.model_used", claudeResp.Model),
                    attribute.Int("claude.input_tokens", claudeResp.Usage.InputTokens),
                    attribute.Int("claude.output_tokens", claudeResp.Usage.OutputTokens),
                    attribute.Int("claude.total_tokens", claudeResp.Usage.InputTokens+claudeResp.Usage.OutputTokens),
                )
                
                // Check if Claude used MCP tools
                for _, content := range claudeResp.Content {
                    if content.Type == "tool_use" && content.Name != "" {
                        responseAttrs = append(responseAttrs,
                            attribute.String("claude.tool_used", content.Name),
                        )
                        if inputJSON, err := json.Marshal(content.Input); err == nil {
                            responseAttrs = append(responseAttrs,
                                attribute.String("claude.tool_input", string(inputJSON)),
                            )
                        }
                        break
                    }
                }
            } else {
                responseAttrs = append(responseAttrs, attribute.String("claude.response_parse_error", err.Error()))
            }
        }
    }
    
    // Error analysis
    if res.StatusCode >= 400 {
        errorCategory := "client_error"
        if res.StatusCode >= 500 {
            errorCategory = "server_error"
        }
        
        responseAttrs = append(responseAttrs,
            attribute.String("error.category", errorCategory),
            attribute.String("error.description", getErrorDescription(res.StatusCode)),
            attribute.Bool("error.occurred", true),
        )
        
        span.RecordError(fmt.Errorf("HTTP %d: %s", res.StatusCode, getErrorDescription(res.StatusCode)))
    }
    
    // Performance categorization
    if duration > 0 {
        perfCategory := "fast"
        if duration > 5000 {
            perfCategory = "slow"
        } else if duration > 2000 {
            perfCategory = "medium"
        }
        responseAttrs = append(responseAttrs, attribute.String("performance.category", perfCategory))
    }
    
    // Set response attributes
    span.SetAttributes(responseAttrs...)
    
    // Set span status based on response
    if res.StatusCode >= 400 {
        span.SetStatus(codes.Error, fmt.Sprintf("HTTP %d", res.StatusCode))
    } else {
        span.SetStatus(codes.Ok, "Request completed successfully")
    }
}

func getClientIP(r *http.Request) string {
    // Check X-Forwarded-For header first
    if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
        ips := strings.Split(xff, ",")
        return strings.TrimSpace(ips[0])
    }
    
    // Check X-Real-IP header
    if xri := r.Header.Get("X-Real-IP"); xri != "" {
        return xri
    }
    
    // Fall back to RemoteAddr
    if ip, _, err := net.SplitHostPort(r.RemoteAddr); err == nil {
        return ip
    }
    
    return r.RemoteAddr
}

func getSessionID(session *user.SessionState) string {
    if session != nil && session.KeyID != "" {
        return session.KeyID
    }
    return "anonymous"
}

func getErrorDescription(statusCode int) string {
    errorMap := map[int]string{
        400: "bad_request",
        401: "authentication_failed",
        403: "authorization_failed",
        404: "resource_not_found",
        429: "rate_limit_exceeded",
        500: "internal_server_error",
        502: "bad_gateway",
        503: "service_unavailable",
        504: "gateway_timeout",
    }
    
    if desc, exists := errorMap[statusCode]; exists {
        return desc
    }
    return "unknown_error"
}

func init() {
    logger.Info("Tyk OTEL Enhancer middleware loaded")
}
