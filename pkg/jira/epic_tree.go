package jira

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

const epicTreeMaxResults uint = 100

// EpicTree represents the full issue hierarchy for an epic.
type EpicTree struct {
	Epic     *Issue           `json:"epic"`
	Children []*EpicTreeChild `json:"children"`
}

// EpicTreeChild holds a direct child issue and its subtasks.
type EpicTreeChild struct {
	Issue    *Issue   `json:"issue"`
	Subtasks []*Issue `json:"subtasks"`
}

// EpicTree fetches an epic and its direct child issues and subtasks.
func (c *Client) EpicTree(key string) (*EpicTree, error) {
	epic, err := c.GetIssueV2(key)
	if err != nil {
		return nil, err
	}

	children, err := c.searchV2All(fmt.Sprintf(`"Epic Link" = %s OR parent = %s ORDER BY created ASC`, key, key))
	if err != nil {
		return nil, err
	}

	tree := &EpicTree{Epic: epic, Children: make([]*EpicTreeChild, 0, len(children))}
	childIndex := make(map[string]*EpicTreeChild, len(children))
	subtaskParentKeys := make([]string, 0, len(children))

	for _, child := range children {
		treeChild := &EpicTreeChild{
			Issue:    child,
			Subtasks: dedupeIssues(child.Fields.Subtasks),
		}
		tree.Children = append(tree.Children, treeChild)
		childIndex[child.Key] = treeChild
		subtaskParentKeys = append(subtaskParentKeys, child.Key)
	}

	if len(subtaskParentKeys) == 0 {
		return tree, nil
	}

	for _, chunk := range chunkStrings(subtaskParentKeys, 50) {
		subtasks, err := c.searchV2All(fmt.Sprintf("parent in (%s) ORDER BY created ASC", strings.Join(chunk, ", ")))
		if err != nil {
			return nil, err
		}

		for _, subtask := range subtasks {
			if subtask.Fields.Parent == nil {
				continue
			}

			parent := childIndex[subtask.Fields.Parent.Key]
			if parent == nil {
				continue
			}

			parent.Subtasks = appendIfMissing(parent.Subtasks, subtask)
		}
	}

	return tree, nil
}

func (c *Client) searchV2All(jql string) ([]*Issue, error) {
	startAt := uint(0)
	issues := make([]*Issue, 0)

	for {
		res, err := c.searchV2Page(jql, startAt, epicTreeMaxResults)
		if err != nil {
			return nil, err
		}

		issues = append(issues, res.Issues...)

		next := res.StartAt + res.MaxResults
		if next >= res.Total || len(res.Issues) == 0 {
			break
		}

		startAt = next
	}

	return issues, nil
}

type searchV2PageResult struct {
	Issues     []*Issue `json:"issues"`
	StartAt    uint     `json:"startAt"`
	MaxResults uint     `json:"maxResults"`
	Total      uint     `json:"total"`
}

func (c *Client) searchV2Page(jql string, from, limit uint) (*searchV2PageResult, error) {
	path := fmt.Sprintf(
		"/search?jql=%s&startAt=%d&maxResults=%d&fields=*all",
		url.QueryEscape(jql),
		from,
		limit,
	)

	res, err := c.GetV2(context.Background(), path, nil)
	if err != nil {
		return nil, err
	}
	if res == nil {
		return nil, ErrEmptyResponse
	}
	defer func() { _ = res.Body.Close() }()

	if res.StatusCode != http.StatusOK {
		return nil, formatUnexpectedResponse(res)
	}

	var out searchV2PageResult
	if err := json.NewDecoder(res.Body).Decode(&out); err != nil {
		return nil, err
	}

	return &out, nil
}

func chunkStrings(items []string, size int) [][]string {
	if len(items) == 0 || size <= 0 {
		return nil
	}

	chunks := make([][]string, 0, (len(items)+size-1)/size)
	for i := 0; i < len(items); i += size {
		end := min(i+size, len(items))
		chunks = append(chunks, items[i:end])
	}

	return chunks
}

func dedupeIssues(issues []Issue) []*Issue {
	out := make([]*Issue, 0, len(issues))
	seen := make(map[string]struct{}, len(issues))

	for idx := range issues {
		if _, ok := seen[issues[idx].Key]; ok {
			continue
		}
		seen[issues[idx].Key] = struct{}{}
		issue := issues[idx]
		out = append(out, &issue)
	}

	return out
}

func appendIfMissing(issues []*Issue, issue *Issue) []*Issue {
	for _, existing := range issues {
		if existing != nil && issue != nil && existing.Key == issue.Key {
			return issues
		}
	}

	return append(issues, issue)
}
