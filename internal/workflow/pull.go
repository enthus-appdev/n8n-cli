package workflow

import (
	"fmt"

	"github.com/enthus-appdev/n8n-cli/internal/api"
)

// Manifest tracks workflow relationships for a pull/push operation
type Manifest struct {
	// RootWorkflow is the main workflow that was pulled
	RootWorkflow string `json:"rootWorkflow"`

	// Workflows maps workflow ID to metadata
	Workflows map[string]WorkflowMeta `json:"workflows"`

	// Dependencies maps workflow ID to IDs of sub-workflows it depends on
	Dependencies map[string][]string `json:"dependencies"`

	// Instance information
	Instance string `json:"instance,omitempty"`
}

// WorkflowMeta contains metadata about a pulled workflow
type WorkflowMeta struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Filename string `json:"filename"`
	Active   bool   `json:"active"`
}

// PullResult contains the results of a recursive pull operation
type PullResult struct {
	Workflows map[string]*api.Workflow
	Manifest  *Manifest
}

// RecursivePuller handles recursive workflow pulling
type RecursivePuller struct {
	client   *api.Client
	pulled   map[string]*api.Workflow
	manifest *Manifest
}

// NewRecursivePuller creates a new recursive puller
func NewRecursivePuller(client *api.Client) *RecursivePuller {
	return &RecursivePuller{
		client: client,
		pulled: make(map[string]*api.Workflow),
		manifest: &Manifest{
			Workflows:    make(map[string]WorkflowMeta),
			Dependencies: make(map[string][]string),
		},
	}
}

// Pull recursively pulls a workflow and all its sub-workflows
func (p *RecursivePuller) Pull(workflowID string) (*PullResult, error) {
	p.manifest.RootWorkflow = workflowID

	if err := p.pullRecursive(workflowID); err != nil {
		return nil, err
	}

	return &PullResult{
		Workflows: p.pulled,
		Manifest:  p.manifest,
	}, nil
}

func (p *RecursivePuller) pullRecursive(workflowID string) error {
	// Skip if already pulled
	if _, exists := p.pulled[workflowID]; exists {
		return nil
	}

	// Fetch the workflow
	wf, err := p.client.GetWorkflow(workflowID)
	if err != nil {
		return fmt.Errorf("failed to get workflow %s: %w", workflowID, err)
	}

	p.pulled[workflowID] = wf

	// Add to manifest
	filename := SanitizeFilename(wf.Name) + ".json"
	p.manifest.Workflows[workflowID] = WorkflowMeta{
		ID:       wf.ID,
		Name:     wf.Name,
		Filename: filename,
		Active:   wf.Active,
	}

	// Extract sub-workflow IDs
	subIDs := ExtractSubWorkflowIDs(wf.Nodes)
	if len(subIDs) > 0 {
		p.manifest.Dependencies[workflowID] = subIDs
	}

	// Recursively pull sub-workflows
	for _, subID := range subIDs {
		if err := p.pullRecursive(subID); err != nil {
			// Log warning but continue - sub-workflow might be deleted or inaccessible
			fmt.Printf("Warning: could not pull sub-workflow %s: %v\n", subID, err)
		}
	}

	return nil
}

// GetPushOrder returns workflow IDs in dependency order (dependencies first)
func (m *Manifest) GetPushOrder() []string {
	// Build reverse dependency graph
	dependedBy := make(map[string][]string)
	inDegree := make(map[string]int)

	for id := range m.Workflows {
		inDegree[id] = 0
	}

	for id, deps := range m.Dependencies {
		for _, dep := range deps {
			if _, exists := m.Workflows[dep]; exists {
				dependedBy[dep] = append(dependedBy[dep], id)
				inDegree[id]++
			}
		}
	}

	// Topological sort using Kahn's algorithm
	var order []string
	var queue []string

	// Start with nodes that have no dependencies
	for id, degree := range inDegree {
		if degree == 0 {
			queue = append(queue, id)
		}
	}

	for len(queue) > 0 {
		// Pop from queue
		id := queue[0]
		queue = queue[1:]
		order = append(order, id)

		// Reduce in-degree for dependents
		for _, dependent := range dependedBy[id] {
			inDegree[dependent]--
			if inDegree[dependent] == 0 {
				queue = append(queue, dependent)
			}
		}
	}

	return order
}
