package formatter

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/taiga533/gh-pr-md/ghapi"
)

func newTime(s string) time.Time {
	t, _ := time.Parse(time.RFC3339, s)
	return t
}

func basePR() *ghapi.PRData {
	return &ghapi.PRData{
		Number: 42,
		Title:  "Add feature X",
		Body:   "This PR adds feature X.",
		Author: ghapi.User{Login: "alice"},
		Assignees: []ghapi.User{
			{Login: "bob"},
			{Login: "charlie"},
		},
	}
}

func TestFormat_HeaderContainsYAMLFrontmatterAndBody(t *testing.T) {
	pr := basePR()
	result := Format(pr, Options{})

	if !strings.Contains(result, "number: 42") {
		t.Error("expected number in frontmatter")
	}
	if !strings.Contains(result, `title: "Add feature X"`) {
		t.Error("expected title in frontmatter")
	}
	if !strings.Contains(result, `author: "alice"`) {
		t.Error("expected author in frontmatter")
	}
	if !strings.Contains(result, `- "bob"`) {
		t.Error("expected assignee bob in frontmatter")
	}
	if !strings.Contains(result, `- "charlie"`) {
		t.Error("expected assignee charlie in frontmatter")
	}
	if !strings.Contains(result, "This PR adds feature X.") {
		t.Error("expected body in output")
	}
	// Verify frontmatter delimiters
	if !strings.HasPrefix(result, "---\n") {
		t.Error("expected frontmatter to start with ---")
	}
	if strings.Count(result, "---\n") < 2 {
		t.Error("expected closing frontmatter delimiter")
	}
}

func TestFormat_EscapesDoubleQuotesInTitleAndAuthor(t *testing.T) {
	pr := basePR()
	pr.Title = `Fix "broken" test`
	pr.Author = ghapi.User{Login: `user"name`}
	result := Format(pr, Options{})

	if !strings.Contains(result, `title: "Fix \"broken\" test"`) {
		t.Errorf("expected escaped title in frontmatter, got:\n%s", result)
	}
	if !strings.Contains(result, `author: "user\"name"`) {
		t.Errorf("expected escaped author in frontmatter, got:\n%s", result)
	}
}

func TestFormat_EscapesBackslashesInTitle(t *testing.T) {
	pr := basePR()
	pr.Title = `Fix path\to\file`
	result := Format(pr, Options{})

	if !strings.Contains(result, `title: "Fix path\\to\\file"`) {
		t.Errorf("expected escaped backslashes in frontmatter, got:\n%s", result)
	}
}

func TestFormat_HandlesEmptyBodyWithoutError(t *testing.T) {
	pr := basePR()
	pr.Body = ""
	result := Format(pr, Options{})

	if !strings.Contains(result, `title: "Add feature X"`) {
		t.Error("expected title in frontmatter even with empty body")
	}
}

func TestFormat_OmitsAssigneesInFrontmatterWhenEmpty(t *testing.T) {
	pr := basePR()
	pr.Assignees = nil
	result := Format(pr, Options{})

	if strings.Contains(result, "assignees") {
		t.Error("expected no assignees field in frontmatter when no assignees")
	}
}

func TestFormat_SortsCommentsChronologically(t *testing.T) {
	pr := basePR()
	pr.Comments = []ghapi.IssueComment{
		{Author: ghapi.User{Login: "user2"}, Body: "Second comment", CreatedAt: newTime("2024-01-15T12:00:00Z")},
		{Author: ghapi.User{Login: "user1"}, Body: "First comment", CreatedAt: newTime("2024-01-15T10:00:00Z")},
	}
	pr.Reviews = []ghapi.Review{
		{
			Author: ghapi.User{Login: "user3"}, Body: "Middle review", State: "APPROVED",
			SubmittedAt: newTime("2024-01-15T11:00:00Z"),
		},
	}

	result := Format(pr, Options{})

	firstIdx := strings.Index(result, "First comment")
	middleIdx := strings.Index(result, "Middle review")
	secondIdx := strings.Index(result, "Second comment")

	if firstIdx == -1 || middleIdx == -1 || secondIdx == -1 {
		t.Fatal("expected all three entries in output")
	}
	if !(firstIdx < middleIdx && middleIdx < secondIdx) {
		t.Error("expected chronological order: first < middle < second")
	}
}

