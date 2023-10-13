package github

import (
	"context"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/google/go-github/v55/github"
	"golang.org/x/oauth2"
)

const (
	owner = "giantswarm"
)

var (
	checkName = "Heimdall - PR Gatekeeper"
	imageURL  = "https://github.com/giantswarm/pr-gatekeeper/assets/3384072/6c85d6be-6726-446c-ab37-c6c26601aec9"
	imageAlt  = "Heimdall"
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

func (c *Client) AddCheck(pending bool, msg string) error {
	status := "in_progress"
	summary := "PR currently blocked from merging"
	if !pending {
		status = "completed"
		summary = "PR meets all defined requirements for merging"
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
			Images: []*github.CheckRunImage{
				{
					Alt:      &imageAlt,
					ImageURL: &imageURL,
				},
			},
		},
	})
	return err
}

func (c *Client) AddRequiredCheck(name string) error {
	status := "queued"
	_, _, err := c.Checks.CreateCheckRun(c.Ctx, owner, c.Repo, github.CreateCheckRunOptions{
		Name:       name,
		HeadSHA:    c.Sha,
		Status:     &status,
		Conclusion: nil,
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

func (c *Client) AddComment(pending bool, msg string) error {
	prNumber, _ := strconv.Atoi(c.PR)

	emoji := ":stop_sign:"
	if !pending {
		emoji = ":white_check_mark:"
	}

	hiddenString := "<!-- PR Gatekeeper Comment -->"
	title := fmt.Sprintf("\n# %s Heimdall - PR Gatekeeper\n\n---\n\n", emoji)
	details := fmt.Sprintf(`<!-- DETAILS_START -->
%s
<!-- DETAILS_END -->`, msg)
	footer := `
---

_Source: https://github.com/giantswarm/pr-gatekeeper_ | _Repo Config: https://github.com/giantswarm/pr-gatekeeper/blob/main/config.yaml_`

	commentBody := hiddenString + title + details + footer

	comments, _, err := c.Issues.ListComments(c.Ctx, owner, c.Repo, prNumber, &github.IssueListCommentsOptions{})
	if err != nil {
		return err
	}
	for _, comment := range comments {
		if strings.Contains(*comment.Body, hiddenString) {
			if *comment.Body == commentBody {
				// No update needed
				return nil
			}

			re := regexp.MustCompile(`(?ms)<!-- DETAILS_START -->(.+)<!-- DETAILS_END -->`)
			match := re.FindStringSubmatch(*comment.Body)

			if len(match) > 1 {
				history := "\n\n---\n\n<details><summary>History</summary>\n" + match[1]

				re = regexp.MustCompile(`(?ms)<details><summary>History</summary>(.+)</details>`)
				match = re.FindStringSubmatch(*comment.Body)
				if len(match) > 1 {
					history += "\n\n" + match[1] + "\n\n"
				}

				history += "</details>\n\n"
				commentBody = hiddenString + title + details + history + footer
			}

			_, _, err = c.Issues.EditComment(c.Ctx, owner, c.Repo, *comment.ID, &github.IssueComment{
				Body: &commentBody,
			})
			return err
		}
	}

	_, _, err = c.Issues.CreateComment(c.Ctx, owner, c.Repo, prNumber, &github.IssueComment{
		Body: &commentBody,
	})

	return err
}
