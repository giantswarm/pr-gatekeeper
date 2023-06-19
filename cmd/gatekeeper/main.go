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

		messages = append(messages, fmt.Sprintf("## Details for commit: `%s`\n", *pullRequest.Head.SHA))

		for _, check := range repoConfig.RequiredChecks {
			checkRun, err := gh.GetCheck(check)
			if err != nil || checkRun == nil {
				accessAllowed = false
				trigger := config.GetKnownTrigger(check)
				if trigger != "" {
					trigger = fmt.Sprintf(" - you can trigger it by commenting on the PR with `%s`", trigger)
				}
				messages = append(messages, fmt.Sprintf("⚠️ Check Run `%s` is required but wasn't found%s\n", check, trigger))
			} else if checkRun.Conclusion == nil {
				accessAllowed = false
				messages = append(messages, fmt.Sprintf("⚠️ Check Run `%s` is required but is still in progress\n", check))
			} else if *checkRun.Conclusion != "success" {
				accessAllowed = false
				trigger := config.GetKnownTrigger(check)
				if trigger != "" {
					trigger = fmt.Sprintf(" - you can re-trigger it by commenting on the PR with `%s`", trigger)
				}
				messages = append(messages, fmt.Sprintf("⚠️ Check Run `%s` is required but didn't completed successfully%s\n", check, trigger))
			} else if *checkRun.Conclusion == "success" {
				messages = append(messages, fmt.Sprintf("✅ Check Run `%s` is required and has completed successfully\n", check))
			}
		}

		// If the PR contains a skip CI label we'll allow access
		for _, label := range pullRequest.Labels {
			if strings.ToLower(*label.Name) == skipLabel {
				accessAllowed = true
				messages = append(messages, fmt.Sprintf("ℹ️ Pull Requests contains the `%s` label - **ignoring other requirements**", skipLabel))
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
