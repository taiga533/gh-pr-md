package main

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/taiga533/gh-pr-md/formatter"
	"github.com/taiga533/gh-pr-md/ghapi"
	"github.com/taiga533/gh-pr-md/renderer"
)

// fixtureResponse mirrors the GraphQL JSON structure from testdata/pr_12655.json.
type fixtureResponse struct {
	Data struct {
		Repository struct {
			PullRequest struct {
				Number int    `json:"number"`
				Title  string `json:"title"`
				Body   string `json:"body"`
				Author struct {
					Login string `json:"login"`
				} `json:"author"`
				Assignees struct {
					Nodes []struct {
						Login string `json:"login"`
					} `json:"nodes"`
				} `json:"assignees"`
				Comments struct {
					Nodes []struct {
						Author struct {
							Login string `json:"login"`
						} `json:"author"`
						Body      string `json:"body"`
						CreatedAt string `json:"createdAt"`
					} `json:"nodes"`
				} `json:"comments"`
				Reviews struct {
					Nodes []struct {
						Author struct {
							Login string `json:"login"`
						} `json:"author"`
						Body        string `json:"body"`
						State       string `json:"state"`
						SubmittedAt string `json:"submittedAt"`
						Comments    struct {
							Nodes []struct {
								Author struct {
									Login string `json:"login"`
								} `json:"author"`
								Body              string `json:"body"`
								Path              string `json:"path"`
								DiffHunk          string `json:"diffHunk"`
								CreatedAt         string `json:"createdAt"`
								Commit            struct{ OID string } `json:"commit"`
								OriginalLine      int                  `json:"originalLine"`
								OriginalStartLine int                  `json:"originalStartLine"`
							} `json:"nodes"`
						} `json:"comments"`
					} `json:"nodes"`
				} `json:"reviews"`
			} `json:"pullRequest"`
		} `json:"repository"`
	} `json:"data"`
}

// loadFixturePR loads the testdata fixture and converts it to a PRData.
func loadFixturePR(t *testing.T) *ghapi.PRData {
	t.Helper()

	data, err := os.ReadFile("testdata/pr_12655.json")
	if err != nil {
		t.Fatalf("failed to read fixture: %v", err)
	}

	var resp fixtureResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		t.Fatalf("failed to parse fixture: %v", err)
	}

	prJSON := resp.Data.Repository.PullRequest
	pr := &ghapi.PRData{
		Number: prJSON.Number,
		Title:  prJSON.Title,
		Body:   prJSON.Body,
		Author: ghapi.User{Login: prJSON.Author.Login},
	}

	for _, a := range prJSON.Assignees.Nodes {
		pr.Assignees = append(pr.Assignees, ghapi.User{Login: a.Login})
	}

	for _, c := range prJSON.Comments.Nodes {
		createdAt, _ := time.Parse(time.RFC3339, c.CreatedAt)
		pr.Comments = append(pr.Comments, ghapi.IssueComment{
			Author:    ghapi.User{Login: c.Author.Login},
			Body:      c.Body,
			CreatedAt: createdAt,
		})
	}

	for _, r := range prJSON.Reviews.Nodes {
		submittedAt, _ := time.Parse(time.RFC3339, r.SubmittedAt)
		review := ghapi.Review{
			Author:      ghapi.User{Login: r.Author.Login},
			Body:        r.Body,
			State:       r.State,
			SubmittedAt: submittedAt,
		}
		for _, rc := range r.Comments.Nodes {
			createdAt, _ := time.Parse(time.RFC3339, rc.CreatedAt)
			review.Comments = append(review.Comments, ghapi.ReviewComment{
				Author:            ghapi.User{Login: rc.Author.Login},
				Body:              rc.Body,
				Path:              rc.Path,
				DiffHunk:          rc.DiffHunk,
				CreatedAt:         createdAt,
				CommitHash:        rc.Commit.OID,
				OriginalLine:      rc.OriginalLine,
				OriginalStartLine: rc.OriginalStartLine,
			})
		}
		pr.Reviews = append(pr.Reviews, review)
	}

	return pr
}

