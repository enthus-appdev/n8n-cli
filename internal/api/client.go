package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

// Client is the n8n API client
type Client struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
}

// NewClient creates a new n8n API client
func NewClient(baseURL, apiKey string) *Client {
	return &Client{
		baseURL: baseURL,
		apiKey:  apiKey,
		httpClient: &http.Client{
			Timeout: 5 * time.Minute,
		},
	}
}

// Workflow represents an n8n workflow
type Workflow struct {
	ID        string                   `json:"id,omitempty"`
	Name      string                   `json:"name"`
	Active    bool                     `json:"active"`
	Nodes     []map[string]interface{} `json:"nodes"`
	Connections map[string]interface{} `json:"connections"`
	Settings    map[string]interface{} `json:"settings,omitempty"`
	StaticData  interface{}            `json:"staticData,omitempty"`
	Tags        []Tag                  `json:"tags,omitempty"`
	Shared      []WorkflowShared       `json:"shared,omitempty"`
	CreatedAt   *time.Time             `json:"createdAt,omitempty"`
	UpdatedAt   *time.Time             `json:"updatedAt,omitempty"`
}

// Tag represents a workflow tag
type Tag struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// Project represents an n8n project
type Project struct {
	ID        string     `json:"id"`
	Name      string     `json:"name"`
	Type      string     `json:"type"`
	CreatedAt *time.Time `json:"createdAt,omitempty"`
	UpdatedAt *time.Time `json:"updatedAt,omitempty"`
}

// WorkflowShared represents the sharing/ownership info for a workflow
type WorkflowShared struct {
	Role      string   `json:"role"`
	ProjectID string   `json:"projectId"`
	Project   *Project `json:"project,omitempty"`
}

// Execution represents a workflow execution
type Execution struct {
	ID           string                 `json:"id"`
	WorkflowID   string                 `json:"workflowId"`
	WorkflowName string                 `json:"workflowName,omitempty"`
	Finished     bool                   `json:"finished"`
	Mode         string                 `json:"mode"`
	Status       string                 `json:"status"`
	StartedAt    *time.Time             `json:"startedAt,omitempty"`
	StoppedAt    *time.Time             `json:"stoppedAt,omitempty"`
	Data         map[string]interface{} `json:"data,omitempty"`
	Error        string                 `json:"error,omitempty"`
}

// ListWorkflowsOptions contains options for listing workflows
type ListWorkflowsOptions struct {
	Active           *bool
	Tags             []string
	Name             string
	ProjectID        string
	ExcludePinnedData bool
	Limit            int
	Cursor           string
}

// ListExecutionsOptions contains options for listing executions
type ListExecutionsOptions struct {
	WorkflowID  string
	Status      string // running, success, error, waiting
	ProjectID   string
	IncludeData bool
	Limit       int
	Cursor      string
}

// Credential represents an n8n credential
type Credential struct {
	ID        string                 `json:"id,omitempty"`
	Name      string                 `json:"name"`
	Type      string                 `json:"type"`
	Data      map[string]interface{} `json:"data,omitempty"`
	CreatedAt *time.Time             `json:"createdAt,omitempty"`
	UpdatedAt *time.Time             `json:"updatedAt,omitempty"`
}

// Variable represents an n8n variable
type Variable struct {
	ID    string `json:"id,omitempty"`
	Key   string `json:"key"`
	Value string `json:"value"`
	Type  string `json:"type,omitempty"`
}

// ListResult contains paginated list results
type ListResult[T any] struct {
	Data       []T    `json:"data"`
	NextCursor string `json:"nextCursor,omitempty"`
}

