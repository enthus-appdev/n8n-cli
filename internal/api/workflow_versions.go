package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

// WorkflowState holds the draft/published version fields of a workflow. It is
// kept out of Workflow so pulled workflow files do not change on every save.
type WorkflowState struct {
	VersionID       string  `json:"versionId,omitempty"`
	ActiveVersionID *string `json:"activeVersionId"`
	IsArchived      bool    `json:"isArchived"`
}

// HasUnpublishedChanges reports whether the draft differs from the published version.
func (s *WorkflowState) HasUnpublishedChanges() bool {
	return s.ActiveVersionID != nil && *s.ActiveVersionID != s.VersionID
}

// GetWorkflowWithState returns a workflow together with its version state.
func (c *Client) GetWorkflowWithState(id string) (*Workflow, *WorkflowState, error) {
	respBody, err := c.request(http.MethodGet, "/workflows/"+url.PathEscape(id), nil)
	if err != nil {
		return nil, nil, err
	}

	var wf Workflow
	var state WorkflowState
	if err := json.Unmarshal(respBody, &wf); err != nil {
		return nil, nil, fmt.Errorf("failed to parse response: %w", err)
	}
	if err := json.Unmarshal(respBody, &state); err != nil {
		return nil, nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &wf, &state, nil
}

func (c *Client) workflowAction(id, action string, body interface{}) (*WorkflowState, error) {
	respBody, err := c.request(http.MethodPost, "/workflows/"+url.PathEscape(id)+"/"+action, body)
	if err != nil {
		return nil, err
	}

	var state WorkflowState
	if err := json.Unmarshal(respBody, &state); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &state, nil
}

// ArchiveWorkflow archives a workflow. Archived workflows are deactivated and hidden from lists.
func (c *Client) ArchiveWorkflow(id string) (*WorkflowState, error) {
	return c.workflowAction(id, "archive", nil)
}

// UnarchiveWorkflow restores an archived workflow.
func (c *Client) UnarchiveWorkflow(id string) (*WorkflowState, error) {
	return c.workflowAction(id, "unarchive", nil)
}

// PublishWorkflow publishes a workflow version; an empty versionID publishes the current draft.
func (c *Client) PublishWorkflow(id, versionID string) (*WorkflowState, error) {
	body := map[string]string{}
	if versionID != "" {
		body["versionId"] = versionID
	}
	return c.workflowAction(id, "publish", body)
}

// UnpublishWorkflow removes the published version so the workflow no longer runs.
func (c *Client) UnpublishWorkflow(id string) (*WorkflowState, error) {
	return c.workflowAction(id, "unpublish", nil)
}

// WorkflowVersion is an entry in a workflow's version history.
type WorkflowVersion struct {
	VersionID   string                   `json:"versionId"`
	WorkflowID  string                   `json:"workflowId"`
	Authors     string                   `json:"authors"`
	Name        *string                  `json:"name"`
	Description *string                  `json:"description"`
	Nodes       []map[string]interface{} `json:"nodes,omitempty"`
	Connections map[string]interface{}   `json:"connections,omitempty"`
	CreatedAt   *time.Time               `json:"createdAt,omitempty"`
	UpdatedAt   *time.Time               `json:"updatedAt,omitempty"`
}

// ListWorkflowHistory returns one page of a workflow's version history, newest first.
func (c *Client) ListWorkflowHistory(id string, limit int, cursor string) (*ListResult[WorkflowVersion], error) {
	params := url.Values{}
	if limit > 0 {
		params.Set("limit", strconv.Itoa(limit))
	}
	if cursor != "" {
		params.Set("cursor", cursor)
	}

	path := "/workflows/" + url.PathEscape(id) + "/history"
	if len(params) > 0 {
		path += "?" + params.Encode()
	}

	return getJSON[ListResult[WorkflowVersion]](c, path)
}

// GetWorkflowVersion returns one version of a workflow, including its nodes and connections.
func (c *Client) GetWorkflowVersion(id, versionID string) (*WorkflowVersion, error) {
	return getJSON[WorkflowVersion](c, "/workflows/"+url.PathEscape(id)+"/versions/"+url.PathEscape(versionID))
}

func getJSON[T any](c *Client, path string) (*T, error) {
	return doJSON[T](c, http.MethodGet, path, nil)
}

func doJSON[T any](c *Client, method, path string, body interface{}) (*T, error) {
	respBody, err := c.request(method, path, body)
	if err != nil {
		return nil, err
	}

	var out T
	if err := json.Unmarshal(respBody, &out); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &out, nil
}
