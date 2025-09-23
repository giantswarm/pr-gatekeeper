package github

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/google/go-github/v75/github"
	"golang.org/x/oauth2"
)

const (
	owner = "giantswarm"
)

var (
	checkName = "Heimdall - PR Gatekeeper"
	imageHTML = `<img src="https://github.com/giantswarm/pr-gatekeeper/assets/3384072/6c85d6be-6726-446c-ab37-c6c26601aec9" height="100px" alt="Heimdall" />`
)

type Client struct {
	*github.Client

	Ctx  context.Context
	Repo string
	PR   string
	Sha  string
}

func New(repo, pr string) Client {
	ctx := context.Background()
	oClient := oauth2.NewClient(ctx, oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: os.Getenv("GITHUB_TOKEN")},
	))
	return Client{
		github.NewClient(oClient),

		ctx,
		repo,
		pr,
		"",
	}
}

func (c *Client) GetPR() (*github.PullRequest, error) {
	prNumber, err := strconv.Atoi(c.PR)
	if err != nil {
		return nil, err
	}
	pullRequest, _, err := c.PullRequests.Get(c.Ctx, owner, c.Repo, prNumber)
	c.Sha = *pullRequest.Head.SHA
	return pullRequest, err
}

func (c *Client) GetPRComments() ([]*github.IssueComment, error) {
	prNumber, err := strconv.Atoi(c.PR)
	if err != nil {
		return nil, err
	}
	comments, _, err := c.Issues.ListComments(c.Ctx, owner, c.Repo, prNumber, &github.IssueListCommentsOptions{ListOptions: github.ListOptions{PerPage: 500}})
	return comments, err
}

func (c *Client) GetLastCommitTimestamp() (*github.Timestamp, error) {
	prNumber, err := strconv.Atoi(c.PR)
	if err != nil {
		return nil, err
	}
	commits, _, err := c.PullRequests.ListCommits(c.Ctx, owner, c.Repo, prNumber, &github.ListOptions{PerPage: 500})
	if err != nil {
		return nil, err
	}
	latestCommit := commits[len(commits)-1]
	return latestCommit.Commit.Committer.Date, nil
}

func (c *Client) AddCheck(pending bool, msg string) error {
	status := "in_progress"
	summary := "🚧 PR currently blocked from merging\n\n" + imageHTML
	if !pending {
		status = "completed"
		summary = "✅ PR meets all defined requirements for merging\n\n" + imageHTML
	}

	msg += "\n\n---\n_Source: https://github.com/giantswarm/pr-gatekeeper_\n_Repo Config: https://github.com/giantswarm/pr-gatekeeper/blob/main/config.yaml_"

	_, _, err := c.Checks.CreateCheckRun(c.Ctx, owner, c.Repo, github.CreateCheckRunOptions{
		Name:       checkName,
		HeadSHA:    c.Sha,
		Status:     &status,
		Conclusion: getConclusion(pending),
		Output: &github.CheckRunOutput{
			Title:   &checkName,
			Summary: &summary,
			Text:    &msg,
			Images:  []*github.CheckRunImage{},
		},
	})
	return err
}

func getConclusion(pending bool) *string {
	if pending {
		return nil
	}
	conclusion := "success"
	return &conclusion
}

func (c *Client) GetCheck(checkName string) (*github.CheckRun, error) {
	checks, _, err := c.Checks.ListCheckRunsForRef(c.Ctx, owner, c.Repo, c.Sha, &github.ListCheckRunsOptions{
		CheckName: &checkName,
	})
	if err != nil {
		return nil, err
	}
	if len(checks.CheckRuns) > 1 {
		return nil, fmt.Errorf("too many matching check runs found")
	}
	if len(checks.CheckRuns) == 0 {
		return nil, nil
	}
	return checks.CheckRuns[0], nil
}

func (c *Client) GetFile(filepath string) (string, bool, error) {
	fileContents, _, resp, err := c.Repositories.GetContents(c.Ctx, owner, c.Repo, filepath, &github.RepositoryContentGetOptions{
		Ref: c.Sha,
	})

	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			fmt.Printf("'%s' doesn't exist in `%s`\n", filepath, c.Repo)
			return "", false, nil
		}

		fmt.Printf(" Error occured while checking for '%s' in `%s`\n", filepath, c.Repo)
		return "", true, err
	}

	body, err := fileContents.GetContent()
	if err != nil {
		return "", true, err
	}

	return body, true, nil
}

