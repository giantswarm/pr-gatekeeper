package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/giantswarm/pr-gatekeeper/internal/config"
	"github.com/giantswarm/pr-gatekeeper/internal/github"
)

const (
	skipLabel = "skip/ci"
)

var (
	repo string
	pr   string
)

func init() {
	repo = os.Getenv("REPO")
	pr = os.Getenv("PR")

	if repo == "" || pr == "" {
		fmt.Println("Both `REPO` and `PR` environment variables must be set")
		os.Exit(1)
	}
}

func main() {
	gh := github.New(repo, pr)
	pullRequest, err := gh.GetPR()
	if err != nil {
		fmt.Println("Failed to fetch Pull Request")
		panic(err)
	}

	err = gh.AddCheck(true, "")
	if err != nil {
		fmt.Println("Failed to add check run")
		panic(err)
	}

	accessAllowed := true
	messages := []string{}

	// Perform checks and stuff
	repoConfig, err := config.GetRepoConfig(repo)
	if err != nil {
		fmt.Println("Failed to load repo config")
		panic(err)
	}

	if repoConfig == nil {
		fmt.Println("No repo config found, skipping checks")
		messages = append(messages, "No repo config found, skipping checks")
	} else {
		for _, check := range repoConfig.RequiredChecks {
			checkRun, err := gh.GetCheck(check)
			if err != nil || checkRun == nil {
				accessAllowed = false
				messages = append(messages, fmt.Sprintf("⚠️ Check Run `%s` is required and but wasn't found\n", check))
			} else if *checkRun.Conclusion != "success" {
				accessAllowed = false
				messages = append(messages, fmt.Sprintf("⚠️ Check Run `%s` is required and not currently completed successfully\n", check))
			} else if *checkRun.Conclusion == "success" {
				messages = append(messages, fmt.Sprintf("✅ Check Run `%s` is required and completed successfully\n", check))
			}
		}

		// If the PR contains a skip CI label we'll allow access
		for _, label := range pullRequest.Labels {
			if strings.ToLower(*label.Name) == skipLabel {
				accessAllowed = true
				messages = append(messages, fmt.Sprintf("ℹ️ Pull Requests contains the '%s' label - ignoring other requirements", skipLabel))
				break
			}
		}
	}

	err = gh.AddCheck(!accessAllowed, strings.Join(messages, "\n"))
	if err != nil {
		fmt.Println("Failed to add check run")
		panic(err)
	}
}
