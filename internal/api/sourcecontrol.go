package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
)

// SourceControlFile is one entry of a source control status, push or pull result.
type SourceControlFile struct {
	File                   string                  `json:"file"`
	ID                     string                  `json:"id"`
	Name                   string                  `json:"name"`
	Type                   string                  `json:"type"`
	Status                 string                  `json:"status"`
	Location               string                  `json:"location"`
	Conflict               bool                    `json:"conflict"`
	UpdatedAt              string                  `json:"updatedAt"`
	Pushed                 *bool                   `json:"pushed,omitempty"`
	IsLocalPublished       *bool                   `json:"isLocalPublished,omitempty"`
	IsRemoteArchived       *bool                   `json:"isRemoteArchived,omitempty"`
	ParentFolderID         *string                 `json:"parentFolderId,omitempty"`
	FolderPath             []string                `json:"folderPath,omitempty"`
	RemoteFolderPath       []string                `json:"remoteFolderPath,omitempty"`
	Owner                  *SourceControlOwner     `json:"owner,omitempty"`
	PublishingError        string                  `json:"publishingError,omitempty"`
	PublishingErrorDetails *PublishingErrorDetails `json:"publishingErrorDetails,omitempty"`
	ContentImportPolicy    json.RawMessage         `json:"contentImportPolicy,omitempty"`
}

// SourceControlOwner is the project that owns a source control item.
type SourceControlOwner struct {
	Type        string `json:"type"`
	ProjectID   string `json:"projectId"`
	ProjectName string `json:"projectName"`
}

// PublishingErrorDetails explains why an imported workflow was not published.
type PublishingErrorDetails struct {
	Reason                  string `json:"reason"`
	WorkflowReviewRequestID string `json:"workflowReviewRequestId,omitempty"`
	VersionID               string `json:"versionId,omitempty"`
}

// SourceControlItem identifies one item to push.
type SourceControlItem struct {
	ID   string `json:"id"`
	Type string `json:"type"`
}

// SourceControlTypes are the item types the source control API accepts.
var SourceControlTypes = []string{"credential", "workflow", "tags", "variables", "file", "folders", "project", "datatable"}

// SourceControlConflictError is returned when a push or pull is rejected with 409.
type SourceControlConflictError struct {
	Message   string
	Conflicts []SourceControlFile
}

func (e *SourceControlConflictError) Error() string {
	msg := e.Message
	if msg == "" {
		msg = "conflicting changes"
	}
	return fmt.Sprintf("source control conflict (%d files): %s", len(e.Conflicts), msg)
}

// withScopeHint adds the required API key scope to a 403 error.
func withScopeHint(err error, scope string) error {
	var apiErr *APIError
	if errors.As(err, &apiErr) && apiErr.StatusCode == http.StatusForbidden {
		return fmt.Errorf("%w (the API key needs the %s scope, and source control must be connected on the instance)", err, scope)
	}
	return err
}

// SourceControlStatus previews pending changes in the given direction ("push" or "pull").
func (c *Client) SourceControlStatus(direction string) ([]SourceControlFile, error) {
	respBody, err := c.request(http.MethodGet, "/source-control/status?direction="+url.QueryEscape(direction), nil)
	if err != nil {
		return nil, withScopeHint(err, "sourceControl:read")
	}

	var resp struct {
		Data []SourceControlFile `json:"data"`
	}
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return resp.Data, nil
}

// SourceControlPushOptions contains the parameters of a source control push.
type SourceControlPushOptions struct {
	CommitMessage string              `json:"commitMessage"`
	FileNames     []SourceControlItem `json:"fileNames"`
	Force         bool                `json:"force,omitempty"`
}

// SourceControlPush commits the selected items and pushes them to the connected branch.
func (c *Client) SourceControlPush(opts SourceControlPushOptions) ([]SourceControlFile, error) {
	respBody, err := c.request(http.MethodPost, "/source-control/push", opts)
	if err != nil {
		var apiErr *APIError
		if errors.As(err, &apiErr) && apiErr.StatusCode == http.StatusConflict {
			var conflict struct {
				Message   string              `json:"message"`
				Conflicts []SourceControlFile `json:"conflicts"`
			}
			if json.Unmarshal(apiErr.Body, &conflict) == nil {
				return nil, &SourceControlConflictError{Message: conflict.Message, Conflicts: conflict.Conflicts}
			}
		}
		return nil, withScopeHint(err, "sourceControl:push")
	}

	var resp struct {
		Data []SourceControlFile `json:"data"`
	}
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return resp.Data, nil
}

// SourceControlPullOptions contains the parameters of a source control pull.
type SourceControlPullOptions struct {
	Force       bool   `json:"force,omitempty"`
	AutoPublish string `json:"autoPublish,omitempty"` // none (server default), all, published
}

// SourceControlPull imports the connected branch into the instance.
// Unlike status and push, the pull endpoint returns a bare array.
func (c *Client) SourceControlPull(opts SourceControlPullOptions) ([]SourceControlFile, error) {
	// The request body is required, so an empty options struct is sent as {}.
	respBody, err := c.request(http.MethodPost, "/source-control/pull", opts)
	if err != nil {
		var apiErr *APIError
		if errors.As(err, &apiErr) && apiErr.StatusCode == http.StatusConflict {
			// The 409 body lists every file in the pull diff; the conflicting
			// ones are flagged or still modified locally.
			var files []SourceControlFile
			if json.Unmarshal(apiErr.Body, &files) == nil {
				var conflicts []SourceControlFile
				for _, f := range files {
					if f.Conflict || f.Status == "modified" {
						conflicts = append(conflicts, f)
					}
				}
				return nil, &SourceControlConflictError{Message: "local changes would be overwritten", Conflicts: conflicts}
			}
		}
		return nil, withScopeHint(err, "sourceControl:pull")
	}

	var files []SourceControlFile
	if err := json.Unmarshal(respBody, &files); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return files, nil
}
