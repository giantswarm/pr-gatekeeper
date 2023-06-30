# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

[Unreleased]: https://github.com/giantswarm/REPOSITORY_NAME/tree/master

### Added

* Added the commit hash to the details to make it clearer its not related to the PR as a whole
* Added support for the `do-not-merge/hold` label to block merging.

### Changed

* If the test is one we know the trigger for we add a note on how to trigger it
* If test is found but not yet completed we add a "... but is still in progress" message (not shown here)
* If a test is found to have failed we add a "... but didn't complete successfully" message (not shown here)