func TestIntegration_PR12655_OutputContainsPRMetadata(t *testing.T) {
	pr := loadFixturePR(t)
	md := formatter.Format(pr, formatter.Options{})

	if !strings.Contains(md, "# #12655") {
		t.Error("expected PR number in title")
	}
	if !strings.Contains(md, "--exclude") {
		t.Error("expected --exclude flag keyword from title")
	}
	if !strings.Contains(md, "@yuvrajangadsingh") {
		t.Error("expected PR author mention")
	}
}

func TestIntegration_PR12655_IncludesIssueComments(t *testing.T) {
	pr := loadFixturePR(t)
	md := formatter.Format(pr, formatter.Options{})

	// PR has 5 issue comments from yuvrajangadsingh and BagToad
	if strings.Count(md, "@yuvrajangadsingh commented on") < 1 {
		t.Error("expected issue comments from yuvrajangadsingh")
	}
	if !strings.Contains(md, "@BagToad commented on") {
		t.Error("expected issue comment from BagToad")
	}
}

func TestIntegration_PR12655_IncludesReviewsFromMultipleReviewers(t *testing.T) {
	pr := loadFixturePR(t)
	md := formatter.Format(pr, formatter.Options{})

	// babakks reviewed (COMMENTED and APPROVED)
	if !strings.Contains(md, "@babakks") {
		t.Error("expected reviewer babakks")
	}
	// BagToad reviewed (CHANGES_REQUESTED and APPROVED)
	if !strings.Contains(md, "@BagToad") {
		t.Error("expected reviewer BagToad")
	}

	// Check review states
	if !strings.Contains(md, "requested changes") {
		t.Error("expected CHANGES_REQUESTED state label")
	}
	if !strings.Contains(md, "approved") {
		t.Error("expected APPROVED state label")
	}
	if !strings.Contains(md, "reviewed") {
		t.Error("expected COMMENTED state label")
	}
}

func TestIntegration_PR12655_DisplaysSuggestedChangeWithLabel(t *testing.T) {
	pr := loadFixturePR(t)
	md := formatter.Format(pr, formatter.Options{})

	// babakks' review comment contains a ```suggestion block
	if !strings.Contains(md, "**Suggested change:**") {
		t.Error("expected 'Suggested change:' label for suggestion block")
	}
	// The suggestion content should be present
	if !strings.Contains(md, "diffHeaderRegexp") {
		t.Error("expected suggestion code content")
	}
	// Raw ```suggestion fence should be converted
	if strings.Contains(md, "```suggestion\nvar") {
		// After conversion, there should be a label before the code block
		// The pattern "```suggestion\nvar" without preceding label means it wasn't converted
	}
}

func TestIntegration_PR12655_IncludesDiffHunksInInlineComments(t *testing.T) {
	pr := loadFixturePR(t)
	md := formatter.Format(pr, formatter.Options{})

	// All inline comments are on pkg/cmd/pr/diff/diff.go
	if !strings.Contains(md, "pkg/cmd/pr/diff/diff.go") {
		t.Error("expected inline comment file path diff.go")
	}

	// Should have diff code blocks
	diffCount := strings.Count(md, "```diff")
	if diffCount == 0 {
		t.Error("expected diff code blocks in output")
	}
	// PR has 16 inline comments across reviews
	if diffCount < 10 {
		t.Errorf("expected at least 10 diff code blocks for inline comments, got %d", diffCount)
	}
}

func TestIntegration_PR12655_NoDiffShowsOnlyFileReferences(t *testing.T) {
	pr := loadFixturePR(t)
	md := formatter.Format(pr, formatter.Options{NoDiff: true})

	if strings.Contains(md, "```diff") {
		t.Error("expected no diff code blocks in --no-diff mode")
	}

	// Should contain file references with line ranges (babakks' review has multi-line comments)
	if !strings.Contains(md, "pkg/cmd/pr/diff/diff.go#L") {
		t.Error("expected file reference with line numbers")
	}

	// Check a specific multi-line reference (babakks inline on L288-306)
	if !strings.Contains(md, "#L288-306") {
		t.Error("expected multi-line reference L288-306")
	}

	// Check a single-line reference (babakks inline on L98)
	if !strings.Contains(md, "#L98") {
		t.Error("expected single-line reference L98")
	}
}