func TestFormat_IncludesDiffHunkInReviewComments(t *testing.T) {
	pr := basePR()
	pr.Reviews = []ghapi.Review{
		{
			Author: ghapi.User{Login: "reviewer"}, State: "COMMENTED",
			SubmittedAt: newTime("2024-01-15T10:00:00Z"),
			Comments: []ghapi.ReviewComment{
				{
					Author:   ghapi.User{Login: "reviewer"},
					Body:     "Fix this",
					Path:     "main.go",
					DiffHunk: "@@ -1,3 +1,5 @@\n+func hello() {}",
					CommitHash: "abc123def456",
					OriginalLine: 5,
					OriginalStartLine: 1,
				},
			},
		},
	}

	result := Format(pr, Options{})

	if !strings.Contains(result, "```diff") {
		t.Error("expected diff code block")
	}
	if !strings.Contains(result, "@@ -1,3 +1,5 @@") {
		t.Error("expected diff hunk content")
	}
	if !strings.Contains(result, "Fix this") {
		t.Error("expected review comment body")
	}
}

func TestFormat_ReplyCommentsOmitDiffHunk(t *testing.T) {
	pr := basePR()
	pr.Reviews = []ghapi.Review{
		{
			Author: ghapi.User{Login: "reviewer"}, State: "COMMENTED",
			SubmittedAt: newTime("2024-01-15T10:00:00Z"),
			Comments: []ghapi.ReviewComment{
				{
					ID:                "root-1",
					Author:            ghapi.User{Login: "reviewer"},
					Body:              "Root comment",
					Path:              "main.go",
					DiffHunk:          "@@ -1,3 +1,5 @@\n+func hello() {}",
					CreatedAt:         newTime("2024-01-15T10:00:00Z"),
					CommitHash:        "abc123def456",
					OriginalLine:      5,
					OriginalStartLine: 1,
				},
				{
					ID:                "reply-1",
					ReplyToID:         "root-1",
					Author:            ghapi.User{Login: "other"},
					Body:              "Reply body",
					Path:              "main.go",
					DiffHunk:          "@@ -1,3 +1,5 @@\n+func hello() {}",
					CreatedAt:         newTime("2024-01-15T11:00:00Z"),
					CommitHash:        "abc123def456",
					OriginalLine:      5,
					OriginalStartLine: 1,
				},
			},
		},
	}

	result := Format(pr, Options{})

	// Diff block should appear exactly once (for the root comment only).
	diffCount := strings.Count(result, "```diff")
	if diffCount != 1 {
		t.Errorf("expected exactly 1 diff block, got %d\n%s", diffCount, result)
	}
	if !strings.Contains(result, "Reply body") {
		t.Error("expected reply body in output")
	}
}

func TestFormat_ReplyCommentsOmitFileReferenceInNoDiffMode(t *testing.T) {
	pr := basePR()
	pr.Reviews = []ghapi.Review{
		{
			Author: ghapi.User{Login: "reviewer"}, State: "COMMENTED",
			SubmittedAt: newTime("2024-01-15T10:00:00Z"),
			Comments: []ghapi.ReviewComment{
				{
					ID:                "root-1",
					Author:            ghapi.User{Login: "reviewer"},
					Body:              "Root comment",
					Path:              "src/main.go",
					CreatedAt:         newTime("2024-01-15T10:00:00Z"),
					CommitHash:        "abc123def456",
					OriginalLine:      20,
					OriginalStartLine: 15,
				},
				{
					ID:                "reply-1",
					ReplyToID:         "root-1",
					Author:            ghapi.User{Login: "other"},
					Body:              "Reply in no-diff",
					Path:              "src/main.go",
					CreatedAt:         newTime("2024-01-15T11:00:00Z"),
					CommitHash:        "abc123def456",
					OriginalLine:      20,
					OriginalStartLine: 15,
				},
			},
		},
	}

	result := Format(pr, Options{NoDiff: true})

	// File reference should appear exactly once (for the root comment only).
	refCount := strings.Count(result, "abc123d@src/main.go")
	if refCount != 1 {
		t.Errorf("expected exactly 1 file reference, got %d\n%s", refCount, result)
	}
	if !strings.Contains(result, "Reply in no-diff") {
		t.Error("expected reply body in output")
	}
}

