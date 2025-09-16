package main

import (
    "log"
    "net/http"
    "os"

    "github.com/gorilla/mux"
    "github.com/your-org/claude-mcp-client/config"
    "github.com/your-org/claude-mcp-client/handlers"
)

func main() {
    cfg, err := config.Load()
    if err != nil {
        log.Fatalf("Failed to load config: %v", err)
    }

    r := mux.NewRouter()
    
    // Health check
    r.HandleFunc("/health", handlers.HealthHandler).Methods("GET")
    
    // Chat endpoint
    r.HandleFunc("/chat", handlers.ChatHandler(cfg)).Methods("POST")
    
    // MCP endpoints
    r.HandleFunc("/mcp/tools", handlers.MCPToolsHandler(cfg)).Methods("GET")
    r.HandleFunc("/mcp/execute", handlers.MCPExecuteHandler(cfg)).Methods("POST")

    port := os.Getenv("PORT")
    if port == "" {
        port = "8080"
    }

    log.Printf("Starting server on port %s", port)
    log.Fatal(http.ListenAndServe(":"+port, r))
}
