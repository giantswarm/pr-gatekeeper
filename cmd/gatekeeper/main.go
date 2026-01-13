package main

import (
	"fmt"
	"os"
	"regexp"
	"slices"
	"strings"

	apptest "github.com/giantswarm/apptest-framework/v3/pkg/config"
	"k8s.io/apimachinery/pkg/util/yaml"

	"github.com/giantswarm/pr-gatekeeper/internal/config"
	"github.com/giantswarm/pr-gatekeeper/internal/github"
	"github.com/giantswarm/pr-gatekeeper/internal/results"
)

const (
	skipLabel          = "skip/ci"
	doNotMergeHold     = "do-not-merge/hold"
	e2eTestConfigFile  = "./tests/e2e/config.yaml"
	appTestCheckPrefix = "App E2E Test Suites"
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
	configFile, ok, err := gh.GetFile(e2eTestConfigFile)
	if err != nil {
		fmt.Println("Failed to check repo for config file")
		panic(err)
	}
	if ok {
		if repoConfig == nil {
			repoConfig = &config.Repo{RequiredChecks: []string{}}
		}

		apptestConfig := &apptest.TestConfig{}
		err = yaml.Unmarshal([]byte(configFile), apptestConfig)
		if err != nil {
			fmt.Println("Failed to parse app test config file")
			panic(err)
		}

		for _, provider := range apptestConfig.Providers {
			checkName := fmt.Sprintf("%s - %s", appTestCheckPrefix, strings.ToLower(provider))
			if !slices.Contains(repoConfig.RequiredChecks, checkName) {
				fmt.Printf("Adding the '%s' required check\n", checkName)
				repoConfig.RequiredChecks = append(repoConfig.RequiredChecks, checkName)
			}
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
	hasDeprecatedSkipLabel := false
	for _, label := range pullRequest.Labels {
		switch strings.ToLower(*label.Name) {
		case skipLabel:
			fmt.Println("The use of the skip/ci label is deprecated, please use the /skip-ci comment trigger instead")
			result.AddMessage(fmt.Sprintf("ℹ️ Please note: The `%s` label no longer controls skipping the CI checks and is now purely informational. Please use the `/skip-ci [reason]` comment trigger with a required reason message.", skipLabel))
			hasDeprecatedSkipLabel = true
		case doNotMergeHold:
			result.HoldPR = true
			result.AddMessage(fmt.Sprintf("⛔️ Pull Requests contains the `%s` label - **blocking merge until removed**", doNotMergeHold))
		default:
			continue
		}
	}

	// Handle skipping CI via `/skip-ci [MESSAGE]` comment trigger
	{
		skipTriggerComment := regexp.MustCompile(`(?mi)^\/skip-ci (?P<reason>.+)(?:\r|\n|$)`)
		skipTriggerCommentWithoutMessage := regexp.MustCompile(`(?mi)^\/skip-ci\s?(?:\r|\n|$)`)

		latestCommitTime, err := gh.GetLastCommitTimestamp()
		if err != nil {
			fmt.Println("Failed to get last commit timestamp, unable to perform check for skip trigger comments")
		}
		comments, err := gh.GetPRComments()
		if err != nil {
			fmt.Println("Failed to get PR Comments, unable to perform check for skip trigger comments")
		}

		oldSkipFound := false
		skipWithoutReasonFound := false
		skipReason := ""
		skippedBy := ""

		if latestCommitTime != nil && comments != nil {
			for _, comment := range comments {
				if comment.AuthorAssociation != nil && (*comment.AuthorAssociation == "OWNER" || *comment.AuthorAssociation == "MEMBER") && comment.Body != nil && *comment.Body != "" {
					triggerMatches := skipTriggerComment.FindAllStringSubmatch(*comment.Body, -1)
					triggerMatchesWihthoutMessage := skipTriggerCommentWithoutMessage.FindAllStringSubmatch(*comment.Body, -1)
					if len(triggerMatches) > 0 {
						if comment.CreatedAt.After(latestCommitTime.Time) {
							result.SkipCI = true
							skipReason = triggerMatches[0][1]
							skippedBy = comment.User.GetLogin()
						} else {
							oldSkipFound = true
							fmt.Println("A `/skip-ci` trigger comment was found but it was made before the last commit, ignoring")
						}
					} else if len(triggerMatchesWihthoutMessage) > 0 {
						skipWithoutReasonFound = true
					}
				}
			}
		}

		if result.SkipCI {
			result.AddMessage(
				fmt.Sprintf("⏭️ Pull Request contains a `/skip-ci` trigger comment - **ignoring other requirements**\n\nSkip reason: `%s`\nSkipped by: @%s",
					skipReason,
					skippedBy,
				),
			)

			_ = gh.AddSkipCILabel()

			// Add a comment to the PR to inform everyone that the CI checks were skipped.
			// We are ok with this failing as the info will be on the GitHub Check too
			_ = gh.AddSkippingComment(skipReason, skippedBy)
		} else {
			_ = gh.RemoveSkipCILabel()

			// Inform the user if a `/skip-ci` trigger comment was found but it was made before the latest commit
			if oldSkipFound {
				result.AddMessage("⚠️ A `/skip-ci` trigger comment was found but it was made before the latest commit, ignoring")
			}

			// Inform the user if a `/skip-ci` trigger comment was found but without a reason given
			if skipWithoutReasonFound {
				result.AddMessage("⚠️ A reason must be provided when using the `/skip-ci` trigger comment, ignoring")
				_ = gh.AddReasonRequiredComment()
			}

			//Inform user that only using the `skip/ci` label is no longer supported
			if !oldSkipFound && !skipWithoutReasonFound && hasDeprecatedSkipLabel {
				_ = gh.AddSkipLabelDeprecatedComment()
			}
		}
	}

	err = gh.AddCheck(!result.AllowAccess(), result.GetMessages())
	if err != nil {
		fmt.Println("Failed to add check run")
		panic(err)
	}
}
