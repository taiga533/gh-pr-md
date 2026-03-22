package ghapi

import "time"

// PRData represents the complete data for a pull request.
type PRData struct {
	Number    int
	Title     string
	Body      string
	Author    User
	Assignees []User
	Comments  []IssueComment
	Reviews   []Review
}

// User represents a GitHub user.
type User struct {
	Login string
}

// IssueComment represents a general comment on a pull request.
type IssueComment struct {
	Author    User
	Body      string
	CreatedAt time.Time
}

// Review represents a pull request review.
type Review struct {
	Author      User
	Body        string
	State       string // APPROVED, CHANGES_REQUESTED, COMMENTED, DISMISSED
	SubmittedAt time.Time
	Comments    []ReviewComment
}

// ReviewComment represents an inline comment on a pull request diff.
type ReviewComment struct {
	ID                string
	ReplyToID         string
	Author            User
	Body              string
	Path              string
	DiffHunk          string
	CreatedAt         time.Time
	CommitHash        string
	OriginalLine      int
	OriginalStartLine int
}
