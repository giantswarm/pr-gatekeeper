module github.com/giantswarm/pr-gatekeeper

go 1.26.3

require (
	github.com/giantswarm/apptest-framework/v5 v5.2.1
	github.com/google/go-github/v88 v88.0.0
	golang.org/x/oauth2 v0.36.0
	k8s.io/apimachinery v0.36.2
)

require (
	github.com/google/go-querystring v1.2.0 // indirect
	go.yaml.in/yaml/v2 v2.4.4 // indirect
	sigs.k8s.io/json v0.0.0-20250730193827-2d320260d730 // indirect
	sigs.k8s.io/yaml v1.6.0 // indirect
)

replace go.opentelemetry.io/otel v1.43.0 => go.opentelemetry.io/otel v1.44.0
