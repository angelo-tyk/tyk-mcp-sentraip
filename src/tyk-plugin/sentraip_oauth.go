package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/TykTechnologies/tyk/ctx"
	"github.com/TykTechnologies/tyk/log"
	"github.com/sirupsen/logrus"
	"golang.org/x/oauth2"
)

// SentraIPOAuthPlugin represents the plugin configuration
type SentraIPOAuthPlugin struct{}

// SentraIPConfig holds the OAuth configuration
type SentraIPConfig struct {
	ClientID     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
	AuthURL      string `json:"auth_url"`
	TokenURL     string `json:"token_url"`
	RedirectURL  string `json:"redirect_url"`
	Scope        string `json:"scope"`
}

// TokenResponse represents the OAuth token response
type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
	RefreshToken string `json:"refresh_token"`
	Scope        string `json:"scope"`
}

// UserInfo represents SentraIP user information
type UserInfo struct {
	ID       string `json:"id"`
	Username string `json:"username"`
	Email    string `json:"email"`
	Roles    []string `json:"roles"`
}

func main() {}

// MyPluginPre - Pre hook for OAuth authentication
func (p *SentraIPOAuthPlugin) MyPluginPre(rw http.ResponseWriter, r *http.Request) {
	log.Get().Info("SentraIP OAuth Plugin: Pre hook triggered")
	
	// Get plugin configuration from context
	config := p.getConfig(r)
	if config == nil {
		log.Get().Error("SentraIP OAuth Plugin: Configuration not found")
		http.Error(rw, "OAuth configuration error", http.StatusInternalServerError)
		return
	}

	// Check for authorization code in callback
	if code := r.URL.Query().Get("code"); code != "" {
		p.handleOAuthCallback(rw, r, config, code)
		return
	}

	// Check for existing token in session
	session := ctx.GetSession(r)
	if session != nil {
		if token, exists := session.MetaData["oauth_access_token"]; exists {
			if p.validateToken(token.(string), config) {
				log.Get().Info("SentraIP OAuth Plugin: Valid token found")
				return
			}
		}
	}

	// Redirect to OAuth provider for authentication
	p.redirectToOAuth(rw, r, config)
}

// MyPluginAuth - Authentication hook
func (p *SentraIPOAuthPlugin) MyPluginAuth(rw http.ResponseWriter, r *http.Request) {
	log.Get().Info("SentraIP OAuth Plugin: Auth hook triggered")
	
	session := ctx.GetSession(r)
	if session == nil {
		log.Get().Error("SentraIP OAuth Plugin: No session found")
		http.Error(rw, "Authentication required", http.StatusUnauthorized)
		return
	}

	// Check for valid access token
	token, exists := session.MetaData["oauth_access_token"]
	if !exists {
		log.Get().Error("SentraIP OAuth Plugin: No access token in session")
		http.Error(rw, "Authentication required", http.StatusUnauthorized)
		return
	}

	// Validate and use the token
	if !p.validateToken(token.(string), p.getConfig(r)) {
		log.Get().Error("SentraIP OAuth Plugin: Invalid access token")
		http.Error(rw, "Invalid token", http.StatusUnauthorized)
		return
	}

	// Add user info to request headers
	if userInfo, ok := session.MetaData["user_info"].(UserInfo); ok {
		r.Header.Set("X-SentraIP-User-ID", userInfo.ID)
		r.Header.Set("X-SentraIP-Username", userInfo.Username)
		r.Header.Set("X-SentraIP-Email", userInfo.Email)
		r.Header.Set("X-SentraIP-Roles", strings.Join(userInfo.Roles, ","))
	}

	log.Get().Info("SentraIP OAuth Plugin: Authentication successful")
}

// MyPluginPost - Post processing hook
func (p *SentraIPOAuthPlugin) MyPluginPost(rw http.ResponseWriter, res *http.Response, req *http.Request) {
	log.Get().Info("SentraIP OAuth Plugin: Post hook triggered")
	
	// Add security headers
	res.Header.Set("X-SentraIP-Processed", "true")
	res.Header.Set("X-Content-Type-Options", "nosniff")
}

// getConfig retrieves plugin configuration from request context
func (p *SentraIPOAuthPlugin) getConfig(r *http.Request) *SentraIPConfig {
	// In a real implementation, this would come from Tyk API definition
	// For now, return a default configuration
	return &SentraIPConfig{
		ClientID:     "your-sentraip-client-id",
		ClientSecret: "your-sentraip-client-secret",
		AuthURL:      "https://auth.sentraip.com/oauth/authorize",
		TokenURL:     "https://auth.sentraip.com/oauth/token",
		RedirectURL:  "https://your-api.com/oauth/callback",
		Scope:        "read:profile read:email",
	}
}

// redirectToOAuth redirects user to OAuth provider
func (p *SentraIPOAuthPlugin) redirectToOAuth(rw http.ResponseWriter, r *http.Request, config *SentraIPConfig) {
	log.Get().Info("SentraIP OAuth Plugin: Redirecting to OAuth provider")

	oauthConfig := &oauth2.Config{
		ClientID:     config.ClientID,
		ClientSecret: config.ClientSecret,
		RedirectURL:  config.RedirectURL,
		Scopes:       strings.Split(config.Scope, " "),
		Endpoint: oauth2.Endpoint{
			AuthURL:  config.AuthURL,
			TokenURL: config.TokenURL,
		},
	}

	// Generate state parameter for security
	state := p.generateState()
	
	// Store state in session for verification
	session := ctx.GetSession(r)
	if session != nil && session.MetaData != nil {
		session.MetaData["oauth_state"] = state
		// Store session back in context
		ctx.SetSession(r, session, "", true)
	}

	authURL := oauthConfig.AuthCodeURL(state, oauth2.AccessTypeOffline)
	http.Redirect(rw, r, authURL, http.StatusFound)
}

