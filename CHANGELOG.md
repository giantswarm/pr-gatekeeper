# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

[Unreleased]: https://github.com/giantswarm/pr-gatekeeper/tree/master

### Fixed

* Correctly check for the require App Test checks for each provider when ./tests/e2e/config.yaml present in repo.
* Fix linting issues.

### Added

* Require `App E2E Test Suites` for every provider a full `/run app-test-suites` covers, taking the union of the providers declared in `./tests/e2e/config.yaml` and in the per-suite configs under `./tests/e2e/suites`, and defaulting to `capa` for configs that declare none. Previously only the providers named in the top level config were required, so a provider declared solely by a test suite could be missing entirely and the PR would still be mergeable.
* Show the `/run app-test-suites-single PROVIDER=<provider>` trigger hint for the per-provider `App E2E Test Suites` checks by matching known triggers against dynamic check names by prefix.
* Require the `E2E Coverage` check on `releases` PRs, so a release is only mergeable once every expected test suite has passed and not just the ones the current release stage runs.
* Require a successful `Generate MC` (MC creation test) for at least one provider updated in a `releases` PR before it can be merged.
* Added the commit hash to the details to make it clearer its not related to the PR as a whole
* Added support for the `do-not-merge/hold` label to block merging.
* Added `mc-bootstrap` required checks
* Added `securityContext` to Tekton Tasks
* Implement support for `/skip-ci [reason]` comment trigger to replace old `skip/ci` label
* Added `E2E Test Suites` required check for `cluster-eks`

### Changed

* If the test is one we know the trigger for we add a note on how to trigger it
* If test is found but not yet completed we add a "... but is still in progress" message (not shown here)
* If a test is found to have failed we add a "... but didn't complete successfully" message (not shown here)
* Label overrides can be used even if repo doesn't have any config setup
* Bumped Go to v1.20
* Migrated image registry to ACR
* Automatically add "E2E Test Suites" required check when ./tests/e2e/config.yaml present in repo.
* Go: Update dependencies.

### Removed

* Removed support of using the `skip/ci` label for bypassing the gatekeeper - replaced with the `/skip-ci [reason]` comment trigger
