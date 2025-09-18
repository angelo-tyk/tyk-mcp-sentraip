module github.com/yourusername/tyk-mcp-sentraip/tyk-plugin

go 1.22

require (
    github.com/TykTechnologies/tyk v5.9.0+incompatible
    go.opentelemetry.io/otel v1.21.0
    go.opentelemetry.io/otel/trace v1.21.0
    github.com/sirupsen/logrus v1.9.3
)

replace github.com/jensneuse/graphql-go-tools => github.com/TykTechnologies/graphql-go-tools v1.6.2-0.20231101160458-b5c1e5f2f975

