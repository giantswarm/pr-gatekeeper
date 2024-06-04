package github

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strconv"

	"github.com/google/go-github/v62/github"
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


func (c *Client) FilePresentInRepo(filepath string) (bool) {
	
	_, _, resp, err := c.Repositories.GetContents(c.Ctx, owner, c.Repo, filepath, &github.RepositoryContentGetOptions{
			Ref: c.Sha,
		})

	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			fmt.Printf("'%s' doesn't exist in `%s`\n", filepath, c.Repo)
			return false
		} else {
			fmt.Printf(" Error occured while checking for '%s' in `%s`\n", filepath, c.Repo)
			return false
		}
	} else {
		fmt.Printf("'%s' exists in `%s`\n", filepath, c.Repo)
		return true
	}
}
