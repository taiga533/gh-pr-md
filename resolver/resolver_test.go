package resolver

import (
	"fmt"
	"testing"
)

type mockPRFinder struct {
	prNumber int
	err      error
}

func (m *mockPRFinder) FindPRByBranch(owner, repo, branch string) (int, error) {
	return m.prNumber, m.err
}

func TestResolve_WithPRNumber(t *testing.T) {
	// Override currentBranch to avoid git dependency
	origCurrentBranch := currentBranch
	defer func() { currentBranch = origCurrentBranch }()

	finder := &mockPRFinder{prNumber: 42}

	// We need to override resolveRepo since we're not in a real git repo
	// Instead, use a repo flag
	result, err := Resolve("123", "owner/repo", finder)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result.PRNumber != 123 {
		t.Errorf("expected PR number 123, got %d", result.PRNumber)
	}
	if result.Owner != "owner" {
		t.Errorf("expected owner 'owner', got '%s'", result.Owner)
	}
	if result.Repo != "repo" {
		t.Errorf("expected repo 'repo', got '%s'", result.Repo)
	}
}

func TestResolve_WithURL(t *testing.T) {
	finder := &mockPRFinder{}

	result, err := Resolve("https://github.com/myorg/myrepo/pull/42", "", finder)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result.PRNumber != 42 {
		t.Errorf("expected PR number 42, got %d", result.PRNumber)
	}
	if result.Owner != "myorg" {
		t.Errorf("expected owner 'myorg', got '%s'", result.Owner)
	}
	if result.Repo != "myrepo" {
		t.Errorf("expected repo 'myrepo', got '%s'", result.Repo)
	}
	if result.Host != "github.com" {
		t.Errorf("expected host 'github.com', got '%s'", result.Host)
	}
}

func TestResolve_WithBranchName(t *testing.T) {
	finder := &mockPRFinder{prNumber: 99}

	result, err := Resolve("feature-branch", "owner/repo", finder)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result.PRNumber != 99 {
		t.Errorf("expected PR number 99, got %d", result.PRNumber)
	}
}

func TestResolve_ReturnsErrorWhenNoPRFoundForBranch(t *testing.T) {
	finder := &mockPRFinder{err: fmt.Errorf("no PR found")}

	_, err := Resolve("feature-branch", "owner/repo", finder)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestResolve_AutoDetectsFromCurrentBranch(t *testing.T) {
	origCurrentBranch := currentBranch
	defer func() { currentBranch = origCurrentBranch }()
	currentBranch = func() (string, error) {
		return "my-feature", nil
	}

	finder := &mockPRFinder{prNumber: 77}

	result, err := Resolve("", "owner/repo", finder)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result.PRNumber != 77 {
		t.Errorf("expected PR number 77, got %d", result.PRNumber)
	}
}

func TestParseURL_ParsesValidPRURLs(t *testing.T) {
	tests := []struct {
		name   string
		url    string
		owner  string
		repo   string
		number int
		host   string
	}{
		{
			name:   "standard GitHub URL",
			url:    "https://github.com/cli/cli/pull/123",
			owner:  "cli",
			repo:   "cli",
			number: 123,
			host:   "github.com",
		},
		{
			name:   "GitHub Enterprise URL",
			url:    "https://github.example.com/org/project/pull/456",
			owner:  "org",
			repo:   "project",
			number: 456,
			host:   "github.example.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parseURL(tt.url)
			if err != nil {
				t.Fatalf("expected no error, got %v", err)
			}
			if result.Owner != tt.owner {
				t.Errorf("expected owner '%s', got '%s'", tt.owner, result.Owner)
			}
			if result.Repo != tt.repo {
				t.Errorf("expected repo '%s', got '%s'", tt.repo, result.Repo)
			}
			if result.PRNumber != tt.number {
				t.Errorf("expected number %d, got %d", tt.number, result.PRNumber)
			}
			if result.Host != tt.host {
				t.Errorf("expected host '%s', got '%s'", tt.host, result.Host)
			}
		})
	}
}

func TestParseURL_ReturnsErrorForInvalidURLs(t *testing.T) {
	tests := []struct {
		name string
		url  string
	}{
		{"not a URL", "just-a-string"},
		{"missing pull path", "https://github.com/owner/repo"},
		{"wrong path segment", "https://github.com/owner/repo/issues/123"},
		{"invalid PR number", "https://github.com/owner/repo/pull/abc"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := parseURL(tt.url)
			if err == nil {
				t.Fatal("expected error, got nil")
			}
		})
	}
}
