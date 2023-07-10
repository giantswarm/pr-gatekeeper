package config

import (
	"os"

	"k8s.io/apimachinery/pkg/util/yaml"
)

type Conf struct {
	KnownTriggers KnownTriggers `json:"knownTriggers"`
	Repos         Repos         `json:"repos"`
}

type KnownTriggers map[string]string
type Repos map[string]Repo

type Repo struct {
	RequiredChecks []string `json:"requiredChecks"`
	SkipDocsOnly   bool     `json:"skipDocsOnly"`
}

func LoadConfig() (*Conf, error) {
	configFile := os.Getenv("CONFIG_FILE")
	if configFile == "" {
		configFile = "config.yaml"
	}
	file, err := os.ReadFile(configFile)
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

func GetKnownTrigger(check string) string {
	conf, err := LoadConfig()
	if err != nil {
		return ""
	}

	trigger, ok := conf.KnownTriggers[check]
	if ok {
		return trigger
	}

	return ""
}
