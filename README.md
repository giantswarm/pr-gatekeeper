# Heimdall - PR Gatekeeper

A PR check run that ensures requirements are met before allowing PRs to be merged.

## Features

- Configure a list of PR Checks that must all be successful before merging
- Allow skipping these checks by placing a `skip/ci` label on the PR
- Allow blocking PR merge by adding a `do-not-merge/hold` label on the PR
- Force a recheck by adding a comment to a PR containing the trigger `/recheck`

## How it works

1. GitHub webhooks are configured on the appropriate repo as [documented in the tekton-resources repo](https://github.com/giantswarm/tekton-resources/blob/main/README.md#repo-setup).
2. Branch protection for the main branch should be updated to require the `Heimdall - PR Gatekeeper` status check to pass before merge.
3. When updates are made to a pull request (e.g. opened, syncronized, check run completed) an event is sent to Tekton which runs the `pr-gatekeeper` app against the PR in question.
4. Upon start `pr-gatekeeper` creates a new Check Run on the PR called `Heimdall - PR Gatekeeper` in an in-progress state.
5. The repos configuration will be loaded from [`config.yaml`](./config.yaml) and will confirm that the required PR checks are in a successful state. If they are, the `Heimdall - PR Gatekeeper` PR check is update to completed successfuly. If the checks haven't completed then the `Heimdall - PR Gatekeeper` PR check remains in-progress.
6. If the PR has the `skip/ci` label then all required checks will be ignored and the PR will be allowed to be merged.
7. If the PR has the `do-not-merge/hold` label then the PR will be blocked from merging until this label is removed from the PR regardless of the other conditions being met.

## Releasing a new version

Currently this application doesn't make use of tagged releases and instead builds a new `latest` container image from the `main` branch.

This may change once automated updating of the image reference in the Tekton task can be handled but until then whatever is merged into `main` should match what is deployed.

## Adding Heimdall as a required PR check

Once the GitHub webhook has been configured on the repo as [documented in the tekton-resources repo](https://github.com/giantswarm/tekton-resources/blob/main/README.md#repo-setup) you can then run the below script to set Heimdall - PR Gatekeeper as a required check on PRs.

> Note: Requires `jq` to be installed and a valid `GITHUB_TOKEN` environment variable set.

```
REPO="default-apps-vsphere" # Replace with the repo name
BRANCH="main"               # Replace with the branch that has branch protection enabled

CHECKS=$(curl -L \
  -H "Accept: application/vnd.github+json" \
  -H "Authorization: Bearer ${GITHUB_TOKEN}"\
  -H "X-GitHub-Api-Version: 2022-11-28" \
  "https://api.github.com/repos/${REPO}/branches/${BRANCH}/protection/required_status_checks")

CHECKS=$(echo ${CHECKS} | jq -r '.contexts += ["Heimdall - PR Gatekeeper"] | .checks += [{"context": "Heimdall - PR Gatekeeper","app_id": 284804}]')

curl -L -X PATCH \
  -H "Accept: application/vnd.github+json" \
  -H "Authorization: Bearer ${GITHUB_TOKEN}" \
  -H "X-GitHub-Api-Version: 2022-11-28" \
  "https://api.github.com/repos/giantswarm/${REPO}/branches/${BRANCH}/protection/required_status_checks" -d "${CHECKS}"
```

## Adding (or updating) the supported labels to a repository

You will need a `GITHUB_TOKEN` environment variable set with a valid GitHub token.

```
REPO="giantswarm/cluster-aws" # Set this to the org/repo you want to add the labels too

curl --silent --fail -L -X PATCH \
  -H "Accept: application/vnd.github+json" -H "X-GitHub-Api-Version: 2022-11-28" -H "Authorization: Bearer ${GITHUB_TOKEN}"\
  "https://api.github.com/repos/${REPO}/labels/do-not-merge/hold" \
  -d '{"name":"do-not-merge/hold","description":"Instructs pr-gatekeeper to prevent a PR from being merged while the label is present","color":"B60205"}' || \
  curl --silent --fail -L -X POST \
    -H "Accept: application/vnd.github+json" -H "X-GitHub-Api-Version: 2022-11-28" -H "Authorization: Bearer ${GITHUB_TOKEN}"\
    "https://api.github.com/repos/${REPO}/labels" \
    -d '{"name":"do-not-merge/hold","description":"Instructs pr-gatekeeper to prevent a PR from being merged while the label is present","color":"B60205"}'


curl --silent --fail -L -X PATCH \
  -H "Accept: application/vnd.github+json" -H "X-GitHub-Api-Version: 2022-11-28" -H "Authorization: Bearer ${GITHUB_TOKEN}"\
  "https://api.github.com/repos/${REPO}/labels/skip/ci" \
  -d '{"name":"skip/ci","description":"Instructs pr-gatekeeper to ignore any required PR checks","color":"1D76DB"}' || \
  curl --silent --fail -L -X POST \
    -H "Accept: application/vnd.github+json" -H "X-GitHub-Api-Version: 2022-11-28" -H "Authorization: Bearer ${GITHUB_TOKEN}"\
    "https://api.github.com/repos/${REPO}/labels" \
    -d '{"name":"skip/ci","description":"Instructs pr-gatekeeper to ignore any required PR checks","color":"1D76DB"}'
```
