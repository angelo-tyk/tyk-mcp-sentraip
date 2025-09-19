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
