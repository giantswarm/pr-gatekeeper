# Heimdall - PR Gatekeeper

A PR check run that ensures requirements are met before allowing PRs to be merged.

## Features

- Configure a list of PR Checks that must all be successful before merging
- Allow skipping these checks by placing a `skip/ci` label on the PR
- Force a recheck by adding a comment to a PR containing the trigger `/recheck`

## How it works

1. GitHub webhooks are configured on the appropriate repo as [documented in the tekton-resources repo](https://github.com/giantswarm/tekton-resources/blob/main/README.md#repo-setup).
2. Branch protection for the main branch should be updated to require the `Heimdall - PR Gatekeeper` status check to pass before merge.
3. When updates are made to a pull request (e.g. opened, syncronized, check run completed) an event is sent to Tekton which runs the `pr-gatekeeper` app against the PR in question.
4. Upon start `pr-gatekeeper` creates a new Check Run on the PR called `Heimdall - PR Gatekeeper` in an in-progress state.
5. The repos configuration will be loaded from [`repos.yaml`](./repos.yaml) and will confirm that the required PR checks are in a successful state. If they are, the `Heimdall - PR Gatekeeper` PR check is update to completed successfuly. If the checks haven't completed then the `Heimdall - PR Gatekeeper` PR check remains in-progress.
6. If the PR has the `skip/ci` label then all required checks will be ignored and the PR will be allowed to be merged.