func (c *Client) AddSkippingComment(reason, user string) error {
	prNumber, err := strconv.Atoi(c.PR)
	if err != nil {
		return err
	}

	hiddenTag := "<!-- CI SKIP COMMENT -->"
	commentMessage := fmt.Sprintf("🚨 **CI checks were skipped**\n\nSkip reason: `%s`\nSkipped by: @%s\n%s", reason, user, hiddenTag)

	// We need to first check for old comments and mark them as outdated to avoid noise
	comments, err := c.GetPRComments()
	if err != nil {
		return err
	}

	if len(comments) > 0 {
		lastComment := comments[len(comments)-1]
		if lastComment.Body != nil && *lastComment.Body == commentMessage {
			// The last comment on the PR is already our skip message
			return nil
		}

		for _, comment := range comments {
			if comment.Body != nil && strings.Contains(*comment.Body, hiddenTag) {
				query := `
					mutation minimizeComment($id: ID!, $classifier: ReportedContentClassifiers!) {
						minimizeComment(input: { subjectId: $id, classifier: $classifier }) {
							clientMutationId
							minimizedComment {
								isMinimized
								minimizedReason
								viewerCanMinimize
							}
						}
					}
				`
				variables := map[string]interface{}{
					"id":         comment.NodeID,
					"classifier": "OUTDATED",
				}

				req, _ := c.NewRequest("POST", "https://api.github.com/graphql", map[string]interface{}{
					"query":     query,
					"variables": variables,
				})
				var result map[string]interface{}
				_, err := c.Do(c.Ctx, req, &result)
				if err != nil {
					fmt.Println("Failed to mark comment as outdated")
				}
			}
		}
	}

	_, _, err = c.Issues.CreateComment(c.Ctx, owner, c.Repo, prNumber, &github.IssueComment{Body: &commentMessage})
	return err
}

func (c *Client) AddReasonRequiredComment() error {
	prNumber, err := strconv.Atoi(c.PR)
	if err != nil {
		return err
	}

	hiddenTag := "<!-- REASON REQUIRED -->"
	commentMessage := fmt.Sprintf("🚨 A reason is required when using the `/skip-ci` comment - e.g. `/skip-ci Test environment is currently down`\n%s", hiddenTag)

	// We need to first check for old comments and delete them to avoid noise
	comments, err := c.GetPRComments()
	if err != nil {
		return err
	}

	if len(comments) > 0 {
		lastComment := comments[len(comments)-1]
		if lastComment.Body != nil && *lastComment.Body == commentMessage {
			// The last comment on the PR is already our skip message
			return nil
		}

		for _, comment := range comments {
			if comment.Body != nil && strings.Contains(*comment.Body, hiddenTag) {
				_, _ = c.Issues.DeleteComment(c.Ctx, owner, c.Repo, *comment.ID)
			}
		}
	}

	_, _, err = c.Issues.CreateComment(c.Ctx, owner, c.Repo, prNumber, &github.IssueComment{Body: &commentMessage})
	return err
}

func (c *Client) AddSkipLabelDeprecatedComment() error {
	prNumber, err := strconv.Atoi(c.PR)
	if err != nil {
		return err
	}

	hiddenTag := "<!-- SKIP LABEL DEPRECATED -->"
	commentMessage := fmt.Sprintf("ℹ️ Please note: The `skip/ci` label no longer controls skipping the CI checks and is now purely informational.\n\nPlease use the `/skip-ci [reason]` comment trigger with a required reason message.\n%s", hiddenTag)

	// We need to first check for old comments and delete them to avoid noise
	comments, err := c.GetPRComments()
	if err != nil {
		return err
	}

	if len(comments) > 0 {
		lastComment := comments[len(comments)-1]
		if lastComment.Body != nil && *lastComment.Body == commentMessage {
			// The last comment on the PR is already our skip message
			return nil
		}

		for _, comment := range comments {
			if comment.Body != nil && strings.Contains(*comment.Body, hiddenTag) {
				_, _ = c.Issues.DeleteComment(c.Ctx, owner, c.Repo, *comment.ID)
			}
		}
	}

	_, _, err = c.Issues.CreateComment(c.Ctx, owner, c.Repo, prNumber, &github.IssueComment{Body: &commentMessage})
	return err
}

func (c *Client) AddSkipCILabel() error {
	prNumber, err := strconv.Atoi(c.PR)
	if err != nil {
		return err
	}

	_, _, err = c.Issues.AddLabelsToIssue(c.Ctx, owner, c.Repo, prNumber, []string{"skip/ci"})
	return err
}

func (c *Client) RemoveSkipCILabel() error {
	prNumber, err := strconv.Atoi(c.PR)
	if err != nil {
		return err
	}

	_, err = c.Issues.RemoveLabelForIssue(c.Ctx, owner, c.Repo, prNumber, "skip/ci")
	return err
}
