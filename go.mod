module github.com/giantswarm/pr-gatekeeper

go 1.22.4

toolchain go1.23.1

require (
	github.com/giantswarm/apptest-framework v1.8.0
	github.com/google/go-github/v64 v64.0.0
	golang.org/x/oauth2 v0.23.0
	k8s.io/apimachinery v0.31.1
)

require (
	github.com/google/go-querystring v1.1.0 // indirect
	sigs.k8s.io/json v0.0.0-20221116044647-bc3834ca7abd // indirect
	sigs.k8s.io/yaml v1.4.0 // indirect
)
