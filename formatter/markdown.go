package formatter

import (
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/taiga533/gh-pr-md/ghapi"
)

var suggestionBlockRe = regexp.MustCompile("(?s)```suggestion\\b[^\n]*\n(.*?)```")
var hunkHeaderRe = regexp.MustCompile(`^@@ -(\d+)(?:,(\d+))? \+(\d+)(?:,(\d+))? @@(.*)`)

// Options holds formatting options for markdown generation.
type Options struct {
	NoDiff bool
}

// timelineEntry represents a single entry in the chronological timeline.
type timelineEntry struct {
	timestamp time.Time
	content   string
}

// Format converts PR data into a markdown string.
func Format(pr *ghapi.PRData, opts Options) string {
	var sb strings.Builder

	writeHeader(&sb, pr)
	writeTimeline(&sb, pr, opts)

	return sb.String()
}

// writeHeader writes the PR title, body, and assignees section.
func writeHeader(sb *strings.Builder, pr *ghapi.PRData) {
	fmt.Fprintf(sb, "# #%d %s\n\n", pr.Number, pr.Title)
	fmt.Fprintf(sb, "**Author:** @%s\n\n", pr.Author.Login)

	if pr.Body != "" {
		sb.WriteString(pr.Body)
		sb.WriteString("\n\n")
	}

	if len(pr.Assignees) > 0 {
		logins := make([]string, len(pr.Assignees))
		for i, a := range pr.Assignees {
			logins[i] = "@" + a.Login
		}
		fmt.Fprintf(sb, "**Assignees:** %s\n\n", strings.Join(logins, ", "))
	}

	sb.WriteString("---\n\n")
}

// writeTimeline collects all comments and reviews, sorts them chronologically,
// and writes them to the markdown output.
func writeTimeline(sb *strings.Builder, pr *ghapi.PRData, opts Options) {
	var entries []timelineEntry

	// Add issue comments
	for _, c := range pr.Comments {
		entries = append(entries, timelineEntry{
			timestamp: c.CreatedAt,
			content:   formatIssueComment(c),
		})
	}

	// Add reviews (with their inline comments)
	for _, r := range pr.Reviews {
		entries = append(entries, timelineEntry{
			timestamp: r.SubmittedAt,
			content:   formatReview(r, opts),
		})
	}

	// Sort chronologically
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].timestamp.Before(entries[j].timestamp)
	})

	for _, e := range entries {
		sb.WriteString(e.content)
	}
}

// formatIssueComment formats a general PR comment as markdown.
func formatIssueComment(c ghapi.IssueComment) string {
	var sb strings.Builder
	fmt.Fprintf(&sb, "### @%s commented on %s\n\n", c.Author.Login, formatTime(c.CreatedAt))
	if c.Body != "" {
		sb.WriteString(c.Body)
		sb.WriteString("\n\n")
	}
	return sb.String()
}

// formatReview formats a review and its inline comments as markdown.
func formatReview(r ghapi.Review, opts Options) string {
	var sb strings.Builder

	stateLabel := formatReviewState(r.State)
	fmt.Fprintf(&sb, "### @%s %s on %s\n\n", r.Author.Login, stateLabel, formatTime(r.SubmittedAt))

	if r.Body != "" {
		sb.WriteString(r.Body)
		sb.WriteString("\n\n")
	}

	for _, rc := range r.Comments {
		sb.WriteString(formatReviewComment(rc, opts))
	}

	return sb.String()
}

// formatReviewComment formats an inline review comment as markdown.
func formatReviewComment(rc ghapi.ReviewComment, opts Options) string {
	var sb strings.Builder

	fmt.Fprintf(&sb, "#### @%s commented on `%s`\n\n", rc.Author.Login, rc.Path)

	if opts.NoDiff {
		sb.WriteString(formatFileReference(rc))
		sb.WriteString("\n\n")
	} else if rc.DiffHunk != "" {
		sb.WriteString("```diff\n")
		sb.WriteString(trimDiffHunk(rc.DiffHunk, rc.OriginalStartLine, rc.OriginalLine))
		sb.WriteString("\n```\n\n")
	}

	if rc.Body != "" {
		sb.WriteString(convertSuggestions(rc.Body))
		sb.WriteString("\n\n")
	}

	return sb.String()
}

// convertSuggestions replaces GitHub ```suggestion blocks with a labeled code block.
func convertSuggestions(body string) string {
	return suggestionBlockRe.ReplaceAllString(body, "**Suggested change:**\n```suggestion\n${1}```")
}

