module github.com/giantswarm/pr-gatekeeper

go 1.27.1

require (
	github.com/giantswarm/apptest-framework/v5 v5.3.0
	github.com/google/go-github/v92 v92.0.0
	golang.org/x/oauth2 v0.37.0
	k8s.io/apimachinery v0.37.1
)

require (
	github.com/google/go-querystring v1.2.0 // indirect
	go.yaml.in/yaml/v2 v2.4.4 // indirect
	sigs.k8s.io/json v0.0.0-20260909141634-11ed52e25bc5 // indirect
	sigs.k8s.io/yaml v1.6.0 // indirect
)

replace go.opentelemetry.io/otel v1.43.0 => go.opentelemetry.io/otel v1.44.0
