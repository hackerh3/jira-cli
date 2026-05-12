package jira

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestEpicTree(t *testing.T) {
	call := 0

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch call {
		case 0:
			assert.Equal(t, "/rest/api/2/issue/EPIC-1", r.URL.Path)
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{
				"key":"EPIC-1",
				"fields":{
					"summary":"Epic summary",
					"issuetype":{"name":"Epic"},
					"status":{"name":"In Progress"}
				}
			}`))
		case 1:
			assert.Equal(t, "/rest/api/2/search", r.URL.Path)
			assert.Equal(t, url.Values{
				"fields":     []string{"*all"},
				"jql":        []string{`"Epic Link" = EPIC-1 OR parent = EPIC-1 ORDER BY created ASC`},
				"maxResults": []string{"100"},
				"startAt":    []string{"0"},
			}, r.URL.Query())
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{
				"startAt":0,
				"maxResults":100,
				"total":2,
				"issues":[
					{
						"key":"STORY-1",
						"fields":{
							"summary":"Story one",
							"issuetype":{"name":"Story"},
							"status":{"name":"To Do"},
							"subtasks":[
								{
									"key":"SUB-1",
									"fields":{
										"summary":"Subtask one",
										"issuetype":{"name":"Sub-task"},
										"status":{"name":"Done"}
									}
								}
							]
						}
					},
					{
						"key":"TASK-1",
						"fields":{
							"summary":"Task one",
							"issuetype":{"name":"Task"},
							"status":{"name":"In Progress"}
						}
					}
				]
			}`))
		case 2:
			assert.Equal(t, "/rest/api/2/search", r.URL.Path)
			assert.Equal(t, url.Values{
				"fields":     []string{"*all"},
				"jql":        []string{"parent in (STORY-1, TASK-1) ORDER BY created ASC"},
				"maxResults": []string{"100"},
				"startAt":    []string{"0"},
			}, r.URL.Query())
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{
				"startAt":0,
				"maxResults":100,
				"total":2,
				"issues":[
					{
						"key":"SUB-1",
						"fields":{
							"summary":"Subtask one",
							"issuetype":{"name":"Sub-task"},
							"status":{"name":"Done"},
							"parent":{"key":"STORY-1"}
						}
					},
					{
						"key":"SUB-2",
						"fields":{
							"summary":"Subtask two",
							"issuetype":{"name":"Sub-task"},
							"status":{"name":"To Do"},
							"parent":{"key":"STORY-1"}
						}
					}
				]
			}`))
		default:
			t.Fatalf("unexpected request %d: %s", call, r.URL.String())
		}

		call++
	}))
	defer server.Close()

	client := NewClient(Config{Server: server.URL}, WithTimeout(3*time.Second))

	actual, err := client.EpicTree("EPIC-1")
	if !assert.NoError(t, err) {
		return
	}

	if assert.NotNil(t, actual) && assert.NotNil(t, actual.Epic) {
		assert.Equal(t, "EPIC-1", actual.Epic.Key)
		assert.Len(t, actual.Children, 2)
		assert.Equal(t, "STORY-1", actual.Children[0].Issue.Key)
		assert.Len(t, actual.Children[0].Subtasks, 2)
		assert.Equal(t, "SUB-1", actual.Children[0].Subtasks[0].Key)
		assert.Equal(t, "SUB-2", actual.Children[0].Subtasks[1].Key)
		assert.Equal(t, "TASK-1", actual.Children[1].Issue.Key)
		assert.Empty(t, actual.Children[1].Subtasks)
	}

	assert.Equal(t, 3, call)
}

func TestEpicTreeUnexpectedResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/rest/api/2/issue/EPIC-1", r.URL.Path)
		w.WriteHeader(http.StatusBadRequest)
	}))
	defer server.Close()

	client := NewClient(Config{Server: server.URL}, WithTimeout(3*time.Second))

	_, err := client.EpicTree("EPIC-1")
	assert.Error(t, &ErrUnexpectedResponse{}, err)
}

func TestEpicTreePagination(t *testing.T) {
	call := 0

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch call {
		case 0:
			assert.Equal(t, "/rest/api/2/issue/EPIC-9", r.URL.Path)
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"key":"EPIC-9","fields":{"summary":"Epic","issuetype":{"name":"Epic"},"status":{"name":"To Do"}}}`))
		case 1:
			assert.Equal(t, "/rest/api/2/search", r.URL.Path)
			assert.Equal(t, "0", r.URL.Query().Get("startAt"))
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"startAt":0,"maxResults":100,"total":101,"issues":[{"key":"STORY-1","fields":{"summary":"Story 1","issuetype":{"name":"Story"},"status":{"name":"To Do"}}}]}`))
		case 2:
			assert.Equal(t, "/rest/api/2/search", r.URL.Path)
			assert.Equal(t, "100", r.URL.Query().Get("startAt"))
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"startAt":100,"maxResults":100,"total":101,"issues":[{"key":"STORY-2","fields":{"summary":"Story 2","issuetype":{"name":"Story"},"status":{"name":"Done"}}}]}`))
		case 3:
			assert.Equal(t, "/rest/api/2/search", r.URL.Path)
			assert.Equal(t, url.Values{
				"fields":     []string{"*all"},
				"jql":        []string{"parent in (STORY-1, STORY-2) ORDER BY created ASC"},
				"maxResults": []string{"100"},
				"startAt":    []string{"0"},
			}, r.URL.Query())
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"startAt":0,"maxResults":100,"total":0,"issues":[]}`))
		default:
			t.Fatalf("unexpected request %d: %s", call, r.URL.String())
		}

		call++
	}))
	defer server.Close()

	client := NewClient(Config{Server: server.URL}, WithTimeout(3*time.Second))

	tree, err := client.EpicTree("EPIC-9")
	if !assert.NoError(t, err) {
		return
	}

	assert.Len(t, tree.Children, 2)
	assert.Equal(t, "STORY-1", tree.Children[0].Issue.Key)
	assert.Equal(t, "STORY-2", tree.Children[1].Issue.Key)
	assert.Equal(t, 4, call)
}
