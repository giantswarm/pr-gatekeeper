package config

import (
	"path/filepath"
	"slices"
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

func TestGetIgnoredPaths(t *testing.T) {
	configFile := filepath.Join("testdata", "config.yaml")

	testCases := []struct {
		name     string
		repo     string
		expected []string
	}{
		{
			name:     "repo without config uses the global paths",
			repo:     "unknown-repo",
			expected: []string{".github/**", "README.md"},
		},
		{
			name:     "repo config without paths uses the global paths",
			repo:     "some-repo",
			expected: []string{".github/**", "README.md"},
		},
		{
			name:     "repo paths override the global paths",
			repo:     "override-repo",
			expected: []string{"docs/**"},
		},
		{
			name:     "repo can opt out with an empty list",
			repo:     "opt-out-repo",
			expected: []string{},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("CONFIG_FILE", configFile)

			paths, err := GetIgnoredPaths(tc.repo)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !slices.Equal(paths, tc.expected) {
				t.Errorf("expected paths %v, got %v", tc.expected, paths)
			}
		})
	}
}

func TestOnlyIgnoredFiles(t *testing.T) {
	patterns := []string{".github/**", "README.md", "CHANGELOG.md", "docs/*.md"}

	testCases := []struct {
		name     string
		files    []string
		patterns []string
		expected bool
	}{
		{
			name:     "only ignored files",
			files:    []string{".github/workflows/ci.yaml", "README.md", "CHANGELOG.md"},
			patterns: patterns,
			expected: true,
		},
		{
			name:     "nested directory below a ** pattern",
			files:    []string{".github/a/b/c.yaml"},
			patterns: patterns,
			expected: true,
		},
		{
			name:     "pattern without a slash only matches at the root",
			files:    []string{"helm/chart/README.md"},
			patterns: patterns,
			expected: false,
		},
		{
			name:     "leading ./ in a pattern is ignored",
			files:    []string{"docs/a/b.md", "notes.txt"},
			patterns: []string{"./docs/**", "./notes.txt"},
			expected: true,
		},
		{
			name:     "pattern with a slash matches the full path",
			files:    []string{"docs/guide.md"},
			patterns: patterns,
			expected: true,
		},
		{
			name:     "pattern with a slash doesn't match deeper paths",
			files:    []string{"docs/nested/guide.md"},
			patterns: patterns,
			expected: false,
		},
		{
			name:     "mixed ignored and other files",
			files:    []string{"README.md", "main.go"},
			patterns: patterns,
			expected: false,
		},
		{
			name:     "directory prefix must end at a path separator",
			files:    []string{".github-extra/file"},
			patterns: patterns,
			expected: false,
		},
		{
			name:     "file named like the ignored directory",
			files:    []string{".github"},
			patterns: patterns,
			expected: false,
		},
		{
			name:     "no changed files",
			files:    []string{},
			patterns: patterns,
			expected: false,
		},
		{
			name:     "no patterns",
			files:    []string{"README.md"},
			patterns: []string{},
			expected: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if got := OnlyIgnoredFiles(tc.files, tc.patterns); got != tc.expected {
				t.Errorf("expected %v, got %v", tc.expected, got)
			}
		})
	}
}
