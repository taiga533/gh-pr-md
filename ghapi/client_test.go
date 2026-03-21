package ghapi

import (
	"encoding/json"
	"fmt"
	"testing"
)

type mockGraphQLClient struct {
	responses []string
	callIndex int
}

func (m *mockGraphQLClient) Do(query string, variables map[string]interface{}, response interface{}) error {
	if m.callIndex >= len(m.responses) {
		return fmt.Errorf("unexpected call index %d", m.callIndex)
	}
	resp := m.responses[m.callIndex]
	m.callIndex++
	return json.Unmarshal([]byte(resp), response)
}

func TestFetchPR_FetchesPRDataCorrectly(t *testing.T) {
	mockResp := `{
		"repository": {
			"pullRequest": {
				"number": 42,
				"title": "Add feature X",
				"body": "This PR adds feature X",
				"author": {"login": "alice"},
				"assignees": {"nodes": [{"login": "bob"}]},
				"comments": {
					"pageInfo": {"hasNextPage": false, "endCursor": ""},
					"nodes": [
						{
							"author": {"login": "charlie"},
							"body": "Looks good!",
							"createdAt": "2024-01-15T10:00:00Z"
						}
					]
				},
				"reviews": {
					"pageInfo": {"hasNextPage": false, "endCursor": ""},
					"nodes": [
						{
							"author": {"login": "dave"},
							"body": "LGTM",
							"state": "APPROVED",
							"submittedAt": "2024-01-15T11:00:00Z",
							"comments": {
								"nodes": [
									{
										"author": {"login": "dave"},
										"body": "Nice implementation",
										"path": "main.go",
										"diffHunk": "@@ -1,3 +1,5 @@\n+func hello() {}",
										"createdAt": "2024-01-15T11:00:00Z",
										"commit": {"oid": "abc123def"},
										"originalLine": 5,
										"originalStartLine": 1
									}
								]
							}
						}
					]
				}
			}
		}
	}`

	mock := &mockGraphQLClient{responses: []string{mockResp}}
	client := NewClientWithGraphQL(mock)

	pr, err := client.FetchPR("owner", "repo", 42)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if pr.Number != 42 {
		t.Errorf("expected number 42, got %d", pr.Number)
	}
	if pr.Title != "Add feature X" {
		t.Errorf("expected title 'Add feature X', got '%s'", pr.Title)
	}
	if pr.Body != "This PR adds feature X" {
		t.Errorf("expected body 'This PR adds feature X', got '%s'", pr.Body)
	}
	if pr.Author.Login != "alice" {
		t.Errorf("expected author 'alice', got '%s'", pr.Author.Login)
	}
	if len(pr.Assignees) != 1 || pr.Assignees[0].Login != "bob" {
		t.Errorf("expected assignee 'bob', got %v", pr.Assignees)
	}
	if len(pr.Comments) != 1 {
		t.Fatalf("expected 1 comment, got %d", len(pr.Comments))
	}
	if pr.Comments[0].Author.Login != "charlie" {
		t.Errorf("expected comment author 'charlie', got '%s'", pr.Comments[0].Author.Login)
	}
	if len(pr.Reviews) != 1 {
		t.Fatalf("expected 1 review, got %d", len(pr.Reviews))
	}
	if pr.Reviews[0].State != "APPROVED" {
		t.Errorf("expected review state 'APPROVED', got '%s'", pr.Reviews[0].State)
	}
	if len(pr.Reviews[0].Comments) != 1 {
		t.Fatalf("expected 1 review comment, got %d", len(pr.Reviews[0].Comments))
	}
	rc := pr.Reviews[0].Comments[0]
	if rc.Path != "main.go" {
		t.Errorf("expected path 'main.go', got '%s'", rc.Path)
	}
	if rc.CommitHash != "abc123def" {
		t.Errorf("expected commit hash 'abc123def', got '%s'", rc.CommitHash)
	}
}

func TestFetchPR_CollectsAllCommentsViaPagination(t *testing.T) {
	firstPage := `{
		"repository": {
			"pullRequest": {
				"number": 1,
				"title": "Test PR",
				"body": "",
				"author": {"login": "alice"},
				"assignees": {"nodes": []},
				"comments": {
					"pageInfo": {"hasNextPage": true, "endCursor": "cursor1"},
					"nodes": [
						{"author": {"login": "user1"}, "body": "comment 1", "createdAt": "2024-01-01T00:00:00Z"}
					]
				},
				"reviews": {
					"pageInfo": {"hasNextPage": false, "endCursor": ""},
					"nodes": []
				}
			}
		}
	}`

	secondPage := `{
		"repository": {
			"pullRequest": {
				"number": 1,
				"title": "Test PR",
				"body": "",
				"author": {"login": "alice"},
				"assignees": {"nodes": []},
				"comments": {
					"pageInfo": {"hasNextPage": false, "endCursor": ""},
					"nodes": [
						{"author": {"login": "user2"}, "body": "comment 2", "createdAt": "2024-01-02T00:00:00Z"}
					]
				},
				"reviews": {
					"pageInfo": {"hasNextPage": false, "endCursor": ""},
					"nodes": []
				}
			}
		}
	}`

	mock := &mockGraphQLClient{responses: []string{firstPage, secondPage}}
	client := NewClientWithGraphQL(mock)

	pr, err := client.FetchPR("owner", "repo", 1)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(pr.Comments) != 2 {
		t.Fatalf("expected 2 comments after pagination, got %d", len(pr.Comments))
	}
	if pr.Comments[0].Body != "comment 1" {
		t.Errorf("expected first comment 'comment 1', got '%s'", pr.Comments[0].Body)
	}
	if pr.Comments[1].Body != "comment 2" {
		t.Errorf("expected second comment 'comment 2', got '%s'", pr.Comments[1].Body)
	}
}

func TestFindPRByBranch_PrefersOpenPRs(t *testing.T) {
	mockResp := `{
		"repository": {
			"pullRequests": {
				"nodes": [{"number": 55}]
			}
		}
	}`

	mock := &mockGraphQLClient{responses: []string{mockResp}}
	client := NewClientWithGraphQL(mock)

	number, err := client.FindPRByBranch("owner", "repo", "feature")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if number != 55 {
		t.Errorf("expected PR number 55, got %d", number)
	}
}

func TestFindPRByBranch_FallsBackToClosedPRs(t *testing.T) {
	emptyResp := `{"repository": {"pullRequests": {"nodes": []}}}`
	closedResp := `{"repository": {"pullRequests": {"nodes": [{"number": 33}]}}}`

	mock := &mockGraphQLClient{responses: []string{emptyResp, closedResp}}
	client := NewClientWithGraphQL(mock)

	number, err := client.FindPRByBranch("owner", "repo", "old-branch")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if number != 33 {
		t.Errorf("expected PR number 33, got %d", number)
	}
}

func TestFindPRByBranch_ReturnsErrorWhenNoPRFound(t *testing.T) {
	emptyResp := `{"repository": {"pullRequests": {"nodes": []}}}`

	mock := &mockGraphQLClient{responses: []string{emptyResp, emptyResp}}
	client := NewClientWithGraphQL(mock)

	_, err := client.FindPRByBranch("owner", "repo", "nonexistent")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}