func TestFormat_NoDiffShowsFileReferenceInsteadOfDiff(t *testing.T) {
	pr := basePR()
	pr.Reviews = []ghapi.Review{
		{
			Author: ghapi.User{Login: "reviewer"}, State: "COMMENTED",
			SubmittedAt: newTime("2024-01-15T10:00:00Z"),
			Comments: []ghapi.ReviewComment{
				{
					Author:            ghapi.User{Login: "reviewer"},
					Body:              "Fix this",
					Path:              "src/main.go",
					DiffHunk:          "@@ -1,3 +1,5 @@\n+func hello() {}",
					CommitHash:        "abc123def456",
					OriginalLine:      20,
					OriginalStartLine: 15,
				},
			},
		},
	}

	result := Format(pr, Options{NoDiff: true})

	if strings.Contains(result, "```diff") {
		t.Error("expected no diff code block with NoDiff option")
	}
	if !strings.Contains(result, "abc123d@src/main.go#L15-20") {
		t.Errorf("expected file reference 'abc123d@src/main.go#L15-20' in output, got:\n%s", result)
	}
}

func TestFormat_NoDiffShowsSingleLineFileReference(t *testing.T) {
	pr := basePR()
	pr.Reviews = []ghapi.Review{
		{
			Author: ghapi.User{Login: "reviewer"}, State: "COMMENTED",
			SubmittedAt: newTime("2024-01-15T10:00:00Z"),
			Comments: []ghapi.ReviewComment{
				{
					Author:            ghapi.User{Login: "reviewer"},
					Body:              "Typo",
					Path:              "README.md",
					CommitHash:        "def456abc789",
					OriginalLine:      10,
					OriginalStartLine: 0,
				},
			},
		},
	}

	result := Format(pr, Options{NoDiff: true})

	if !strings.Contains(result, "def456a@README.md#L10") {
		t.Errorf("expected single-line file reference in output, got:\n%s", result)
	}
}

func TestFormat_DisplaysReviewStateLabelsCorrectly(t *testing.T) {
	tests := []struct {
		state    string
		expected string
	}{
		{"APPROVED", "approved"},
		{"CHANGES_REQUESTED", "requested changes"},
		{"COMMENTED", "reviewed"},
		{"DISMISSED", "dismissed review"},
	}

	for _, tt := range tests {
		t.Run(tt.state, func(t *testing.T) {
			pr := basePR()
			pr.Reviews = []ghapi.Review{
				{
					Author: ghapi.User{Login: "reviewer"}, State: tt.state,
					SubmittedAt: newTime("2024-01-15T10:00:00Z"),
				},
			}
			result := Format(pr, Options{})
			if !strings.Contains(result, tt.expected) {
				t.Errorf("expected label '%s' for state '%s' in output", tt.expected, tt.state)
			}
		})
	}
}

func TestFormat_ConvertsSuggestionBlockToLabeledCodeBlock(t *testing.T) {
	pr := basePR()
	pr.Reviews = []ghapi.Review{
		{
			Author: ghapi.User{Login: "reviewer"}, State: "COMMENTED",
			SubmittedAt: newTime("2024-01-15T10:00:00Z"),
			Comments: []ghapi.ReviewComment{
				{
					Author:       ghapi.User{Login: "reviewer"},
					Body:         "This can be simplified:\n\n```suggestion\nvar x = 1\n```",
					Path:         "main.go",
					DiffHunk:     "@@ -1,3 +1,5 @@\n+var x = 2",
					CommitHash:   "abc123",
					OriginalLine: 5,
				},
			},
		},
	}
	result := Format(pr, Options{})

	if !strings.Contains(result, "**Suggested change:**") {
		t.Error("expected 'Suggested change:' label")
	}
	if !strings.Contains(result, "var x = 1") {
		t.Error("expected suggestion code content")
	}
}

func TestFormat_ConvertsMultipleSuggestionBlocks(t *testing.T) {
	pr := basePR()
	pr.Reviews = []ghapi.Review{
		{
			Author: ghapi.User{Login: "reviewer"}, State: "COMMENTED",
			SubmittedAt: newTime("2024-01-15T10:00:00Z"),
			Comments: []ghapi.ReviewComment{
				{
					Author:       ghapi.User{Login: "reviewer"},
					Body:         "Fix both:\n\n```suggestion\nline1\n```\n\nAlso:\n\n```suggestion\nline2\n```",
					Path:         "main.go",
					DiffHunk:     "@@ -1 +1 @@",
					CommitHash:   "abc123",
					OriginalLine: 1,
				},
			},
		},
	}
	result := Format(pr, Options{})

	count := strings.Count(result, "**Suggested change:**")
	if count != 2 {
		t.Errorf("expected 2 suggestion labels, got %d", count)
	}
}

