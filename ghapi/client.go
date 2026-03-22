package ghapi

import (
	"fmt"
	"time"

	"github.com/cli/go-gh/v2/pkg/api"
)

// Client defines the interface for fetching PR data from GitHub.
type Client interface {
	FetchPR(owner, repo string, number int) (*PRData, error)
	FindPRByBranch(owner, repo, branch string) (int, error)
}

// GraphQLClient provides methods to interact with the GitHub GraphQL API.
type GraphQLClient interface {
	Do(query string, variables map[string]interface{}, response interface{}) error
}

// ghClient implements Client using the GitHub GraphQL API.
type ghClient struct {
	gql GraphQLClient
}

// NewClient creates a new GitHub API client using the default GraphQL client.
func NewClient() (Client, error) {
	gql, err := api.DefaultGraphQLClient()
	if err != nil {
		return nil, fmt.Errorf("failed to create GraphQL client: %w", err)
	}
	return &ghClient{gql: gql}, nil
}

// NewClientWithGraphQL creates a new GitHub API client with a custom GraphQL client.
func NewClientWithGraphQL(gql GraphQLClient) Client {
	return &ghClient{gql: gql}
}

const fetchPRQuery = `
query($owner: String!, $repo: String!, $number: Int!, $commentsCursor: String, $reviewsCursor: String) {
  repository(owner: $owner, name: $repo) {
    pullRequest(number: $number) {
      number
      title
      body
      author { login }
      assignees(first: 20) {
        nodes { login }
      }
      comments(first: 100, after: $commentsCursor) {
        pageInfo { hasNextPage endCursor }
        nodes {
          author { login }
          body
          createdAt
        }
      }
      reviews(first: 100, after: $reviewsCursor) {
        pageInfo { hasNextPage endCursor }
        nodes {
          author { login }
          body
          state
          submittedAt
          comments(first: 100) {
            nodes {
              id
              replyTo { id }
              author { login }
              body
              path
              diffHunk
              createdAt
              commit { oid }
              originalLine
              originalStartLine
            }
          }
        }
      }
    }
  }
}
`

type graphQLResponse struct {
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
				PageInfo pageInfo `json:"pageInfo"`
				Nodes    []struct {
					Author struct {
						Login string `json:"login"`
					} `json:"author"`
					Body      string `json:"body"`
					CreatedAt string `json:"createdAt"`
				} `json:"nodes"`
			} `json:"comments"`
			Reviews struct {
				PageInfo pageInfo `json:"pageInfo"`
				Nodes    []struct {
					Author struct {
						Login string `json:"login"`
					} `json:"author"`
					Body        string `json:"body"`
					State       string `json:"state"`
					SubmittedAt string `json:"submittedAt"`
					Comments    struct {
						Nodes []struct {
							ID      string `json:"id"`
							ReplyTo *struct {
								ID string `json:"id"`
							} `json:"replyTo"`
							Author struct {
								Login string `json:"login"`
							} `json:"author"`
							Body              string `json:"body"`
							Path              string `json:"path"`
							DiffHunk          string `json:"diffHunk"`
							CreatedAt         string `json:"createdAt"`
							Commit            struct {
								OID string `json:"oid"`
							} `json:"commit"`
							OriginalLine      int `json:"originalLine"`
							OriginalStartLine int `json:"originalStartLine"`
						} `json:"nodes"`
					} `json:"comments"`
				} `json:"nodes"`
			} `json:"reviews"`
		} `json:"pullRequest"`
	} `json:"repository"`
}

type pageInfo struct {
	HasNextPage bool   `json:"hasNextPage"`
	EndCursor   string `json:"endCursor"`
}

