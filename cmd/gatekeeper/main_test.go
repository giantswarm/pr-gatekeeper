package main

import (
	"slices"
	"testing"
)

func TestGetAppTestProviders(t *testing.T) {
	testCases := []struct {
		name     string
		configs  []string
		expected []string
	}{
		{
			name:     "providers from the top level config",
			configs:  []string{"appName: hello-world\nproviders:\n- capa\n- capz\n"},
			expected: []string{"capa", "capz"},
		},
		{
			name:     "a config without providers falls back to the default provider",
			configs:  []string{"appName: hello-world\n"},
			expected: []string{"capa"},
		},
		{
			name: "providers only declared in a suite config are included",
			configs: []string{
				"appName: hello-world\nproviders:\n- capa\n",
				"appName: hello-world\nproviders:\n- capz\n",
			},
			expected: []string{"capa", "capz"},
		},
		{
			name: "providers are lowercased and de-duplicated",
			configs: []string{
				"appName: hello-world\nproviders:\n- capa\n- CAPZ\n",
				"appName: hello-world\nproviders:\n- capz\n- vsphere\n",
			},
			expected: []string{"capa", "capz", "vsphere"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			providers, err := getAppTestProviders(tc.configs)
			if err != nil {
				t.Fatalf("unexpected error: %s", err)
			}
			if !slices.Equal(providers, tc.expected) {
				t.Errorf("expected providers %v, got %v", tc.expected, providers)
			}
		})
	}
}

func TestGetAppTestProvidersInvalidConfig(t *testing.T) {
	_, err := getAppTestProviders([]string{"providers: [}"})
	if err == nil {
		t.Error("expected an error for an unparsable config, got none")
	}
}