// request makes an HTTP request to the n8n API
func (c *Client) request(method, path string, body interface{}) ([]byte, error) {
	var reqBody io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		reqBody = bytes.NewReader(jsonBody)
	}

	reqURL := c.baseURL + "/api/v1" + path
	req, err := http.NewRequest(method, reqURL, reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("X-N8N-API-KEY", c.apiKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode >= 400 {
		var apiErr struct {
			Message string `json:"message"`
		}
		if json.Unmarshal(respBody, &apiErr) == nil && apiErr.Message != "" {
			return nil, fmt.Errorf("API error (%d): %s", resp.StatusCode, apiErr.Message)
		}
		return nil, fmt.Errorf("API error (%d): %s", resp.StatusCode, string(respBody))
	}

	return respBody, nil
}

// ListWorkflows returns all workflows
func (c *Client) ListWorkflows(opts ListWorkflowsOptions) (*ListResult[Workflow], error) {
	params := url.Values{}
	if opts.Limit > 0 {
		params.Set("limit", strconv.Itoa(opts.Limit))
	}
	if opts.Active != nil {
		params.Set("active", strconv.FormatBool(*opts.Active))
	}
	if opts.Cursor != "" {
		params.Set("cursor", opts.Cursor)
	}
	if opts.Name != "" {
		params.Set("name", opts.Name)
	}
	if opts.ProjectID != "" {
		params.Set("projectId", opts.ProjectID)
	}
	if opts.ExcludePinnedData {
		params.Set("excludePinnedData", "true")
	}
	for _, tag := range opts.Tags {
		params.Add("tags", tag)
	}

	path := "/workflows"
	if len(params) > 0 {
		path += "?" + params.Encode()
	}

	respBody, err := c.request(http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}

	var resp ListResult[Workflow]
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &resp, nil
}

// GetWorkflow returns a workflow by ID
func (c *Client) GetWorkflow(id string) (*Workflow, error) {
	respBody, err := c.request(http.MethodGet, "/workflows/"+url.PathEscape(id), nil)
	if err != nil {
		return nil, err
	}

	var wf Workflow
	if err := json.Unmarshal(respBody, &wf); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &wf, nil
}

// CreateWorkflow creates a new workflow
func (c *Client) CreateWorkflow(wf *Workflow) (*Workflow, error) {
	// Only send fields that the API accepts (id, active, tags are read-only)
	body := &WorkflowUpdateRequest{
		Name:        wf.Name,
		Nodes:       wf.Nodes,
		Connections: wf.Connections,
		Settings:    wf.Settings,
		StaticData:  wf.StaticData,
	}

	respBody, err := c.request(http.MethodPost, "/workflows", body)
	if err != nil {
		return nil, err
	}

	var created Workflow
	if err := json.Unmarshal(respBody, &created); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &created, nil
}

// WorkflowUpdateRequest contains only the fields allowed in update requests
type WorkflowUpdateRequest struct {
	Name        string                   `json:"name"`
	Nodes       []map[string]interface{} `json:"nodes"`
	Connections map[string]interface{}   `json:"connections"`
	Settings    map[string]interface{}   `json:"settings,omitempty"`
	StaticData  interface{}              `json:"staticData,omitempty"`
}

// UpdateWorkflow updates an existing workflow
func (c *Client) UpdateWorkflow(id string, wf *Workflow) (*Workflow, error) {
	// Only send fields that the API accepts (id, active, tags are read-only)
	body := &WorkflowUpdateRequest{
		Name:        wf.Name,
		Nodes:       wf.Nodes,
		Connections: wf.Connections,
		Settings:    wf.Settings,
		StaticData:  wf.StaticData,
	}

	respBody, err := c.request(http.MethodPut, "/workflows/"+url.PathEscape(id), body)
	if err != nil {
		return nil, err
	}

	var updated Workflow
	if err := json.Unmarshal(respBody, &updated); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &updated, nil
}

// DeleteWorkflow deletes a workflow
func (c *Client) DeleteWorkflow(id string) error {
	_, err := c.request(http.MethodDelete, "/workflows/"+url.PathEscape(id), nil)
	return err
}

// ActivateWorkflow activates a workflow
func (c *Client) ActivateWorkflow(id string) error {
	_, err := c.request(http.MethodPost, "/workflows/"+url.PathEscape(id)+"/activate", nil)
	return err
}

// DeactivateWorkflow deactivates a workflow
func (c *Client) DeactivateWorkflow(id string) error {
	_, err := c.request(http.MethodPost, "/workflows/"+url.PathEscape(id)+"/deactivate", nil)
	return err
}

// GetWorkflowTags returns tags for a workflow
func (c *Client) GetWorkflowTags(id string) ([]Tag, error) {
	respBody, err := c.request(http.MethodGet, "/workflows/"+url.PathEscape(id)+"/tags", nil)
	if err != nil {
		return nil, err
	}

	var tags []Tag
	if err := json.Unmarshal(respBody, &tags); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return tags, nil
}

// UpdateWorkflowTags updates tags for a workflow
func (c *Client) UpdateWorkflowTags(id string, tagIDs []string) ([]Tag, error) {
	body := map[string]interface{}{
		"tagIds": tagIDs,
	}

	respBody, err := c.request(http.MethodPut, "/workflows/"+url.PathEscape(id)+"/tags", body)
	if err != nil {
		return nil, err
	}

	var tags []Tag
	if err := json.Unmarshal(respBody, &tags); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return tags, nil
}

// TransferWorkflow transfers a workflow to another project
func (c *Client) TransferWorkflow(id, destinationProjectID string) error {
	body := map[string]string{
		"destinationProjectId": destinationProjectID,
	}
	_, err := c.request(http.MethodPut, "/workflows/"+url.PathEscape(id)+"/transfer", body)
	return err
}

// ListProjects returns all projects
func (c *Client) ListProjects(limit int, cursor string) (*ListResult[Project], error) {
	params := url.Values{}
	if limit > 0 {
		params.Set("limit", strconv.Itoa(limit))
	}
	if cursor != "" {
		params.Set("cursor", cursor)
	}

	path := "/projects"
	if len(params) > 0 {
		path += "?" + params.Encode()
	}

	respBody, err := c.request(http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}

	var resp ListResult[Project]
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &resp, nil
}

// ExecuteWorkflow executes a workflow (requires n8n 1.x with execute endpoint)
func (c *Client) ExecuteWorkflow(id string, data map[string]interface{}, wait bool) (*Execution, error) {
	body := map[string]interface{}{}
	if data != nil {
		body["data"] = data
	}

	path := "/workflows/" + url.PathEscape(id) + "/execute"
	if wait {
		path += "?wait=true"
	}

	respBody, err := c.request(http.MethodPost, path, body)
	if err != nil {
		return nil, err
	}

	var exec Execution
	if err := json.Unmarshal(respBody, &exec); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &exec, nil
}

// ListExecutions returns workflow executions
func (c *Client) ListExecutions(opts ListExecutionsOptions) (*ListResult[Execution], error) {
	params := url.Values{}
	if opts.Limit > 0 {
		params.Set("limit", strconv.Itoa(opts.Limit))
	}
	if opts.WorkflowID != "" {
		params.Set("workflowId", opts.WorkflowID)
	}
	if opts.Status != "" {
		params.Set("status", opts.Status)
	}
	if opts.ProjectID != "" {
		params.Set("projectId", opts.ProjectID)
	}
	if opts.IncludeData {
		params.Set("includeData", "true")
	}
	if opts.Cursor != "" {
		params.Set("cursor", opts.Cursor)
	}

	path := "/executions"
	if len(params) > 0 {
		path += "?" + params.Encode()
	}

	respBody, err := c.request(http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}

	var resp ListResult[Execution]
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &resp, nil
}

// GetExecution returns an execution by ID
func (c *Client) GetExecution(id string, includeData bool) (*Execution, error) {
	path := "/executions/" + url.PathEscape(id)
	if includeData {
		path += "?includeData=true"
	}
	respBody, err := c.request(http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}

	var exec Execution
	if err := json.Unmarshal(respBody, &exec); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &exec, nil
}

// RetryExecution retries a failed execution
func (c *Client) RetryExecution(id string, loadWorkflow bool) (*Execution, error) {
	var body interface{}
	if loadWorkflow {
		body = map[string]bool{"loadWorkflow": true}
	}

	respBody, err := c.request(http.MethodPost, "/executions/"+url.PathEscape(id)+"/retry", body)
	if err != nil {
		return nil, err
	}

	var exec Execution
	if err := json.Unmarshal(respBody, &exec); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &exec, nil
}

// DeleteExecution deletes an execution
func (c *Client) DeleteExecution(id string) error {
	_, err := c.request(http.MethodDelete, "/executions/"+url.PathEscape(id), nil)
	return err
}

// --- Tags ---

// ListTags returns all tags
func (c *Client) ListTags(limit int, cursor string) ([]Tag, error) {
	params := url.Values{}
	if limit > 0 {
		params.Set("limit", strconv.Itoa(limit))
	}
	if cursor != "" {
		params.Set("cursor", cursor)
	}

	path := "/tags"
	if len(params) > 0 {
		path += "?" + params.Encode()
	}

	respBody, err := c.request(http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}

	var resp struct {
		Data []Tag `json:"data"`
	}
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return resp.Data, nil
}

// GetTag returns a tag by ID
func (c *Client) GetTag(id string) (*Tag, error) {
	respBody, err := c.request(http.MethodGet, "/tags/"+url.PathEscape(id), nil)
	if err != nil {
		return nil, err
	}

	var tag Tag
	if err := json.Unmarshal(respBody, &tag); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &tag, nil
}

// CreateTag creates a new tag
func (c *Client) CreateTag(name string) (*Tag, error) {
	body := map[string]string{"name": name}
	respBody, err := c.request(http.MethodPost, "/tags", body)
	if err != nil {
		return nil, err
	}

	var tag Tag
	if err := json.Unmarshal(respBody, &tag); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &tag, nil
}

// UpdateTag updates a tag
func (c *Client) UpdateTag(id, name string) (*Tag, error) {
	body := map[string]string{"name": name}
	respBody, err := c.request(http.MethodPut, "/tags/"+url.PathEscape(id), body)
	if err != nil {
		return nil, err
	}

	var tag Tag
	if err := json.Unmarshal(respBody, &tag); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &tag, nil
}

// DeleteTag deletes a tag
func (c *Client) DeleteTag(id string) error {
	_, err := c.request(http.MethodDelete, "/tags/"+url.PathEscape(id), nil)
	return err
}

// --- Credentials ---

// CreateCredential creates a new credential
func (c *Client) CreateCredential(cred *Credential) (*Credential, error) {
	respBody, err := c.request(http.MethodPost, "/credentials", cred)
	if err != nil {
		return nil, err
	}

	var created Credential
	if err := json.Unmarshal(respBody, &created); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &created, nil
}

// DeleteCredential deletes a credential
func (c *Client) DeleteCredential(id string) error {
	_, err := c.request(http.MethodDelete, "/credentials/"+url.PathEscape(id), nil)
	return err
}

// GetCredentialSchema returns the schema for a credential type
func (c *Client) GetCredentialSchema(typeName string) (map[string]interface{}, error) {
	respBody, err := c.request(http.MethodGet, "/credentials/schema/"+url.PathEscape(typeName), nil)
	if err != nil {
		return nil, err
	}

	var schema map[string]interface{}
	if err := json.Unmarshal(respBody, &schema); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return schema, nil
}

// TransferCredential transfers a credential to another project
func (c *Client) TransferCredential(id, destinationProjectID string) error {
	body := map[string]string{
		"destinationProjectId": destinationProjectID,
	}
	_, err := c.request(http.MethodPut, "/credentials/"+url.PathEscape(id)+"/transfer", body)
	return err
}

// --- Variables ---

// ListVariables returns all variables
func (c *Client) ListVariables(limit int, cursor string) ([]Variable, error) {
	params := url.Values{}
	if limit > 0 {
		params.Set("limit", strconv.Itoa(limit))
	}
	if cursor != "" {
		params.Set("cursor", cursor)
	}

	path := "/variables"
	if len(params) > 0 {
		path += "?" + params.Encode()
	}

	respBody, err := c.request(http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}

	var resp struct {
		Data []Variable `json:"data"`
	}
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return resp.Data, nil
}

// CreateVariable creates a new variable
func (c *Client) CreateVariable(key, value string) error {
	body := map[string]string{"key": key, "value": value}
	_, err := c.request(http.MethodPost, "/variables", body)
	return err
}

// UpdateVariable updates a variable
func (c *Client) UpdateVariable(id, key, value string) error {
	body := map[string]string{"key": key, "value": value}
	_, err := c.request(http.MethodPut, "/variables/"+url.PathEscape(id), body)
	return err
}

// DeleteVariable deletes a variable
func (c *Client) DeleteVariable(id string) error {
	_, err := c.request(http.MethodDelete, "/variables/"+url.PathEscape(id), nil)
	return err
}