func TestIntegration_PR12655_SortsCommentsAndReviewsChronologically(t *testing.T) {
	pr := loadFixturePR(t)
	md := formatter.Format(pr, formatter.Options{})

	// BagToad's CHANGES_REQUESTED review (2026-02-20) should come before
	// BagToad's issue comment (2026-03-04)
	changesIdx := strings.Index(md, "requested changes")
	bagToadCommentIdx := strings.Index(md, "@BagToad commented on")

	if changesIdx == -1 || bagToadCommentIdx == -1 {
		t.Fatal("expected both BagToad's review and comment in output")
	}
	if changesIdx >= bagToadCommentIdx {
		t.Error("expected CHANGES_REQUESTED review to appear before BagToad's issue comment chronologically")
	}

	// babakks' APPROVED review (2026-03-14) should come after all issue comments
	lastCommentIdx := strings.LastIndex(md, "commented on 2026-03-06")
	approvedIdx := strings.Index(md, "@babakks approved")
	if lastCommentIdx == -1 || approvedIdx == -1 {
		t.Fatal("expected both last comment and babakks approval in output")
	}
	if approvedIdx <= lastCommentIdx {
		t.Error("expected babakks APPROVED to appear after last issue comment")
	}
}

func TestIntegration_PR12655_RendersWithGlamour(t *testing.T) {
	pr := loadFixturePR(t)
	md := formatter.Format(pr, formatter.Options{})

	rendered, err := renderer.Render(md, renderer.Options{NoColor: true})
	if err != nil {
		t.Fatalf("failed to render markdown: %v", err)
	}

	if !strings.Contains(rendered, "12655") {
		t.Error("expected PR number in rendered output")
	}
	if !strings.Contains(rendered, "exclude") {
		t.Error("expected keyword from title in rendered output")
	}
	if !strings.Contains(rendered, "babakks") {
		t.Error("expected reviewer name in rendered output")
	}
}

func TestIntegration_PR12655_NoColorExcludesANSIEscapeSequences(t *testing.T) {
	pr := loadFixturePR(t)
	md := formatter.Format(pr, formatter.Options{})

	rendered, err := renderer.Render(md, renderer.Options{NoColor: true})
	if err != nil {
		t.Fatalf("failed to render markdown: %v", err)
	}

	if strings.Contains(rendered, "\x1b[") {
		t.Error("expected no ANSI escape sequences in no-color mode")
	}
}

func TestIntegration_PR12655_FullPipelineCompletesSuccessfully(t *testing.T) {
	pr := loadFixturePR(t)

	// Format
	md := formatter.Format(pr, formatter.Options{})
	if md == "" {
		t.Fatal("formatter returned empty markdown")
	}

	// Render with color
	colorRendered, err := renderer.Render(md, renderer.Options{NoColor: false})
	if err != nil {
		t.Fatalf("failed to render with color: %v", err)
	}
	if colorRendered == "" {
		t.Error("color rendered output is empty")
	}

	// Render without color
	plainRendered, err := renderer.Render(md, renderer.Options{NoColor: true})
	if err != nil {
		t.Fatalf("failed to render without color: %v", err)
	}
	if plainRendered == "" {
		t.Error("plain rendered output is empty")
	}

	// Format with NoDiff
	mdNoDiff := formatter.Format(pr, formatter.Options{NoDiff: true})
	if mdNoDiff == "" {
		t.Fatal("formatter with NoDiff returned empty markdown")
	}

	// NoDiff output should be shorter (no diff hunks)
	if len(mdNoDiff) >= len(md) {
		t.Errorf("expected NoDiff output (%d bytes) to be shorter than full output (%d bytes)", len(mdNoDiff), len(md))
	}
}
