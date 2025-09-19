module github.com/yourusername/tyk-mcp-sentraip/tyk-plugin

go 1.22

require (
        github.com/sirupsen/logrus v1.9.3
        go.opentelemetry.io/otel v1.21.0
        go.opentelemetry.io/otel/trace v1.21.0
        golang.org/x/oauth2 v0.15.0
)

require golang.org/x/sys v0.0.0-20220715151400-c0bba94af5f8 // indirect
replace github.com/hashicorp/terraform => github.com/hashicorp/terraform v0.11.15
