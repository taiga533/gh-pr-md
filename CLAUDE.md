# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Language

All code, comments, commit messages, documentation, PR descriptions, and test case names in this repository must be written in English.

## Project Overview

A GitHub CLI (`gh`) extension that runs as the `gh pr-md` command. It displays PR information — including review comments, inline comments with diff hunks, and suggested changes — as formatted markdown in the terminal. Built with Go, using `github.com/cli/go-gh/v2` for GitHub API access, Cobra for CLI, and Glamour (via go-gh) for terminal rendering.

## Build & Run

```bash
# Build
go build -o gh-pr-md

# Run locally as a gh extension
gh pr-md

# Test
go test ./...

# Run a single test
go test -run TestFunctionName ./...
```

## Architecture

The data flows linearly through five packages:

```
cmd → resolver → ghapi → formatter → renderer → stdout
```

- **`cmd/`** — Cobra command definition, flag parsing (`-R`, `--no-diff`, `--no-color`), orchestrates the pipeline in `run()`.
- **`resolver/`** — Resolves CLI argument (PR number, GitHub URL, branch name, or auto-detect from current branch) into owner/repo/PR number. Uses `go-gh/pkg/repository` for repo detection.
- **`ghapi/`** — Fetches PR data via GitHub GraphQL API using raw queries (`Do()` method) with cursor-based pagination. Defines domain types (`PRData`, `Review`, `ReviewComment`, etc.) and the `Client` interface for testability.
- **`formatter/`** — Pure function converting `PRData` into a markdown string. Merges issue comments and reviews chronologically. Handles diff hunks, `--no-diff` file references, and ````suggestion` block conversion.
- **`renderer/`** — Wraps go-gh's `pkg/markdown.Render()` (Glamour). Strips trailing whitespace including ANSI-wrapped spaces. `--no-color` uses the "notty" style.

Integration tests in `integration_test.go` use a real PR fixture (`testdata/pr_12655.json`) to verify the full formatter→renderer pipeline.

## Testing (CRITICAL)

**Every implementation must have corresponding behavioral tests.** No code change should be merged without tests that verify the expected behavior. Tests must focus on observable behavior (inputs and outputs), not internal implementation details.

The `ghapi` package uses a `GraphQLClient` interface and `NewClientWithGraphQL()` to inject mock clients in tests. The `resolver` package uses a `PRFinder` interface and a replaceable `currentBranch` var for the same purpose.

## Release

Pushing a `v*` tag triggers GitHub Actions (`cli/gh-extension-precompile@v2`) to automatically build and release cross-compiled binaries.