func TestFormat_SortsReviewCommentsByThread(t *testing.T) {
	pr := basePR()
	pr.Reviews = []ghapi.Review{
		{
			Author: ghapi.User{Login: "reviewer"}, State: "COMMENTED",
			SubmittedAt: newTime("2024-01-15T10:00:00Z"),
			Comments: []ghapi.ReviewComment{
				{
					ID:        "reply-b1",
					ReplyToID: "root-b",
					Author:    ghapi.User{Login: "reviewer"},
					Body:      "Reply to thread B",
					Path:      "main.go",
					CreatedAt: newTime("2024-01-15T12:00:00Z"),
				},
				{
					ID:     "root-a",
					Author: ghapi.User{Login: "reviewer"},
					Body:   "Thread A root (earliest)",
					Path:   "main.go",
					CreatedAt: newTime("2024-01-15T09:00:00Z"),
				},
				{
					ID:     "root-b",
					Author: ghapi.User{Login: "reviewer"},
					Body:   "Thread B root",
					Path:   "main.go",
					CreatedAt: newTime("2024-01-15T11:00:00Z"),
				},
			},
		},
	}

	result := Format(pr, Options{})

	rootAIdx := strings.Index(result, "Thread A root (earliest)")
	rootBIdx := strings.Index(result, "Thread B root")
	replyBIdx := strings.Index(result, "Reply to thread B")

	if rootAIdx == -1 || rootBIdx == -1 || replyBIdx == -1 {
		t.Fatal("expected all three comments in output")
	}
	if !(rootAIdx < rootBIdx) {
		t.Error("expected Thread A root before Thread B root (earlier parent)")
	}
	if !(rootBIdx < replyBIdx) {
		t.Error("expected Thread B root before its reply")
	}
}

func TestFormat_SortsRepliesWithinThreadByCreationTime(t *testing.T) {
	pr := basePR()
	pr.Reviews = []ghapi.Review{
		{
			Author: ghapi.User{Login: "reviewer"}, State: "COMMENTED",
			SubmittedAt: newTime("2024-01-15T10:00:00Z"),
			Comments: []ghapi.ReviewComment{
				{
					ID:        "reply-2",
					ReplyToID: "root-1",
					Author:    ghapi.User{Login: "reviewer"},
					Body:      "Second reply",
					Path:      "main.go",
					CreatedAt: newTime("2024-01-15T13:00:00Z"),
				},
				{
					ID:        "reply-1",
					ReplyToID: "root-1",
					Author:    ghapi.User{Login: "reviewer"},
					Body:      "First reply",
					Path:      "main.go",
					CreatedAt: newTime("2024-01-15T12:00:00Z"),
				},
				{
					ID:     "root-1",
					Author: ghapi.User{Login: "reviewer"},
					Body:   "Root comment",
					Path:   "main.go",
					CreatedAt: newTime("2024-01-15T10:00:00Z"),
				},
			},
		},
	}

	result := Format(pr, Options{})

	rootIdx := strings.Index(result, "Root comment")
	reply1Idx := strings.Index(result, "First reply")
	reply2Idx := strings.Index(result, "Second reply")

	if rootIdx == -1 || reply1Idx == -1 || reply2Idx == -1 {
		t.Fatal("expected all three comments in output")
	}
	if !(rootIdx < reply1Idx && reply1Idx < reply2Idx) {
		t.Error("expected root < first reply < second reply")
	}
}

func TestFormat_ThreadSortHandlesReplyToExternalParent(t *testing.T) {
	pr := basePR()
	// reply-ext references a parent not in this review's comments
	pr.Reviews = []ghapi.Review{
		{
			Author: ghapi.User{Login: "reviewer"}, State: "COMMENTED",
			SubmittedAt: newTime("2024-01-15T10:00:00Z"),
			Comments: []ghapi.ReviewComment{
				{
					ID:     "local-root",
					Author: ghapi.User{Login: "reviewer"},
					Body:   "Local root",
					Path:   "main.go",
					CreatedAt: newTime("2024-01-15T11:00:00Z"),
				},
				{
					ID:        "reply-ext",
					ReplyToID: "external-parent",
					Author:    ghapi.User{Login: "reviewer"},
					Body:      "Reply to external",
					Path:      "main.go",
					CreatedAt: newTime("2024-01-15T09:00:00Z"),
				},
			},
		},
	}

	result := Format(pr, Options{})

	extIdx := strings.Index(result, "Reply to external")
	localIdx := strings.Index(result, "Local root")

	if extIdx == -1 || localIdx == -1 {
		t.Fatal("expected both comments in output")
	}
	// reply-ext has CreatedAt 09:00 (used as proxy for its external parent),
	// local-root has CreatedAt 11:00, so external thread should come first.
	if !(extIdx < localIdx) {
		t.Error("expected external reply thread before local root (earlier proxy time)")
	}
}

