package config

import (
	"os"

	"k8s.io/apimachinery/pkg/util/yaml"
)

type Repos map[string]Repo

type Repo struct {
	RequiredChecks []string `json:"requiredChecks"`
}

func LoadConfig() (Repos, error) {
	file, err := os.ReadFile("repos.yaml")
	if err != nil {
		return nil, err
	}

	var repoConfig Repos
	err = yaml.Unmarshal(file, &repoConfig)
	if err != nil {
		return nil, err
	}

	return repoConfig, nil
}

func GetRepoConfig(repo string) (*Repo, error) {
	repoConfig, err := LoadConfig()
	if err != nil {
		return nil, err
	}

	config, ok := repoConfig[repo]
	if ok {
		return &config, nil
	}

	return nil, nil

	// // TODO: Replace with actual code...
	// return Repo{
	// 	RequiredChecks: []string{"E2E Tests"},
	// }
}
