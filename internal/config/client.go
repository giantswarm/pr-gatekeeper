package config

import (
	"os"
	"path"
	"strings"

	"k8s.io/apimachinery/pkg/util/yaml"
)

const (
	// checkNameSeparator separates the static part of a check name from its
	// dynamic suffix, e.g. "App E2E Test Suites - capa".
	checkNameSeparator = " - "
	// suffixPlaceholder is replaced in a known trigger with the dynamic suffix
	// of the check name it was matched against by prefix.
	suffixPlaceholder = "{{suffix}}"
)

type Conf struct {
	KnownTriggers KnownTriggers `json:"knownTriggers"`
	IgnoredPaths  []string      `json:"ignoredPaths"`
	Repos         Repos         `json:"repos"`
}

type KnownTriggers map[string]string
type Repos map[string]Repo

type Repo struct {
	RequiredChecks []string `json:"requiredChecks"`
	// IgnoredPaths overrides the global IgnoredPaths for this repo when set.
	IgnoredPaths []string `json:"ignoredPaths"`
}

func LoadConfig() (*Conf, error) {
	configFile := os.Getenv("CONFIG_FILE")
	if configFile == "" {
		configFile = "config.yaml"
	}
	file, err := os.ReadFile(configFile) // nolint:gosec
	if err != nil {
		return nil, err
	}

	var conf Conf
	err = yaml.Unmarshal(file, &conf)
	if err != nil {
		return nil, err
	}

	return &conf, nil
}

func GetRepoConfig(repo string) (*Repo, error) {
	conf, err := LoadConfig()
	if err != nil {
		return nil, err
	}

	config, ok := conf.Repos[repo]
	if ok {
		return &config, nil
	}

	return nil, nil
}

// GetKnownTrigger returns the PR comment trigger for the given check run name,
// or an empty string if none is configured.
//
// Check names are matched exactly first. Checks with a dynamic suffix (e.g. the
// per-provider "App E2E Test Suites - capa" checks added from a repo's apptest
// config) are then matched against the longest configured name that is a prefix
// of them, with the suffix substituted into any `{{suffix}}` placeholder in the
// trigger.
func GetKnownTrigger(check string) string {
	conf, err := LoadConfig()
	if err != nil {
		return ""
	}

	trigger, ok := conf.KnownTriggers[check]
	if ok {
		return trigger
	}

	prefix := ""
	for name := range conf.KnownTriggers {
		if strings.HasPrefix(check, name+checkNameSeparator) && len(name) > len(prefix) {
			prefix = name
		}
	}
	if prefix == "" {
		return ""
	}

	suffix := strings.TrimPrefix(check, prefix+checkNameSeparator)

	return strings.ReplaceAll(conf.KnownTriggers[prefix], suffixPlaceholder, suffix)
}

// GetIgnoredPaths returns the path patterns that don't require any checks for
// the given repo: the repo's own IgnoredPaths if set, otherwise the global ones.
func GetIgnoredPaths(repo string) ([]string, error) {
	conf, err := LoadConfig()
	if err != nil {
		return nil, err
	}

	if repoConf, ok := conf.Repos[repo]; ok && repoConf.IgnoredPaths != nil {
		return repoConf.IgnoredPaths, nil
	}

	return conf.IgnoredPaths, nil
}

// OnlyIgnoredFiles returns true if there is at least one file and every file
// matches at least one of the patterns.
//
// Patterns are relative to the repo root, a leading `./` is ignored. Patterns
// ending in `/**` match everything below that directory, any other pattern is
// matched against the full path using path.Match (so `README.md` only matches
// the root README and `*` doesn't cross directories).
func OnlyIgnoredFiles(files, patterns []string) bool {
	if len(files) == 0 || len(patterns) == 0 {
		return false
	}

	for _, file := range files {
		if !matchesAny(file, patterns) {
			return false
		}
	}

	return true
}

func matchesAny(file string, patterns []string) bool {
	for _, pattern := range patterns {
		if matchPath(pattern, file) {
			return true
		}
	}
	return false
}

func matchPath(pattern, file string) bool {
	pattern = strings.TrimPrefix(pattern, "./")

	if dir, ok := strings.CutSuffix(pattern, "/**"); ok {
		return strings.HasPrefix(file, dir+"/")
	}

	matched, err := path.Match(pattern, file)
	return err == nil && matched
}
