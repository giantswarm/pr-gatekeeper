package main

import (
	"slices"
	"testing"
	"time"
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

func TestFindSkipTrigger(t *testing.T) {
	lastCommit := time.Date(2026, 11, 25, 22, 58, 0, 0, time.UTC)
	before := lastCommit.Add(-time.Hour)
	after := lastCommit.Add(time.Hour)

	tests := []struct {
		name       string
		candidates []skipCandidate
		want       skipTrigger
	}{
		{
			name:       "no candidates",
			candidates: nil,
			want:       skipTrigger{},
		},
		{
			name:       "member comment with reason",
			candidates: []skipCandidate{{Body: "/skip-ci patch bump", AuthorAssociation: "MEMBER", Author: "alice", CreatedAt: after}},
			want:       skipTrigger{Skip: true, Reason: "patch bump", SkippedBy: "alice"},
		},
		{
			name:       "trigger on a later line of a review body",
			candidates: []skipCandidate{{Body: "LGTM\r\n/skip-ci patch bump\r\n", AuthorAssociation: "OWNER", Author: "bob", CreatedAt: after}},
			want:       skipTrigger{Skip: true, Reason: "patch bump", SkippedBy: "bob"},
		},
		{
			name:       "non-member is ignored",
			candidates: []skipCandidate{{Body: "/skip-ci patch bump", AuthorAssociation: "CONTRIBUTOR", Author: "eve", CreatedAt: after}},
			want:       skipTrigger{},
		},
		{
			name:       "made before the last commit",
			candidates: []skipCandidate{{Body: "/skip-ci patch bump", AuthorAssociation: "MEMBER", Author: "alice", CreatedAt: before}},
			want:       skipTrigger{OldSkipFound: true},
		},
		{
			name:       "missing reason",
			candidates: []skipCandidate{{Body: "/skip-ci", AuthorAssociation: "MEMBER", Author: "alice", CreatedAt: after}},
			want:       skipTrigger{WithoutReasonFound: true},
		},
		{
			name:       "empty approval body",
			candidates: []skipCandidate{{Body: "", AuthorAssociation: "MEMBER", Author: "alice", CreatedAt: after}},
			want:       skipTrigger{},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := findSkipTrigger(tc.candidates, lastCommit)
			if got != tc.want {
				t.Errorf("findSkipTrigger() = %+v, want %+v", got, tc.want)
			}
		})
	}
}