// FetchPR fetches complete PR data including comments, reviews, and inline comments.
func (c *ghClient) FetchPR(owner, repo string, number int) (*PRData, error) {
	pr := &PRData{}
	var commentsCursor *string
	var reviewsCursor *string
	commentsDone := false
	reviewsDone := false

	for {
		variables := map[string]interface{}{
			"owner":  owner,
			"repo":   repo,
			"number": number,
		}
		if commentsCursor != nil {
			variables["commentsCursor"] = *commentsCursor
		}
		if reviewsCursor != nil {
			variables["reviewsCursor"] = *reviewsCursor
		}

		var resp graphQLResponse
		if err := c.gql.Do(fetchPRQuery, variables, &resp); err != nil {
			return nil, fmt.Errorf("GraphQL query failed: %w", err)
		}

		prData := resp.Repository.PullRequest

		// Set metadata on first page
		if commentsCursor == nil && reviewsCursor == nil {
			pr.Number = prData.Number
			pr.Title = prData.Title
			pr.Body = prData.Body
			pr.Author = User{Login: prData.Author.Login}
			for _, a := range prData.Assignees.Nodes {
				pr.Assignees = append(pr.Assignees, User{Login: a.Login})
			}
		}

		// Collect comments only if not yet fully paginated.
		if !commentsDone {
			for _, c := range prData.Comments.Nodes {
				createdAt, err := time.Parse(time.RFC3339, c.CreatedAt)
				if err != nil {
					return nil, fmt.Errorf("failed to parse comment timestamp %q: %w", c.CreatedAt, err)
				}
				pr.Comments = append(pr.Comments, IssueComment{
					Author:    User{Login: c.Author.Login},
					Body:      c.Body,
					CreatedAt: createdAt,
				})
			}
		}

		// Collect reviews only if not yet fully paginated.
		if !reviewsDone {
			for _, r := range prData.Reviews.Nodes {
				submittedAt, err := time.Parse(time.RFC3339, r.SubmittedAt)
				if err != nil {
					return nil, fmt.Errorf("failed to parse review timestamp %q: %w", r.SubmittedAt, err)
				}
				review := Review{
					Author:      User{Login: r.Author.Login},
					Body:        r.Body,
					State:       r.State,
					SubmittedAt: submittedAt,
				}
				for _, rc := range r.Comments.Nodes {
					createdAt, err := time.Parse(time.RFC3339, rc.CreatedAt)
					if err != nil {
						return nil, fmt.Errorf("failed to parse review comment timestamp %q: %w", rc.CreatedAt, err)
					}
					var replyToID string
					if rc.ReplyTo != nil {
						replyToID = rc.ReplyTo.ID
					}
					review.Comments = append(review.Comments, ReviewComment{
						ID:                rc.ID,
						ReplyToID:         replyToID,
						Author:            User{Login: rc.Author.Login},
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
		}

		// Check pagination and mark completed resources.
		if !prData.Comments.PageInfo.HasNextPage {
			commentsDone = true
		} else {
			cursor := prData.Comments.PageInfo.EndCursor
			commentsCursor = &cursor
		}

		if !prData.Reviews.PageInfo.HasNextPage {
			reviewsDone = true
		} else {
			cursor := prData.Reviews.PageInfo.EndCursor
			reviewsCursor = &cursor
		}

		if commentsDone && reviewsDone {
			break
		}
	}

	return pr, nil
}

const findPRByBranchQuery = `
query($owner: String!, $repo: String!, $head: String!, $states: [PullRequestState!]) {
  repository(owner: $owner, name: $repo) {
    pullRequests(headRefName: $head, states: $states, first: 1, orderBy: {field: UPDATED_AT, direction: DESC}) {
      nodes { number }
    }
  }
}
`

type findPRResponse struct {
	Repository struct {
		PullRequests struct {
			Nodes []struct {
				Number int `json:"number"`
			} `json:"nodes"`
		} `json:"pullRequests"`
	} `json:"repository"`
}

// FindPRByBranch finds a PR number by branch name, preferring open PRs.
func (c *ghClient) FindPRByBranch(owner, repo, branch string) (int, error) {
	// Try open PRs first
	variables := map[string]interface{}{
		"owner":  owner,
		"repo":   repo,
		"head":   branch,
		"states": []string{"OPEN"},
	}

	var resp findPRResponse
	if err := c.gql.Do(findPRByBranchQuery, variables, &resp); err != nil {
		return 0, fmt.Errorf("GraphQL query failed: %w", err)
	}

	if len(resp.Repository.PullRequests.Nodes) > 0 {
		return resp.Repository.PullRequests.Nodes[0].Number, nil
	}

	// Try closed/merged PRs
	variables["states"] = []string{"CLOSED", "MERGED"}
	if err := c.gql.Do(findPRByBranchQuery, variables, &resp); err != nil {
		return 0, fmt.Errorf("GraphQL query failed: %w", err)
	}

	if len(resp.Repository.PullRequests.Nodes) > 0 {
		return resp.Repository.PullRequests.Nodes[0].Number, nil
	}

	return 0, fmt.Errorf("no pull request found for branch %q", branch)
}
