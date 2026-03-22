# gh-pr-md

English | [日本語](README_ja.md)

A [GitHub CLI](https://cli.github.com/) extension that outputs **full PR information** — including review comments, inline comments with diff hunks, and suggested changes — as formatted Markdown in the terminal.

Built for both **AI Agents** and **humans** who need the complete picture of a pull request in a single, readable Markdown stream.

## Installation

```bash
gh extension install taiga533/gh-pr-md
```

> Requires [GitHub CLI](https://cli.github.com/) (`gh`) v2.0+

## Usage

```bash
gh pr-md [<number> | <url> | <branch>]
```

**Arguments** (all optional — omit to auto-detect from current branch):

| Argument | Example |
|----------|---------|
| PR number | `gh pr-md 123` |
| GitHub URL | `gh pr-md https://github.com/owner/repo/pull/123` |
| Branch name | `gh pr-md feature-branch` |

**Flags:**

| Flag | Description |
|------|-------------|
| `-R`, `--repo [HOST/]OWNER/REPO` | Target a specific repository |
| `--no-diff` | Omit diff hunks; show file references only (e.g. `abc123@path/to/file.go#L10-20`) |
| `--no-color` | Disable ANSI color output |

**Pipe-friendly:** When stdout is not a TTY (e.g. piped to another command or file), `gh pr-md` outputs plain Markdown without rendering — ideal for feeding into AI agents or other tools.

```bash
# Feed a PR into an AI agent
gh pr-md 123 | your-ai-tool

# Save to file
gh pr-md 123 > pr-review.md
```

## Why gh-pr-md?

The built-in `gh pr view --comments` has a significant blind spot: **it cannot retrieve review comments or inline code comments.** These are often the most important part of a code review — the line-by-line feedback, suggested changes, and threaded discussions that drive a PR toward merge.

`gh pr-md` fills this gap:

- **Review comments & inline comments** — Fetches every comment type, including those attached to specific lines of code.
- **Diff hunks** — Shows the exact code context where each inline comment was made, trimmed to the relevant lines.
- **Suggested changes** — Renders GitHub's ` ```suggestion ` blocks clearly so you can see what was proposed.
- **Chronological timeline** — Merges issue comments, reviews, and inline comments into a single time-ordered stream.
- **Threaded discussions** — Groups reply chains on inline comments so conversations read naturally.
- **Terminal rendering** — Uses [Glamour](https://github.com/charmbracelet/glamour) for styled Markdown with syntax highlighting in the terminal.
- **Plain Markdown output** — When piped, outputs raw Markdown that AI agents and LLMs can parse directly.

## Output Format

The output is structured as follows:

```
---
number: 123
title: "Add feature X"
author: username
assignees: reviewer1, reviewer2
---

## Description

PR body content here...

---

**@commenter** commented on 2024-01-15 10:30:00

Comment body...

---

**@reviewer** requested changes on 2024-01-15 11:00:00

Review body...

> **@reviewer** commented on `path/to/file.go`
>
> ```diff
>  func example() {
> -    old code
> +    new code
>  }
> ```
>
> Inline comment body...
```

## License

[MIT](LICENSE)
