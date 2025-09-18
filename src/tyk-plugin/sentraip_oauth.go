package main

import (
    "context"
    "encoding/base64"
    "encoding/json"
    "fmt"
    "io"
    "net/http"
    "net/url"
    "os"
    "strconv"
    "strings"
    "sync"
    "time"

    "github.com/TykTechnologies/tyk/ctx"
    "github.com/TykTechnologies/tyk/log"
    "github.com/sirupsen/logrus"
)

// TokenCache manages OAuth tokens with thread safety
type TokenCache struct {
    mutex     sync.RWMutex
    token     string
    expiresAt time.Time
}

// OAuthResponse represents SentraIP OAuth response
type OAuthResponse struct {
    AccessToken string `json:"access_token"`
    TokenType   string `json:"token_type"`
    ExpiresIn   int    `json:"expires_in"`
    Scope       string `json:"scope"`
}

var (
    sentraipTokenCache = &TokenCache{}
    logger             = log.Get()
)

// SentraIPOAuthMiddleware handles OAuth2 client credentials flow
func SentraIPOAuthMiddleware(rw http.ResponseWriter, r *http.Request) {
    // Get API configuration
    session := ctx.GetSession(r)
    spec := ctx.GetDefinition(r)
    
    clientID := os.Getenv("SENTRAIP_CLIENT_ID")
    clientSecret := os.Getenv("SENTRAIP_CLIENT_SECRET")
    
    if clientID == "" || clientSecret == "" {
        logger.Error("SentraIP OAuth credentials not configured")
        http.Error(rw, `{"error":"oauth_config_missing","message":"SentraIP credentials not configured"}`, 401)
        return
    }

    // Check if we need a fresh token
    token, err := getValidToken(clientID, clientSecret)
    if err != nil {
        logger.WithError(err).Error("Failed to obtain OAuth token")
        http.Error(rw, `{"error":"oauth_token_failure","message":"Unable to obtain SentraIP OAuth token"}`, 401)
        return
    }

    // Inject OAuth token into request
    r.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
    
    // Set context for tracing
    ctx.SetContext(r, "oauth.token_injected", "true")
    ctx.SetContext(r, "oauth.token_expires", strconv.FormatInt(sentraipTokenCache.expiresAt.Unix(), 10))
    ctx.SetContext(r, "oauth.client_id", clientID)
    
    logger.WithFields(logrus.Fields{
        "api_id": spec.APIID,
        "org_id": spec.OrgID,
        "token_expires": sentraipTokenCache.expiresAt.Format(time.RFC3339),
    }).Info("OAuth token injected successfully")
}

func getValidToken(clientID, clientSecret string) (string, error) {
    sentraipTokenCache.mutex.Lock()
    defer sentraipTokenCache.mutex.Unlock()
    
    // Check if current token is still valid (with 60 second buffer)
    if sentraipTokenCache.token != "" && time.Now().Before(sentraipTokenCache.expiresAt.Add(-60*time.Second)) {
        return sentraipTokenCache.token, nil
    }
    
    // Request new token
    tokenURL := "https://auth.sentraip.com/oauth/token"
    
    data := url.Values{}
    data.Set("grant_type", "client_credentials")
    data.Set("scope", "threat-intelligence")
    
    req, err := http.NewRequest("POST", tokenURL, strings.NewReader(data.Encode()))
    if err != nil {
        return "", fmt.Errorf("failed to create token request: %w", err)
    }
    
    // Set headers
    req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
    auth := base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%s:%s", clientID, clientSecret)))
    req.Header.Set("Authorization", fmt.Sprintf("Basic %s", auth))
    req.Header.Set("User-Agent", "Tyk-MCP-Gateway/1.0")
    
    // Make request
    client := &http.Client{
        Timeout: 10 * time.Second,
    }
    
    resp, err := client.Do(req)
    if err != nil {
        return "", fmt.Errorf("oauth request failed: %w", err)
    }
    defer resp.Body.Close()
    
    if resp.StatusCode != http.StatusOK {
        body, _ := io.ReadAll(resp.Body)
        return "", fmt.Errorf("oauth request failed with status %d: %s", resp.StatusCode, string(body))
    }
    
    // Parse response
    var oauthResp OAuthResponse
    if err := json.NewDecoder(resp.Body).Decode(&oauthResp); err != nil {
        return "", fmt.Errorf("failed to parse oauth response: %w", err)
    }
    
    // Update cache
    sentraipTokenCache.token = oauthResp.AccessToken
    sentraipTokenCache.expiresAt = time.Now().Add(time.Duration(oauthResp.ExpiresIn) * time.Second)
    
    logger.WithFields(logrus.Fields{
        "token_type": oauthResp.TokenType,
        "expires_in": oauthResp.ExpiresIn,
        "scope": oauthResp.Scope,
        "expires_at": sentraipTokenCache.expiresAt.Format(time.RFC3339),
    }).Info("New OAuth token obtained successfully")
    
    return oauthResp.AccessToken, nil
}

func init() {
    logger.Info("SentraIP OAuth middleware loaded")
}
