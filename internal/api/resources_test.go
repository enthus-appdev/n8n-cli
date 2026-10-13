package api

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"testing"
)

type capturedRequest struct {
	Method string
	Path   string
	Query  url.Values
	Body   string
}

// newRecordingServer serves body with status for every request and records the last request.
func newRecordingServer(t *testing.T, status int, body string) (*Client, *capturedRequest) {
	t.Helper()
	rec := &capturedRequest{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		*rec = capturedRequest{Method: r.Method, Path: r.URL.Path, Query: r.URL.Query(), Body: string(b)}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(srv.Close)
	return NewClient(srv.URL, "test-key"), rec
}

func assertJSONEqual(t *testing.T, got, want string) {
	t.Helper()
	var g, w interface{}
	if err := json.Unmarshal([]byte(got), &g); err != nil {
		t.Fatalf("body %q is not JSON: %v", got, err)
	}
	if err := json.Unmarshal([]byte(want), &w); err != nil {
		t.Fatalf("want %q is not JSON: %v", want, err)
	}
	if !reflect.DeepEqual(g, w) {
		t.Errorf("body = %s, want %s", got, want)
	}
}

func TestGetWorkflowWithState(t *testing.T) {
	client, rec := newRecordingServer(t, http.StatusOK,
		`{"id":"wf1","name":"Router","active":true,"nodes":[],"connections":{},"versionId":"v2","activeVersionId":"v1","isArchived":false}`)

	wf, state, err := client.GetWorkflowWithState("wf1")
	if err != nil {
		t.Fatalf("GetWorkflowWithState: %v", err)
	}
	if rec.Path != "/api/v1/workflows/wf1" {
		t.Errorf("path = %s", rec.Path)
	}
	if wf.Name != "Router" || state.VersionID != "v2" || state.ActiveVersionID == nil || *state.ActiveVersionID != "v1" {
		t.Errorf("wf = %+v, state = %+v", wf, state)
	}
	if !state.HasUnpublishedChanges() {
		t.Error("HasUnpublishedChanges = false, want true")
	}
}

func TestHasUnpublishedChanges(t *testing.T) {
	v1 := "v1"
	tests := []struct {
		name  string
		state WorkflowState
		want  bool
	}{
		{"never published", WorkflowState{VersionID: "v1"}, false},
		{"published draft", WorkflowState{VersionID: "v1", ActiveVersionID: &v1}, false},
		{"draft ahead", WorkflowState{VersionID: "v2", ActiveVersionID: &v1}, true},
	}
	for _, tt := range tests {
		if got := tt.state.HasUnpublishedChanges(); got != tt.want {
			t.Errorf("%s: got %v, want %v", tt.name, got, tt.want)
		}
	}
}

func TestPublishWorkflowBody(t *testing.T) {
	tests := []struct {
		versionID string
		wantBody  string
	}{
		{"", `{}`},
		{"v7", `{"versionId":"v7"}`},
	}
	for _, tt := range tests {
		client, rec := newRecordingServer(t, http.StatusOK, `{"versionId":"v7","activeVersionId":"v7"}`)
		state, err := client.PublishWorkflow("wf1", tt.versionID)
		if err != nil {
			t.Fatalf("PublishWorkflow: %v", err)
		}
		if rec.Method != http.MethodPost || rec.Path != "/api/v1/workflows/wf1/publish" {
			t.Errorf("request = %s %s", rec.Method, rec.Path)
		}
		assertJSONEqual(t, rec.Body, tt.wantBody)
		if state.ActiveVersionID == nil || *state.ActiveVersionID != "v7" {
			t.Errorf("state = %+v", state)
		}
	}
}

func TestWorkflowLifecyclePaths(t *testing.T) {
	tests := []struct {
		call func(*Client) (*WorkflowState, error)
		path string
	}{
		{func(c *Client) (*WorkflowState, error) { return c.ArchiveWorkflow("wf1") }, "/api/v1/workflows/wf1/archive"},
		{func(c *Client) (*WorkflowState, error) { return c.UnarchiveWorkflow("wf1") }, "/api/v1/workflows/wf1/unarchive"},
		{func(c *Client) (*WorkflowState, error) { return c.UnpublishWorkflow("wf1") }, "/api/v1/workflows/wf1/unpublish"},
	}
	for _, tt := range tests {
		client, rec := newRecordingServer(t, http.StatusOK, `{"versionId":"v1","activeVersionId":null,"isArchived":true}`)
		if _, err := tt.call(client); err != nil {
			t.Fatalf("%s: %v", tt.path, err)
		}
		if rec.Method != http.MethodPost || rec.Path != tt.path {
			t.Errorf("request = %s %s, want POST %s", rec.Method, rec.Path, tt.path)
		}
	}
}

func TestListWorkflowHistory(t *testing.T) {
	client, rec := newRecordingServer(t, http.StatusOK,
		`{"data":[{"versionId":"v2","workflowId":"wf1","authors":"Jane","name":null,"description":null,"createdAt":"2026-10-13T10:00:00Z","updatedAt":"2026-10-13T10:00:00Z"}],"nextCursor":"c2"}`)

	result, err := client.ListWorkflowHistory("wf1", 20, "c1")
	if err != nil {
		t.Fatalf("ListWorkflowHistory: %v", err)
	}
	if rec.Path != "/api/v1/workflows/wf1/history" || rec.Query.Get("limit") != "20" || rec.Query.Get("cursor") != "c1" {
		t.Errorf("request = %s?%s", rec.Path, rec.Query.Encode())
	}
	if len(result.Data) != 1 || result.Data[0].VersionID != "v2" || result.NextCursor != "c2" {
		t.Errorf("result = %+v", result)
	}
}

func TestGetWorkflowVersionPath(t *testing.T) {
	client, rec := newRecordingServer(t, http.StatusOK, `{"versionId":"v2","workflowId":"wf1","authors":"Jane","nodes":[{"name":"Start"}],"connections":{}}`)

	v, err := client.GetWorkflowVersion("wf1", "v2")
	if err != nil {
		t.Fatalf("GetWorkflowVersion: %v", err)
	}
	if rec.Path != "/api/v1/workflows/wf1/versions/v2" {
		t.Errorf("path = %s", rec.Path)
	}
	if len(v.Nodes) != 1 {
		t.Errorf("nodes = %+v", v.Nodes)
	}
}

func TestStopExecutions(t *testing.T) {
	client, rec := newRecordingServer(t, http.StatusOK, `{"stopped":3}`)

	n, err := client.StopExecutions(StopExecutionsOptions{Status: []string{"running", "waiting"}, WorkflowID: "wf1"})
	if err != nil {
		t.Fatalf("StopExecutions: %v", err)
	}
	if rec.Path != "/api/v1/executions/stop" || n != 3 {
		t.Errorf("path = %s, stopped = %d", rec.Path, n)
	}
	assertJSONEqual(t, rec.Body, `{"status":["running","waiting"],"workflowId":"wf1"}`)
}

func TestStopExecution(t *testing.T) {
	client, rec := newRecordingServer(t, http.StatusOK, `{"mode":"trigger","status":"canceled","finished":false,"startedAt":"2026-10-13T10:00:00Z"}`)

	exec, err := client.StopExecution("42")
	if err != nil {
		t.Fatalf("StopExecution: %v", err)
	}
	if rec.Method != http.MethodPost || rec.Path != "/api/v1/executions/42/stop" || exec.Status != "canceled" {
		t.Errorf("request = %s %s, exec = %+v", rec.Method, rec.Path, exec)
	}
}

func TestTestCredential(t *testing.T) {
	client, rec := newRecordingServer(t, http.StatusOK, `{"status":"Error","message":"401 Unauthorized"}`)

	result, err := client.TestCredential("c1")
	if err != nil {
		t.Fatalf("TestCredential: %v", err)
	}
	if rec.Method != http.MethodPost || rec.Path != "/api/v1/credentials/c1/test" {
		t.Errorf("request = %s %s", rec.Method, rec.Path)
	}
	if result.Status != "Error" || result.Message != "401 Unauthorized" {
		t.Errorf("result = %+v", result)
	}
}

func TestListCredentials(t *testing.T) {
	client, rec := newRecordingServer(t, http.StatusOK,
		`{"data":[{"id":"c1","name":"SharePoint","type":"oAuth2Api","shared":[{"id":"p1","name":"DMS","role":"credential:owner"}]}],"nextCursor":null}`)

	result, err := client.ListCredentials(50, "")
	if err != nil {
		t.Fatalf("ListCredentials: %v", err)
	}
	if rec.Path != "/api/v1/credentials" || rec.Query.Get("limit") != "50" || rec.Query.Has("cursor") {
		t.Errorf("request = %s?%s", rec.Path, rec.Query.Encode())
	}
	if len(result.Data) != 1 || result.Data[0].Shared[0].Name != "DMS" || result.NextCursor != "" {
		t.Errorf("result = %+v", result)
	}
}

func TestListFolders(t *testing.T) {
	client, rec := newRecordingServer(t, http.StatusOK, `{"count":5,"data":[{"id":"f1","name":"Invoices","parentFolderId":null}]}`)

	folders, count, err := client.ListFolders("p1", 10, 1)
	if err != nil {
		t.Fatalf("ListFolders: %v", err)
	}
	if rec.Path != "/api/v1/projects/p1/folders" || rec.Query.Get("skip") != "10" || rec.Query.Get("take") != "1" {
		t.Errorf("request = %s?%s", rec.Path, rec.Query.Encode())
	}
	if count != 5 || len(folders) != 1 || folders[0].ParentFolderID != nil {
		t.Errorf("count = %d, folders = %+v", count, folders)
	}
}

func TestUpdateFolderSendsOnlySetFields(t *testing.T) {
	client, rec := newRecordingServer(t, http.StatusOK, `{"id":"f1","name":"New"}`)

	if _, err := client.UpdateFolder("p1", "f1", "New", ""); err != nil {
		t.Fatalf("UpdateFolder: %v", err)
	}
	if rec.Method != http.MethodPatch || rec.Path != "/api/v1/projects/p1/folders/f1" {
		t.Errorf("request = %s %s", rec.Method, rec.Path)
	}
	assertJSONEqual(t, rec.Body, `{"name":"New"}`)
}

func TestDeleteFolderTransfer(t *testing.T) {
	client, rec := newRecordingServer(t, http.StatusNoContent, ``)

	if err := client.DeleteFolder("p1", "f1", "f2"); err != nil {
		t.Fatalf("DeleteFolder: %v", err)
	}
	if rec.Method != http.MethodDelete || rec.Path != "/api/v1/projects/p1/folders/f1" || rec.Query.Get("transferToFolderId") != "f2" {
		t.Errorf("request = %s %s?%s", rec.Method, rec.Path, rec.Query.Encode())
	}
}

func TestListUsers(t *testing.T) {
	client, rec := newRecordingServer(t, http.StatusOK, `{"data":[{"id":"u1","email":"a@example.com","role":"global:member"}],"nextCursor":null}`)

	result, err := client.ListUsers(10, "", "p1")
	if err != nil {
		t.Fatalf("ListUsers: %v", err)
	}
	if rec.Query.Get("includeRole") != "true" || rec.Query.Get("projectId") != "p1" || rec.Query.Get("limit") != "10" {
		t.Errorf("query = %s", rec.Query.Encode())
	}
	if result.Data[0].Role != "global:member" {
		t.Errorf("result = %+v", result)
	}
}

func TestGetUserByEmail(t *testing.T) {
	client, rec := newRecordingServer(t, http.StatusOK, `{"id":"u1","email":"a@example.com"}`)

	if _, err := client.GetUser("a@example.com"); err != nil {
		t.Fatalf("GetUser: %v", err)
	}
	if rec.Path != "/api/v1/users/a@example.com" || rec.Query.Get("includeRole") != "true" {
		t.Errorf("request = %s?%s", rec.Path, rec.Query.Encode())
	}
}

func TestProjectMembers(t *testing.T) {
	client, rec := newRecordingServer(t, http.StatusCreated, ``)
	if err := client.AddProjectMembers("p1", []ProjectRelation{{UserID: "u1", Role: "project:editor"}}); err != nil {
		t.Fatalf("AddProjectMembers: %v", err)
	}
	if rec.Method != http.MethodPost || rec.Path != "/api/v1/projects/p1/users" {
		t.Errorf("request = %s %s", rec.Method, rec.Path)
	}
	assertJSONEqual(t, rec.Body, `{"relations":[{"userId":"u1","role":"project:editor"}]}`)

	client, rec = newRecordingServer(t, http.StatusNoContent, ``)
	if err := client.SetProjectMemberRole("p1", "u1", "project:viewer"); err != nil {
		t.Fatalf("SetProjectMemberRole: %v", err)
	}
	if rec.Method != http.MethodPatch || rec.Path != "/api/v1/projects/p1/users/u1" {
		t.Errorf("request = %s %s", rec.Method, rec.Path)
	}
	assertJSONEqual(t, rec.Body, `{"role":"project:viewer"}`)

	client, rec = newRecordingServer(t, http.StatusNoContent, ``)
	if err := client.RemoveProjectMember("p1", "u1"); err != nil {
		t.Fatalf("RemoveProjectMember: %v", err)
	}
	if rec.Method != http.MethodDelete || rec.Path != "/api/v1/projects/p1/users/u1" {
		t.Errorf("request = %s %s", rec.Method, rec.Path)
	}
}

func TestGenerateAuditBody(t *testing.T) {
	tests := []struct {
		opts AuditOptions
		want string
	}{
		{AuditOptions{}, `{}`},
		{AuditOptions{Categories: []string{"credentials"}, DaysAbandonedWorkflow: 30},
			`{"additionalOptions":{"categories":["credentials"],"daysAbandonedWorkflow":30}}`},
	}
	for _, tt := range tests {
		client, rec := newRecordingServer(t, http.StatusOK, `{"Credentials Risk Report":{"risk":"credentials"}}`)
		report, err := client.GenerateAudit(tt.opts)
		if err != nil {
			t.Fatalf("GenerateAudit: %v", err)
		}
		if rec.Path != "/api/v1/audit" {
			t.Errorf("path = %s", rec.Path)
		}
		assertJSONEqual(t, rec.Body, tt.want)
		if _, ok := report["Credentials Risk Report"]; !ok {
			t.Errorf("report = %+v", report)
		}
	}
}
