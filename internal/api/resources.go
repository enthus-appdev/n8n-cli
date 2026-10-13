package api

import (
	"net/http"
	"net/url"
	"strconv"
	"time"
)

func pagedPath(path string, limit int, cursor string, extra url.Values) string {
	params := url.Values{}
	for k, v := range extra {
		params[k] = v
	}
	if limit > 0 {
		params.Set("limit", strconv.Itoa(limit))
	}
	if cursor != "" {
		params.Set("cursor", cursor)
	}
	if len(params) > 0 {
		path += "?" + params.Encode()
	}
	return path
}

// --- Executions ---

// StopExecution stops a running or waiting execution.
func (c *Client) StopExecution(id string) (*Execution, error) {
	return doJSON[Execution](c, http.MethodPost, "/executions/"+url.PathEscape(id)+"/stop", nil)
}

// StopExecutionsOptions selects the executions to stop. Status is required.
type StopExecutionsOptions struct {
	Status        []string `json:"status"`
	WorkflowID    string   `json:"workflowId,omitempty"`
	StartedAfter  string   `json:"startedAfter,omitempty"`
	StartedBefore string   `json:"startedBefore,omitempty"`
}

// StopExecutions stops every execution matching opts and returns how many were stopped.
func (c *Client) StopExecutions(opts StopExecutionsOptions) (int, error) {
	resp, err := doJSON[struct {
		Stopped int `json:"stopped"`
	}](c, http.MethodPost, "/executions/stop", opts)
	if err != nil {
		return 0, err
	}
	return resp.Stopped, nil
}

// --- Credentials ---

// CredentialSummary is a credential as returned by the list endpoint (without secrets).
type CredentialSummary struct {
	ID        string          `json:"id"`
	Name      string          `json:"name"`
	Type      string          `json:"type"`
	CreatedAt *time.Time      `json:"createdAt,omitempty"`
	UpdatedAt *time.Time      `json:"updatedAt,omitempty"`
	Shared    []SharedProject `json:"shared,omitempty"`
}

// SharedProject is a project a credential is shared with.
type SharedProject struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Role string `json:"role"`
}

// ListCredentials returns one page of credentials.
func (c *Client) ListCredentials(limit int, cursor string) (*ListResult[CredentialSummary], error) {
	return getJSON[ListResult[CredentialSummary]](c, pagedPath("/credentials", limit, cursor, nil))
}

// CredentialTestResult is the outcome of a credential test.
type CredentialTestResult struct {
	Status  string `json:"status"`
	Message string `json:"message"`
}

// TestCredential checks a credential against its service.
func (c *Client) TestCredential(id string) (*CredentialTestResult, error) {
	return doJSON[CredentialTestResult](c, http.MethodPost, "/credentials/"+url.PathEscape(id)+"/test", nil)
}

// ListTagsPage returns one page of tags with its pagination cursor.
func (c *Client) ListTagsPage(limit int, cursor string) (*ListResult[Tag], error) {
	return getJSON[ListResult[Tag]](c, pagedPath("/tags", limit, cursor, nil))
}

// --- Folders ---

// Folder is a workflow folder inside a project.
type Folder struct {
	ID              string     `json:"id"`
	Name            string     `json:"name"`
	ParentFolderID  *string    `json:"parentFolderId"`
	CreatedAt       *time.Time `json:"createdAt,omitempty"`
	UpdatedAt       *time.Time `json:"updatedAt,omitempty"`
	TotalSubFolders *int       `json:"totalSubFolders,omitempty"`
	TotalWorkflows  *int       `json:"totalWorkflows,omitempty"`
}

func folderPath(projectID string) string {
	return "/projects/" + url.PathEscape(projectID) + "/folders"
}

// ListFolders returns the folders of a project. Folders use skip/take instead of cursors.
func (c *Client) ListFolders(projectID string, skip, take int) ([]Folder, int, error) {
	params := url.Values{}
	if skip > 0 {
		params.Set("skip", strconv.Itoa(skip))
	}
	if take > 0 {
		params.Set("take", strconv.Itoa(take))
	}
	path := folderPath(projectID)
	if len(params) > 0 {
		path += "?" + params.Encode()
	}

	resp, err := getJSON[struct {
		Count int      `json:"count"`
		Data  []Folder `json:"data"`
	}](c, path)
	if err != nil {
		return nil, 0, err
	}
	return resp.Data, resp.Count, nil
}

// GetFolder returns a folder with its sub-folder and workflow counts.
func (c *Client) GetFolder(projectID, folderID string) (*Folder, error) {
	return getJSON[Folder](c, folderPath(projectID)+"/"+url.PathEscape(folderID))
}

