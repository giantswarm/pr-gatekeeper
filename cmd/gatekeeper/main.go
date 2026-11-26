package main

import (
	"fmt"
	"os"
	"regexp"
	"slices"
	"strings"
	"time"

	apptest "github.com/giantswarm/apptest-framework/v5/pkg/config"
	"k8s.io/apimachinery/pkg/util/yaml"

	"github.com/giantswarm/pr-gatekeeper/internal/config"
	"github.com/giantswarm/pr-gatekeeper/internal/github"
	"github.com/giantswarm/pr-gatekeeper/internal/results"
)

const (
	skipLabel         = "skip/ci"
	doNotMergeHold    = "do-not-merge/hold"
	e2eTestConfigFile = "./tests/e2e/config.yaml"
	// e2eTestSuitesDir holds one directory per test suite, each of which may
	// carry its own apptest config declaring extra providers.
	e2eTestSuitesDir   = "tests/e2e/suites"
	appTestCheckPrefix = "App E2E Test Suites"
	// defaultAppTestProvider matches the `DEFAULT_PROVIDER` default of the
	// app-test-suites pipeline, used for any apptest config that declares no
	// providers of its own.
	defaultAppTestProvider = "capa"
)

var (
	repo string
	pr   string
)

func main() {
	repo = os.Getenv("REPO")
	pr = os.Getenv("PR")

	if repo == "" || pr == "" {
		fmt.Println("Both `REPO` and `PR` environment variables must be set")
		os.Exit(1)
	}

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

	ignoredPaths, err := config.GetIgnoredPaths(repo)
	if err != nil {
		fmt.Println("Failed to load ignored paths")
		panic(err)
	}

	var changedFiles []string
	if len(ignoredPaths) > 0 {
		changedFiles, err = gh.GetChangedFiles()
		if err != nil {
			fmt.Println("Failed to list changed files")
		}
	}
	onlyIgnored := config.OnlyIgnoredFiles(changedFiles, ignoredPaths)
	isRelease := config.IsReleaseBranch(pullRequest.GetHead().GetRef())

	result.AddMessage(fmt.Sprintf("## Details for commit: `%s`\n", *pullRequest.Head.SHA))

	if onlyIgnored && isRelease {
		result.AddMessage("ℹ️ Only files matching the ignored paths were changed, but this is a release PR - **running required checks**\n")
	}

	if onlyIgnored && !isRelease {
		result.AddMessage(fmt.Sprintf("ℹ️ Only files matching the ignored paths were changed (`%s`) - **skipping required checks**\n",
			strings.Join(ignoredPaths, "`, `"),
		))
	} else {
		checkRequirements(&gh, result, repoConfig)
	}

	// Check labels on the PR for overriding behaviour
	hasDeprecatedSkipLabel := false
	for _, label := range pullRequest.Labels {
		switch strings.ToLower(label.Name) {
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
		latestCommitTime, err := gh.GetLastCommitTimestamp()
		if err != nil {
			fmt.Println("Failed to get last commit timestamp, unable to perform check for skip trigger comments")
		}
		comments, err := gh.GetPRComments()
		if err != nil {
			fmt.Println("Failed to get PR Comments, unable to perform check for skip trigger comments")
		}
		// A `/skip-ci` written into the "Review changes" box is a review, not a comment
		reviews, err := gh.GetPRReviews()
		if err != nil {
			fmt.Println("Failed to get PR Reviews, unable to perform check for skip triggers in review bodies")
		}

		oldSkipFound := false
		skipWithoutReasonFound := false
		skipReason := ""
		skippedBy := ""

		if latestCommitTime != nil && (comments != nil || reviews != nil) {
			candidates := []skipCandidate{}
			for _, comment := range comments {
				candidates = append(candidates, skipCandidate{
					Body:              comment.GetBody(),
					AuthorAssociation: comment.GetAuthorAssociation(),
					Author:            comment.GetUser().GetLogin(),
					CreatedAt:         comment.GetCreatedAt().Time,
				})
			}
			for _, review := range reviews {
				candidates = append(candidates, skipCandidate{
					Body:              review.GetBody(),
					AuthorAssociation: review.GetAuthorAssociation(),
					Author:            review.GetUser().GetLogin(),
					CreatedAt:         review.GetSubmittedAt().Time,
				})
			}

			skip := findSkipTrigger(candidates, latestCommitTime.Time)
			result.SkipCI = skip.Skip
			skipReason = skip.Reason
			skippedBy = skip.SkippedBy
			oldSkipFound = skip.OldSkipFound
			skipWithoutReasonFound = skip.WithoutReasonFound
			if oldSkipFound {
				fmt.Println("A `/skip-ci` trigger comment was found but it was made before the last commit, ignoring")
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

// checkRequirements verifies the repo's required checks, including those added
// from its apptest configs.
func checkRequirements(gh *github.Client, result *results.Result, repoConfig *config.Repo) {
	// Check if config file is present in the github repo. If present automatically add the E2E Test Suites check
	appTestProviders := []string{}
	configFile, ok, err := gh.GetFile(e2eTestConfigFile)
	if err != nil {
		fmt.Println("Failed to check repo for config file")
		panic(err)
	}
	if ok {
		if repoConfig == nil {
			repoConfig = &config.Repo{RequiredChecks: []string{}}
		}

		appTestConfigs, err := getAppTestConfigs(gh, configFile)
		if err != nil {
			fmt.Println("Failed to load app test config files")
			panic(err)
		}

		appTestProviders, err = getAppTestProviders(appTestConfigs)
		if err != nil {
			fmt.Println("Failed to parse app test config file")
			panic(err)
		}

		for _, provider := range appTestProviders {
			checkName := fmt.Sprintf("%s - %s", appTestCheckPrefix, provider)
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
		if len(appTestProviders) > 0 {
			result.AddMessage(fmt.Sprintf("ℹ️ App E2E tests are required for every provider configured in `%s` and in the per-suite configs under `%s`: `%s`\n",
				e2eTestConfigFile,
				e2eTestSuitesDir,
				strings.Join(appTestProviders, "`, `"),
			))
		}

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
}

// getAppTestConfigs returns the contents of every apptest config that applies to
// the repo: the top level config (already fetched, passed in as rootConfig)
// plus any per-suite config under e2eTestSuitesDir.
func getAppTestConfigs(gh *github.Client, rootConfig string) ([]string, error) {
	configs := []string{rootConfig}

	suiteConfigFiles, err := gh.FindFiles(e2eTestSuitesDir, "config.yaml")
	if err != nil {
		return nil, err
	}

	for _, suiteConfigFile := range suiteConfigFiles {
		contents, ok, err := gh.GetFile(suiteConfigFile)
		if err != nil {
			return nil, err
		}
		if ok {
			configs = append(configs, contents)
		}
	}

	return configs, nil
}

// getAppTestProviders returns the full set of providers a `/run app-test-suites`
// covers for the repo: the union of the providers declared across all its
// apptest configs, with configs declaring none falling back to the default
// provider. This mirrors the `gather-facts` task of the app-test-suites pipeline
// (see giantswarm/tekton-resources), so that every provider a full run would
// produce a check for is required - not just the ones the top level config
// happens to name, or the ones a targeted single-provider run covered.
func getAppTestProviders(configs []string) ([]string, error) {
	providers := []string{}

	for _, contents := range configs {
		apptestConfig := &apptest.TestConfig{}
		err := yaml.Unmarshal([]byte(contents), apptestConfig)
		if err != nil {
			return nil, err
		}

		declared := apptestConfig.Providers
		if len(declared) == 0 {
			declared = []string{defaultAppTestProvider}
		}

		for _, provider := range declared {
			provider = strings.ToLower(strings.TrimSpace(provider))
			if provider != "" && !slices.Contains(providers, provider) {
				providers = append(providers, provider)
			}
		}
	}
	slices.Sort(providers)

	return providers, nil
}

var (
	skipTriggerComment               = regexp.MustCompile(`(?mi)^\/skip-ci (?P<reason>.+)(?:\r|\n|$)`)
	skipTriggerCommentWithoutMessage = regexp.MustCompile(`(?mi)^\/skip-ci\s?(?:\r|\n|$)`)
)

// skipCandidate is a PR comment or PR review body that may carry a `/skip-ci` trigger
type skipCandidate struct {
	Body              string
	AuthorAssociation string
	Author            string
	CreatedAt         time.Time
}

type skipTrigger struct {
	Skip               bool
	Reason             string
	SkippedBy          string
	OldSkipFound       bool
	WithoutReasonFound bool
}

// findSkipTrigger looks for a `/skip-ci [reason]` trigger made by an org member after the latest commit
func findSkipTrigger(candidates []skipCandidate, latestCommitTime time.Time) skipTrigger {
	found := skipTrigger{}
	for _, candidate := range candidates {
		if candidate.Body == "" || (candidate.AuthorAssociation != "OWNER" && candidate.AuthorAssociation != "MEMBER") {
			continue
		}
		triggerMatches := skipTriggerComment.FindAllStringSubmatch(candidate.Body, -1)
		if len(triggerMatches) > 0 {
			if candidate.CreatedAt.After(latestCommitTime) {
				found.Skip = true
				found.Reason = strings.TrimSpace(triggerMatches[0][1])
				found.SkippedBy = candidate.Author
			} else {
				found.OldSkipFound = true
			}
		} else if skipTriggerCommentWithoutMessage.MatchString(candidate.Body) {
			found.WithoutReasonFound = true
		}
	}
	return found
}
