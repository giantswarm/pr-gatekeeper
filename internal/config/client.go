package config

import (
	"os"
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
	Repos         Repos         `json:"repos"`
}

type KnownTriggers map[string]string
type Repos map[string]Repo

type Repo struct {
	RequiredChecks []string `json:"requiredChecks"`
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
