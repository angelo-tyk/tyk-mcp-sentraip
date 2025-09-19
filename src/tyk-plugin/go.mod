module github.com/tyk-mcp-sentraip/tyk-mcp-sentraip/tyk-plugin

go 1.22

require (
    github.com/TykTechnologies/tyk latest
    github.com/sirupsen/logrus v1.9.3
    go.opentelemetry.io/otel v1.21.0
    go.opentelemetry.io/otel/trace v1.21.0
    golang.org/x/oauth2 v0.15.0
)

replace (
    github.com/hashicorp/terraform => github.com/hashicorp/terraform v0.12.31
)