func TestFormat_GroupsThreadCommentsAcrossReviews(t *testing.T) {
	pr := basePR()
	// Review A has the root comment of thread-1.
	// Review B has a reply to thread-1 and a new root for thread-2.
	// Expected output order: thread-1 root (09:00) -> thread-1 reply (12:00) -> thread-2 root (11:00)
	pr.Reviews = []ghapi.Review{
		{
			Author: ghapi.User{Login: "alice"}, State: "COMMENTED",
			SubmittedAt: newTime("2024-01-15T09:00:00Z"),
			Comments: []ghapi.ReviewComment{
				{
					ID:        "thread1-root",
					Author:    ghapi.User{Login: "alice"},
					Body:      "Thread 1 root from Review A",
					Path:      "main.go",
					CreatedAt: newTime("2024-01-15T09:00:00Z"),
				},
			},
		},
		{
			Author: ghapi.User{Login: "bob"}, State: "COMMENTED",
			SubmittedAt: newTime("2024-01-15T11:00:00Z"),
			Comments: []ghapi.ReviewComment{
				{
					ID:     "thread2-root",
					Author: ghapi.User{Login: "bob"},
					Body:   "Thread 2 root from Review B",
					Path:   "util.go",
					CreatedAt: newTime("2024-01-15T11:00:00Z"),
				},
				{
					ID:        "thread1-reply",
					ReplyToID: "thread1-root",
					Author:    ghapi.User{Login: "bob"},
					Body:      "Thread 1 reply from Review B",
					Path:      "main.go",
					CreatedAt: newTime("2024-01-15T12:00:00Z"),
				},
			},
		},
	}

	result := Format(pr, Options{})

	rootIdx := strings.Index(result, "Thread 1 root from Review A")
	replyIdx := strings.Index(result, "Thread 1 reply from Review B")
	thread2Idx := strings.Index(result, "Thread 2 root from Review B")

	if rootIdx == -1 || replyIdx == -1 || thread2Idx == -1 {
		t.Fatalf("expected all three comments in output, got:\n%s", result)
	}
	// Thread 1 (root at 09:00) should appear before thread 2 (root at 11:00).
	if !(rootIdx < thread2Idx) {
		t.Error("expected thread 1 root before thread 2 root (earlier parent)")
	}
	// Thread 1 reply should appear right after thread 1 root (same thread),
	// even though it came from a different review.
	if !(rootIdx < replyIdx && replyIdx < thread2Idx) {
		t.Error("expected thread 1 reply grouped with thread 1 root, before thread 2")
	}
}

func TestFormat_DoesNotConvertRegularCodeBlocks(t *testing.T) {
	pr := basePR()
	pr.Reviews = []ghapi.Review{
		{
			Author: ghapi.User{Login: "reviewer"}, State: "COMMENTED",
			SubmittedAt: newTime("2024-01-15T10:00:00Z"),
			Comments: []ghapi.ReviewComment{
				{
					Author:       ghapi.User{Login: "reviewer"},
					Body:         "Example:\n\n```go\nfmt.Println(\"hello\")\n```",
					Path:         "main.go",
					DiffHunk:     "@@ -1 +1 @@",
					CommitHash:   "abc123",
					OriginalLine: 1,
				},
			},
		},
	}
	result := Format(pr, Options{})

	if strings.Contains(result, "**Suggested change:**") {
		t.Error("expected no suggestion label for regular code block")
	}
}

func TestTrimDiffHunk_ReturnsEmptyForEmptyHunk(t *testing.T) {
	result := trimDiffHunk("", 0, 0)
	if result != "" {
		t.Errorf("expected empty string, got %q", result)
	}
}

func TestTrimDiffHunk_ReturnsUnchangedWhenOriginalLineIsZero(t *testing.T) {
	hunk := "@@ -1,3 +1,3 @@\n context\n-old\n+new"
	result := trimDiffHunk(hunk, 0, 0)
	if result != hunk {
		t.Errorf("expected unchanged hunk, got %q", result)
	}
}