// trimDiffHunk trims a unified diff hunk to only include lines within the
// referenced line range (originalStartLine to originalLine). Returns the
// original hunk unchanged if parsing fails or all lines are already in range.
func trimDiffHunk(diffHunk string, originalStartLine, originalLine int) string {
	if originalLine == 0 || diffHunk == "" {
		return diffHunk
	}

	lines := strings.Split(diffHunk, "\n")
	if len(lines) == 0 {
		return diffHunk
	}

	matches := hunkHeaderRe.FindStringSubmatch(lines[0])
	if matches == nil {
		return diffHunk
	}

	oldStart, _ := strconv.Atoi(matches[1])
	newStart, _ := strconv.Atoi(matches[3])
	funcContext := matches[5]

	startLine := originalStartLine
	if startLine <= 0 {
		startLine = originalLine
	}

	// Track line numbers and find the range of diff-line indices to keep.
	type lineInfo struct {
		newLineNum int
		oldLineNum int
	}
	infos := make([]lineInfo, len(lines)-1)
	curNew := newStart
	curOld := oldStart
	for i := 1; i < len(lines); i++ {
		line := lines[i]
		if len(line) == 0 {
			// Empty line treated as context
			infos[i-1] = lineInfo{newLineNum: curNew, oldLineNum: curOld}
			curNew++
			curOld++
			continue
		}
		switch line[0] {
		case '+':
			infos[i-1] = lineInfo{newLineNum: curNew, oldLineNum: 0}
			curNew++
		case '-':
			infos[i-1] = lineInfo{newLineNum: 0, oldLineNum: curOld}
			curOld++
		default: // context line (space prefix or other)
			infos[i-1] = lineInfo{newLineNum: curNew, oldLineNum: curOld}
			curNew++
			curOld++
		}
	}

	// Find first and last diff-line indices that fall within the target range.
	firstIdx := -1
	lastIdx := -1
	for i, info := range infos {
		if info.newLineNum >= startLine && info.newLineNum <= originalLine && info.newLineNum > 0 {
			if firstIdx == -1 {
				firstIdx = i
			}
			lastIdx = i
		}
	}

	if firstIdx == -1 {
		return diffHunk
	}

	// Expand to include adjacent deletion lines that are part of the same change.
	for firstIdx > 0 && infos[firstIdx-1].newLineNum == 0 {
		firstIdx--
	}
	for lastIdx < len(infos)-1 && infos[lastIdx+1].newLineNum == 0 {
		lastIdx++
	}

	// If the entire hunk is selected, return unchanged.
	if firstIdx == 0 && lastIdx == len(infos)-1 {
		return diffHunk
	}

	// Build new hunk header with correct line counts.
	selected := lines[firstIdx+1 : lastIdx+2] // +1 because infos is offset by 1 from lines
	newHunkStart := 0
	oldHunkStart := 0
	newCount := 0
	oldCount := 0
	for i := firstIdx; i <= lastIdx; i++ {
		info := infos[i]
		line := lines[i+1]
		prefix := byte(' ')
		if len(line) > 0 {
			prefix = line[0]
		}
		switch prefix {
		case '+':
			if newHunkStart == 0 {
				newHunkStart = info.newLineNum
			}
			newCount++
		case '-':
			if oldHunkStart == 0 {
				oldHunkStart = info.oldLineNum
			}
			oldCount++
		default:
			if newHunkStart == 0 {
				newHunkStart = info.newLineNum
			}
			if oldHunkStart == 0 {
				oldHunkStart = info.oldLineNum
			}
			newCount++
			oldCount++
		}
	}
	if oldHunkStart == 0 {
		oldHunkStart = oldStart
	}

	header := fmt.Sprintf("@@ -%d,%d +%d,%d @@%s", oldHunkStart, oldCount, newHunkStart, newCount, funcContext)
	result := make([]string, 0, len(selected)+1)
	result = append(result, header)
	result = append(result, selected...)
	return strings.Join(result, "\n")
}

// formatFileReference formats a file reference with commit hash and line numbers.
func formatFileReference(rc ghapi.ReviewComment) string {
	shortHash := rc.CommitHash
	if len(shortHash) > 7 {
		shortHash = shortHash[:7]
	}

	if rc.OriginalStartLine > 0 && rc.OriginalStartLine != rc.OriginalLine {
		return fmt.Sprintf("`%s@%s#L%d-%d`", shortHash, rc.Path, rc.OriginalStartLine, rc.OriginalLine)
	}
	return fmt.Sprintf("`%s@%s#L%d`", shortHash, rc.Path, rc.OriginalLine)
}

// formatReviewState converts a review state to a human-readable label.
func formatReviewState(state string) string {
	switch state {
	case "APPROVED":
		return "approved"
	case "CHANGES_REQUESTED":
		return "requested changes"
	case "COMMENTED":
		return "reviewed"
	case "DISMISSED":
		return "dismissed review"
	default:
		return "reviewed"
	}
}

// formatTime formats a timestamp for display.
func formatTime(t time.Time) string {
	return t.Format("2006-01-02 15:04")
}