// CreateFolder creates a folder; an empty parentFolderID creates it at the project root.
func (c *Client) CreateFolder(projectID, name, parentFolderID string) (*Folder, error) {
	body := map[string]string{"name": name}
	if parentFolderID != "" {
		body["parentFolderId"] = parentFolderID
	}
	return doJSON[Folder](c, http.MethodPost, folderPath(projectID), body)
}

// UpdateFolder renames and/or moves a folder. Empty arguments are left unchanged.
func (c *Client) UpdateFolder(projectID, folderID, name, parentFolderID string) (*Folder, error) {
	body := map[string]string{}
	if name != "" {
		body["name"] = name
	}
	if parentFolderID != "" {
		body["parentFolderId"] = parentFolderID
	}
	return doJSON[Folder](c, http.MethodPatch, folderPath(projectID)+"/"+url.PathEscape(folderID), body)
}

// DeleteFolder deletes a folder. With transferToFolderID set, its workflows and
// sub-folders move there; otherwise its workflows move to the project root and
// are archived, and its sub-folders are deleted.
func (c *Client) DeleteFolder(projectID, folderID, transferToFolderID string) error {
	path := folderPath(projectID) + "/" + url.PathEscape(folderID)
	if transferToFolderID != "" {
		path += "?transferToFolderId=" + url.QueryEscape(transferToFolderID)
	}
	_, err := c.request(http.MethodDelete, path, nil)
	return err
}

// --- Users and project members ---

// User is an n8n user.
type User struct {
	ID         string     `json:"id"`
	Email      string     `json:"email"`
	FirstName  string     `json:"firstName,omitempty"`
	LastName   string     `json:"lastName,omitempty"`
	IsPending  bool       `json:"isPending"`
	Role       string     `json:"role,omitempty"`
	MFAEnabled bool       `json:"mfaEnabled,omitempty"`
	CreatedAt  *time.Time `json:"createdAt,omitempty"`
	UpdatedAt  *time.Time `json:"updatedAt,omitempty"`
}

// ListUsers returns one page of users, optionally limited to a project's members.
func (c *Client) ListUsers(limit int, cursor, projectID string) (*ListResult[User], error) {
	extra := url.Values{"includeRole": {"true"}}
	if projectID != "" {
		extra.Set("projectId", projectID)
	}
	return getJSON[ListResult[User]](c, pagedPath("/users", limit, cursor, extra))
}

// GetUser returns a user by ID or email address.
func (c *Client) GetUser(idOrEmail string) (*User, error) {
	return getJSON[User](c, "/users/"+url.PathEscape(idOrEmail)+"?includeRole=true")
}

func projectUsersPath(projectID string) string {
	return "/projects/" + url.PathEscape(projectID) + "/users"
}

// ListProjectMembers returns one page of a project's members with their project role.
func (c *Client) ListProjectMembers(projectID string, limit int, cursor string) (*ListResult[User], error) {
	return getJSON[ListResult[User]](c, pagedPath(projectUsersPath(projectID), limit, cursor, nil))
}

// ProjectRelation assigns a project role to a user.
type ProjectRelation struct {
	UserID string `json:"userId"`
	Role   string `json:"role"`
}

// AddProjectMembers adds users to a project with the given roles.
func (c *Client) AddProjectMembers(projectID string, relations []ProjectRelation) error {
	_, err := c.request(http.MethodPost, projectUsersPath(projectID), map[string]interface{}{"relations": relations})
	return err
}

// SetProjectMemberRole changes a member's role in a project.
func (c *Client) SetProjectMemberRole(projectID, userID, role string) error {
	_, err := c.request(http.MethodPatch, projectUsersPath(projectID)+"/"+url.PathEscape(userID), map[string]string{"role": role})
	return err
}

// RemoveProjectMember removes a user from a project.
func (c *Client) RemoveProjectMember(projectID, userID string) error {
	_, err := c.request(http.MethodDelete, projectUsersPath(projectID)+"/"+url.PathEscape(userID), nil)
	return err
}

// --- Audit ---

// AuditOptions narrows a security audit. Zero values use the server defaults.
type AuditOptions struct {
	Categories            []string
	DaysAbandonedWorkflow int
}

// GenerateAudit runs a security audit and returns the report keyed by risk category.
func (c *Client) GenerateAudit(opts AuditOptions) (map[string]interface{}, error) {
	extra := map[string]interface{}{}
	if len(opts.Categories) > 0 {
		extra["categories"] = opts.Categories
	}
	if opts.DaysAbandonedWorkflow > 0 {
		extra["daysAbandonedWorkflow"] = opts.DaysAbandonedWorkflow
	}
	body := map[string]interface{}{}
	if len(extra) > 0 {
		body["additionalOptions"] = extra
	}

	resp, err := doJSON[map[string]interface{}](c, http.MethodPost, "/audit", body)
	if err != nil {
		return nil, err
	}
	return *resp, nil
}