func TestTrimDiffHunk_ReturnsUnchangedWhenEntireHunkIsInRange(t *testing.T) {
	// Hunk starts at new line 1, has 3 new-file lines (1,2,3).
	// Comment references lines 1-3, so all lines are in range.
	hunk := "@@ -1,3 +1,3 @@\n context\n-old\n+new\n last"
	result := trimDiffHunk(hunk, 1, 3)
	if result != hunk {
		t.Errorf("expected unchanged hunk, got:\n%s", result)
	}
}

func TestTrimDiffHunk_TrimsSingleLineComment(t *testing.T) {
	// Hunk: new lines 10,11,12,13,14. Comment on line 14 only.
	hunk := "@@ -10,5 +10,5 @@\n line10\n line11\n line12\n line13\n line14"
	result := trimDiffHunk(hunk, 0, 14)

	if strings.Contains(result, "line10") {
		t.Error("expected line10 to be trimmed")
	}
	if !strings.Contains(result, "line14") {
		t.Error("expected line14 to be present")
	}
}

func TestTrimDiffHunk_TrimsLinesToMultiLineRange(t *testing.T) {
	// Hunk: new lines 100..109. Comment references 107-109.
	lines := []string{"@@ -100,10 +100,10 @@"}
	for i := 100; i <= 109; i++ {
		lines = append(lines, fmt.Sprintf(" line%d", i))
	}
	hunk := strings.Join(lines, "\n")

	result := trimDiffHunk(hunk, 107, 109)

	if strings.Contains(result, "line100") {
		t.Error("expected line100 to be trimmed")
	}
	if strings.Contains(result, "line106") {
		t.Error("expected line106 to be trimmed")
	}
	if !strings.Contains(result, "line107") {
		t.Error("expected line107 to be present")
	}
	if !strings.Contains(result, "line109") {
		t.Error("expected line109 to be present")
	}
}

func TestTrimDiffHunk_PreservesInterspersedDeletionLines(t *testing.T) {
	// Lines: context(10), deleted, added(11), context(12)
	// Comment on 10-12 => should include the deletion line too
	hunk := "@@ -10,4 +10,3 @@\n context10\n-deleted\n+added11\n context12"
	result := trimDiffHunk(hunk, 10, 12)

	if !strings.Contains(result, "-deleted") {
		t.Error("expected deletion line to be preserved within range")
	}
	if !strings.Contains(result, "+added11") {
		t.Error("expected addition line to be preserved")
	}
}

func TestTrimDiffHunk_GeneratesCorrectHunkHeader(t *testing.T) {
	// Hunk: old 10-14, new 10-14 (5 context lines). Comment on new lines 13-14.
	hunk := "@@ -10,5 +10,5 @@\n line10\n line11\n line12\n line13\n line14"
	result := trimDiffHunk(hunk, 13, 14)

	if !strings.HasPrefix(result, "@@ -13,2 +13,2 @@") {
		t.Errorf("expected header @@ -13,2 +13,2 @@, got:\n%s", result)
	}
}

func TestTrimDiffHunk_PreservesFunctionContextInHeader(t *testing.T) {
	hunk := "@@ -10,5 +10,5 @@ func hello()\n line10\n line11\n line12\n line13\n line14"
	result := trimDiffHunk(hunk, 14, 14)

	if !strings.Contains(result, "@@ func hello()") {
		t.Errorf("expected function context preserved in header, got:\n%s", result)
	}
}

func TestTrimDiffHunk_ExpandsToIncludeAdjacentDeletionLines(t *testing.T) {
	// Deletion line right before the target range should be included
	// Lines: context(5), deleted(no new line), added(6)
	// Comment on line 6 only
	hunk := "@@ -5,3 +5,2 @@\n context5\n-deleted\n+added6"
	result := trimDiffHunk(hunk, 0, 6)

	if !strings.Contains(result, "-deleted") {
		t.Error("expected adjacent deletion line to be included")
	}
	if !strings.Contains(result, "+added6") {
		t.Error("expected added line to be present")
	}
}

func TestFormat_OutputsOnlyFrontmatterAndBodyWhenNoCommentsOrReviews(t *testing.T) {
	pr := basePR()
	result := Format(pr, Options{})

	if !strings.Contains(result, "number: 42") {
		t.Error("expected frontmatter in output")
	}
	// Should not contain any bold comment metadata
	// The frontmatter uses **bold** for nothing, and no comment sections should exist
	boldCount := strings.Count(result, "**")
	if boldCount > 0 {
		t.Error("expected no bold comment metadata sections")
	}
}