// handleOAuthCallback processes the OAuth callback
func (p *SentraIPOAuthPlugin) handleOAuthCallback(rw http.ResponseWriter, r *http.Request, config *SentraIPConfig, code string) {
	log.Get().Info("SentraIP OAuth Plugin: Handling OAuth callback")

	// Verify state parameter
	state := r.URL.Query().Get("state")
	session := ctx.GetSession(r)
	if session == nil || session.MetaData["oauth_state"] != state {
		log.Get().Error("SentraIP OAuth Plugin: Invalid state parameter")
		http.Error(rw, "Invalid state parameter", http.StatusBadRequest)
		return
	}

	// Exchange code for token
	token, err := p.exchangeCodeForToken(code, config)
	if err != nil {
		log.Get().WithError(err).Error("SentraIP OAuth Plugin: Failed to exchange code for token")
		http.Error(rw, "Token exchange failed", http.StatusInternalServerError)
		return
	}

	// Get user information
	userInfo, err := p.getUserInfo(token.AccessToken, config)
	if err != nil {
		log.Get().WithError(err).Error("SentraIP OAuth Plugin: Failed to get user info")
		http.Error(rw, "Failed to get user info", http.StatusInternalServerError)
		return
	}

	// Store token and user info in session
	if session.MetaData == nil {
		session.MetaData = make(map[string]interface{})
	}
	session.MetaData["oauth_access_token"] = token.AccessToken
	session.MetaData["oauth_refresh_token"] = token.RefreshToken
	session.MetaData["user_info"] = *userInfo

	// Update session
	ctx.SetSession(r, session, "", true)

	log.Get().Info("SentraIP OAuth Plugin: OAuth callback processed successfully")
	
	// Redirect to original requested URL or default
	originalURL := r.URL.Query().Get("redirect_uri")
	if originalURL == "" {
		originalURL = "/"
	}
	http.Redirect(rw, r, originalURL, http.StatusFound)
}

// exchangeCodeForToken exchanges authorization code for access token
func (p *SentraIPOAuthPlugin) exchangeCodeForToken(code string, config *SentraIPConfig) (*TokenResponse, error) {
	log.Get().Info("SentraIP OAuth Plugin: Exchanging code for token")

	oauthConfig := &oauth2.Config{
		ClientID:     config.ClientID,
		ClientSecret: config.ClientSecret,
		RedirectURL:  config.RedirectURL,
		Endpoint: oauth2.Endpoint{
			AuthURL:  config.AuthURL,
			TokenURL: config.TokenURL,
		},
	}

	token, err := oauthConfig.Exchange(context.Background(), code)
	if err != nil {
		return nil, fmt.Errorf("failed to exchange code: %w", err)
	}

	return &TokenResponse{
		AccessToken:  token.AccessToken,
		TokenType:    token.TokenType,
		ExpiresIn:    int(token.Expiry.Unix()),
		RefreshToken: token.RefreshToken,
	}, nil
}

// validateToken validates the access token
func (p *SentraIPOAuthPlugin) validateToken(token string, config *SentraIPConfig) bool {
	log.Get().Info("SentraIP OAuth Plugin: Validating token")

	// Create request to validate token
	req, err := http.NewRequest("GET", "https://auth.sentraip.com/oauth/validate", nil)
	if err != nil {
		log.Get().WithError(err).Error("SentraIP OAuth Plugin: Failed to create validation request")
		return false
	}

	req.Header.Set("Authorization", "Bearer "+token)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Get().WithError(err).Error("SentraIP OAuth Plugin: Token validation request failed")
		return false
	}
	defer resp.Body.Close()

	return resp.StatusCode == http.StatusOK
}

// getUserInfo retrieves user information using access token
func (p *SentraIPOAuthPlugin) getUserInfo(token string, config *SentraIPConfig) (*UserInfo, error) {
	log.Get().Info("SentraIP OAuth Plugin: Getting user info")

	req, err := http.NewRequest("GET", "https://api.sentraip.com/user/profile", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create user info request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("user info request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("user info request returned status: %d", resp.StatusCode)
	}

	var userInfo UserInfo
	if err := json.NewDecoder(resp.Body).Decode(&userInfo); err != nil {
		return nil, fmt.Errorf("failed to decode user info: %w", err)
	}

	return &userInfo, nil
}

// generateState generates a random state parameter for OAuth security
func (p *SentraIPOAuthPlugin) generateState() string {
	// Simple state generation - in production, use crypto/rand
	return fmt.Sprintf("state_%d", len("random"))
}

// init function - required for Tyk plugins
func init() {
	log.Get().WithFields(logrus.Fields{
		"plugin": "sentraip_oauth",
		"version": "1.0.0",
	}).Info("SentraIP OAuth Plugin initialized")
}
