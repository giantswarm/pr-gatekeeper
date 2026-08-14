package config

import (
	"path/filepath"
	"testing"
)

func TestGetKnownTrigger(t *testing.T) {
	configFile := filepath.Join("testdata", "config.yaml")

	testCases := []struct {
		name     string
		check    string
		expected string
	}{
		{
			name:     "exact match",
			check:    "E2E Test Suites",
			expected: "/run cluster-test-suites",
		},
		{
			name:     "prefix match substitutes the suffix",
			check:    "App E2E Test Suites - capa",
			expected: "/run app-test-suites-single PROVIDER=capa",
		},
		{
			name:     "longest prefix wins",
			check:    "Nested - Check - capz",
			expected: "/run long SUFFIX=capz",
		},
		{
			name:     "unknown check",
			check:    "Something Else",
			expected: "",
		},
		{
			name:     "prefix without the separator doesn't match",
			check:    "E2E Test Suites Extra",
			expected: "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("CONFIG_FILE", configFile)

			trigger := GetKnownTrigger(tc.check)
			if trigger != tc.expected {
				t.Errorf("expected trigger %q, got %q", tc.expected, trigger)
			}
		})
	}
}
