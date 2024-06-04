package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/giantswarm/pr-gatekeeper/internal/config"
	"github.com/giantswarm/pr-gatekeeper/internal/github"
	"github.com/giantswarm/pr-gatekeeper/internal/results"
)

const (
	skipLabel                  = "skip/ci"
	doNotMergeHold     = "do-not-merge/hold"
	e2eTestConfigFile    = "./tests/e2e/config.yaml"
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

	result := results.New()

	// Perform checks and stuff
	repoConfig, err := config.GetRepoConfig(repo)
	if err != nil {
		fmt.Println("Failed to load repo config")
		panic(err)
	}

	// Check if config file is present in the github repo. If present automatically add the E2E Test Suites check
	configFileInRepo := gh.FilePresentInRepo(e2eTestConfigFile)
	if configFileInRepo {
		fmt.Println("'E2E Test Suites' check automatically added to the required checks")
		if repoConfig == nil {
			repoConfig = &config.Repo{RequiredChecks: []string{"E2E Test Suites"}}
		}else{
			repoConfig.RequiredChecks = append(repoConfig.RequiredChecks, "E2E Test Suites")
		}
	}

	if repoConfig == nil {
		fmt.Println("No repo config found, skipping checks")
		result.AddMessage("No repo config found, skipping checks")
	} else {
		result.AddMessage(fmt.Sprintf("## Details for commit: `%s`\n", *pullRequest.Head.SHA))

		for _, check := range repoConfig.RequiredChecks {
			checkRun, err := gh.GetCheck(check)
			switch {
			case err != nil || checkRun == nil:
				result.ChecksPassing = false
				trigger := config.GetKnownTrigger(check)
				if trigger != "" {
					trigger = fmt.Sprintf(" - you can trigger it by commenting on the PR with `%s`", trigger)
				}
				result.AddMessage(fmt.Sprintf("⚠️ Check Run `%s` is required but wasn't found%s\n", check, trigger))

			case checkRun.Conclusion == nil:
				result.ChecksPassing = false
				result.AddMessage(fmt.Sprintf("⚠️ Check Run `%s` is required but is still in progress\n", check))

			case *checkRun.Conclusion == "success":
				result.AddMessage(fmt.Sprintf("✅ Check Run `%s` is required and has completed successfully\n", check))

			default:
				result.ChecksPassing = false
				trigger := config.GetKnownTrigger(check)
				if trigger != "" {
					trigger = fmt.Sprintf(" - you can re-trigger it by commenting on the PR with `%s`", trigger)
				}
				result.AddMessage(fmt.Sprintf("⚠️ Check Run `%s` is required but didn't completed successfully%s\n", check, trigger))
			}
		}
	}

	// Check labels on the PR for overriding behaviour
	for _, label := range pullRequest.Labels {
		switch strings.ToLower(*label.Name) {
		case skipLabel:
			result.SkipCI = true
			result.AddMessage(fmt.Sprintf("ℹ️ Pull Requests contains the `%s` label - **ignoring other requirements**", skipLabel))
		case doNotMergeHold:
			result.HoldPR = true
			result.AddMessage(fmt.Sprintf("⛔️ Pull Requests contains the `%s` label - **blocking merge until removed**", doNotMergeHold))
		default:
			continue
		}
	}

	err = gh.AddCheck(!result.AllowAccess(), result.GetMessages())
	if err != nil {
		fmt.Println("Failed to add check run")
		panic(err)
	}
}
