package resolver

import (
	"fmt"
	"net/url"
	"os/exec"
	"strconv"
	"strings"

	"github.com/cli/go-gh/v2/pkg/repository"
)

// PRFinder finds a PR number by branch name.
type PRFinder interface {
	FindPRByBranch(owner, repo, branch string) (int, error)
}

// RepoInfo holds the resolved repository information.
type RepoInfo struct {
	Owner string
	Repo  string
	Host  string
}

// Result holds the resolved PR target information.
type Result struct {
	RepoInfo
	PRNumber int
}

// Resolve resolves the CLI argument and repo flag into a Result containing
// owner, repo, host, and PR number.
func Resolve(arg string, repoFlag string, finder PRFinder) (*Result, error) {
	if arg != "" {
		// Try PR number
		if n, err := strconv.Atoi(arg); err == nil {
			repo, err := resolveRepo(repoFlag)
			if err != nil {
				return nil, err
			}
			return &Result{RepoInfo: *repo, PRNumber: n}, nil
		}

		// Try URL
		if result, err := parseURL(arg); err == nil {
			return result, nil
		}

		// Treat as branch name
		repo, err := resolveRepo(repoFlag)
		if err != nil {
			return nil, err
		}
		prNumber, err := finder.FindPRByBranch(repo.Owner, repo.Repo, arg)
		if err != nil {
			return nil, fmt.Errorf("failed to find PR for branch %q: %w", arg, err)
		}
		return &Result{RepoInfo: *repo, PRNumber: prNumber}, nil
	}

	// No argument: auto-detect from current branch
	repo, err := resolveRepo(repoFlag)
	if err != nil {
		return nil, err
	}
	branch, err := currentBranch()
	if err != nil {
		return nil, fmt.Errorf("failed to detect current branch: %w", err)
	}
	prNumber, err := finder.FindPRByBranch(repo.Owner, repo.Repo, branch)
	if err != nil {
		return nil, fmt.Errorf("failed to find PR for branch %q: %w", branch, err)
	}
	return &Result{RepoInfo: *repo, PRNumber: prNumber}, nil
}

// parseURL extracts owner, repo, and PR number from a GitHub pull request URL.
func parseURL(rawURL string) (*Result, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return nil, err
	}
	if u.Scheme == "" || u.Host == "" {
		return nil, fmt.Errorf("not a valid URL: %s", rawURL)
	}

	// Expected path: /OWNER/REPO/pull/NUMBER
	parts := strings.Split(strings.Trim(u.Path, "/"), "/")
	if len(parts) < 4 || parts[2] != "pull" {
		return nil, fmt.Errorf("not a valid pull request URL: %s", rawURL)
	}

	number, err := strconv.Atoi(parts[3])
	if err != nil {
		return nil, fmt.Errorf("invalid PR number in URL: %s", parts[3])
	}

	return &Result{
		RepoInfo: RepoInfo{
			Owner: parts[0],
			Repo:  parts[1],
			Host:  u.Host,
		},
		PRNumber: number,
	}, nil
}

// resolveRepo resolves the repository from the -R flag or current git context.
func resolveRepo(repoFlag string) (*RepoInfo, error) {
	if repoFlag != "" {
		r, err := repository.Parse(repoFlag)
		if err != nil {
			return nil, fmt.Errorf("failed to parse repo %q: %w", repoFlag, err)
		}
		return &RepoInfo{Owner: r.Owner, Repo: r.Name, Host: r.Host}, nil
	}

	r, err := repository.Current()
	if err != nil {
		return nil, fmt.Errorf("failed to detect repository: %w", err)
	}
	return &RepoInfo{Owner: r.Owner, Repo: r.Name, Host: r.Host}, nil
}

// currentBranch returns the name of the current git branch.
var currentBranch = func() (string, error) {
	out, err := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD").Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}
