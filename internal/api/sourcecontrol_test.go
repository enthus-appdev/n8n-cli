package api

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type recordedRequest struct {
	Method string
	Path   string
	Query  string
	Body   string
	APIKey string
}

func newTestServer(t *testing.T, status int, body string) (*Client, *recordedRequest) {
	t.Helper()
	rec := &recordedRequest{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		*rec = recordedRequest{Method: r.Method, Path: r.URL.Path, Query: r.URL.RawQuery, Body: string(b), APIKey: r.Header.Get("X-N8N-API-KEY")}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(srv.Close)
	return NewClient(srv.URL, "test-key"), rec
}

const fileJSON = `{"file":"workflows/wf1.json","id":"wf1","name":"Invoice routing","type":"workflow","status":"modified","location":"local","conflict":false,"updatedAt":"2026-10-13T10:00:00.000Z"}`

func TestSourceControlStatus(t *testing.T) {
	client, rec := newTestServer(t, http.StatusOK, `{"data":[`+fileJSON+`]}`)

	files, err := client.SourceControlStatus("push")
	if err != nil {
		t.Fatalf("SourceControlStatus: %v", err)
	}
	if rec.Method != http.MethodGet || rec.Path != "/api/v1/source-control/status" || rec.Query != "direction=push" {
		t.Errorf("request = %s %s?%s", rec.Method, rec.Path, rec.Query)
	}
	if rec.APIKey != "test-key" {
		t.Errorf("API key header = %q", rec.APIKey)
	}
	if len(files) != 1 || files[0].ID != "wf1" || files[0].Status != "modified" {
		t.Errorf("files = %+v", files)
	}
}

func TestSourceControlPushBody(t *testing.T) {
	client, rec := newTestServer(t, http.StatusOK, `{"data":[`+fileJSON+`]}`)

	pushed, err := client.SourceControlPush(SourceControlPushOptions{
		CommitMessage: "Fix routing",
		FileNames:     []SourceControlItem{{ID: "wf1", Type: "workflow"}, {ID: "cred1", Type: "credential"}},
	})
	if err != nil {
		t.Fatalf("SourceControlPush: %v", err)
	}
	if rec.Method != http.MethodPost || rec.Path != "/api/v1/source-control/push" {
		t.Errorf("request = %s %s", rec.Method, rec.Path)
	}
	want := `{"commitMessage":"Fix routing","fileNames":[{"id":"wf1","type":"workflow"},{"id":"cred1","type":"credential"}]}`
	if rec.Body != want {
		t.Errorf("body = %s, want %s", rec.Body, want)
	}
	if len(pushed) != 1 {
		t.Errorf("pushed = %+v", pushed)
	}
}

func TestSourceControlPushForce(t *testing.T) {
	client, rec := newTestServer(t, http.StatusOK, `{"data":[]}`)

	if _, err := client.SourceControlPush(SourceControlPushOptions{
		CommitMessage: "m",
		FileNames:     []SourceControlItem{{ID: "wf1", Type: "workflow"}},
		Force:         true,
	}); err != nil {
		t.Fatalf("SourceControlPush: %v", err)
	}
	var body map[string]any
	if err := json.Unmarshal([]byte(rec.Body), &body); err != nil {
		t.Fatalf("body is not JSON: %v", err)
	}
	if body["force"] != true {
		t.Errorf("force = %v, want true", body["force"])
	}
}

func TestSourceControlPushConflict(t *testing.T) {
	conflictFile := strings.Replace(fileJSON, `"conflict":false`, `"conflict":true`, 1)
	client, _ := newTestServer(t, http.StatusConflict, `{"message":"Conflicts found","conflicts":[`+conflictFile+`]}`)

	_, err := client.SourceControlPush(SourceControlPushOptions{CommitMessage: "m", FileNames: []SourceControlItem{{ID: "wf1", Type: "workflow"}}})
	var conflict *SourceControlConflictError
	if !errors.As(err, &conflict) {
		t.Fatalf("err = %v, want SourceControlConflictError", err)
	}
	if conflict.Message != "Conflicts found" || len(conflict.Conflicts) != 1 || conflict.Conflicts[0].ID != "wf1" {
		t.Errorf("conflict = %+v", conflict)
	}
}

func TestSourceControlPullSendsEmptyObject(t *testing.T) {
	client, rec := newTestServer(t, http.StatusOK, `[`+fileJSON+`]`)

	files, err := client.SourceControlPull(SourceControlPullOptions{})
	if err != nil {
		t.Fatalf("SourceControlPull: %v", err)
	}
	if rec.Method != http.MethodPost || rec.Path != "/api/v1/source-control/pull" {
		t.Errorf("request = %s %s", rec.Method, rec.Path)
	}
	if rec.Body != `{}` {
		t.Errorf("body = %q, want {}", rec.Body)
	}
	if len(files) != 1 || files[0].Name != "Invoice routing" {
		t.Errorf("files = %+v", files)
	}
}

func TestSourceControlPullOptions(t *testing.T) {
	client, rec := newTestServer(t, http.StatusOK, `[]`)

	if _, err := client.SourceControlPull(SourceControlPullOptions{Force: true, AutoPublish: "published"}); err != nil {
		t.Fatalf("SourceControlPull: %v", err)
	}
	if want := `{"force":true,"autoPublish":"published"}`; rec.Body != want {
		t.Errorf("body = %s, want %s", rec.Body, want)
	}
}

func TestSourceControlPullConflict(t *testing.T) {
	unchanged := strings.Replace(fileJSON, `"status":"modified"`, `"status":"new"`, 1)
	unchanged = strings.Replace(unchanged, `"id":"wf1"`, `"id":"wf2"`, 1)
	flagged := strings.Replace(unchanged, `"id":"wf2"`, `"id":"wf3"`, 1)
	flagged = strings.Replace(flagged, `"conflict":false`, `"conflict":true`, 1)
	client, _ := newTestServer(t, http.StatusConflict, `[`+fileJSON+`,`+unchanged+`,`+flagged+`]`)

	_, err := client.SourceControlPull(SourceControlPullOptions{})
	var conflict *SourceControlConflictError
	if !errors.As(err, &conflict) {
		t.Fatalf("err = %v, want SourceControlConflictError", err)
	}
	var ids []string
	for _, f := range conflict.Conflicts {
		ids = append(ids, f.ID)
	}
	if strings.Join(ids, ",") != "wf1,wf3" {
		t.Errorf("conflict IDs = %v, want [wf1 wf3]", ids)
	}
}

func TestSourceControlForbiddenNamesScope(t *testing.T) {
	tests := []struct {
		name  string
		call  func(*Client) error
		scope string
	}{
		{"status", func(c *Client) error { _, err := c.SourceControlStatus("push"); return err }, "sourceControl:read"},
		{"push", func(c *Client) error {
			_, err := c.SourceControlPush(SourceControlPushOptions{CommitMessage: "m", FileNames: []SourceControlItem{{ID: "x", Type: "workflow"}}})
			return err
		}, "sourceControl:push"},
		{"pull", func(c *Client) error { _, err := c.SourceControlPull(SourceControlPullOptions{}); return err }, "sourceControl:pull"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, _ := newTestServer(t, http.StatusForbidden, `{"message":"Forbidden"}`)
			err := tt.call(client)
			if err == nil || !strings.Contains(err.Error(), tt.scope) || !strings.Contains(err.Error(), "API error (403): Forbidden") {
				t.Errorf("err = %v, want 403 with %s hint", err, tt.scope)
			}
			var apiErr *APIError
			if !errors.As(err, &apiErr) {
				t.Errorf("err does not wrap *APIError")
			}
		})
	}
}

func TestAPIErrorMessage(t *testing.T) {
	tests := []struct {
		body string
		want string
	}{
		{`{"message":"not found"}`, "API error (404): not found"},
		{`plain text`, "API error (404): plain text"},
		{`{"other":"x"}`, `API error (404): {"other":"x"}`},
	}
	for _, tt := range tests {
		client, _ := newTestServer(t, http.StatusNotFound, tt.body)
		_, err := client.GetWorkflow("x")
		if err == nil || err.Error() != tt.want {
			t.Errorf("body %s: err = %v, want %s", tt.body, err, tt.want)
		}
	}
}
