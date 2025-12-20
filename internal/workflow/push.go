package workflow

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/enthus-appdev/n8n-cli/internal/api"
)

// Pusher handles pushing workflows to n8n
type Pusher struct {
	client *api.Client
	dir    string
	// Maps old IDs to new IDs (for create mode)
	idMapping map[string]string
}

// NewPusher creates a new workflow pusher
func NewPusher(client *api.Client, dir string) *Pusher {
	return &Pusher{
		client:    client,
		dir:       dir,
		idMapping: make(map[string]string),
	}
}

// Push pushes workflows according to the manifest
func (p *Pusher) Push(manifest *Manifest, create bool) error {
	// Get push order (dependencies first)
	order := manifest.GetPushOrder()

	for _, id := range order {
		meta, exists := manifest.Workflows[id]
		if !exists {
			continue
		}

		// Read workflow file
		filePath := filepath.Join(p.dir, meta.Filename)
		data, err := os.ReadFile(filePath)
		if err != nil {
			return fmt.Errorf("failed to read %s: %w", meta.Filename, err)
		}

		var wf api.Workflow
		if err := json.Unmarshal(data, &wf); err != nil {
			return fmt.Errorf("failed to parse %s: %w", meta.Filename, err)
		}

		// Update sub-workflow references if we're creating new workflows
		if create && len(p.idMapping) > 0 {
			p.updateSubWorkflowReferences(&wf)
		}

		if create {
			// Remove ID so n8n generates a new one
			wf.ID = ""
			created, err := p.client.CreateWorkflow(&wf)
			if err != nil {
				return fmt.Errorf("failed to create workflow %s: %w", meta.Name, err)
			}
			// Store ID mapping for dependent workflows
			p.idMapping[id] = created.ID
			fmt.Printf("Created: %s (ID: %s)\n", created.Name, created.ID)
		} else {
			updated, err := p.client.UpdateWorkflow(wf.ID, &wf)
			if err != nil {
				return fmt.Errorf("failed to update workflow %s: %w", meta.Name, err)
			}
			fmt.Printf("Updated: %s (ID: %s)\n", updated.Name, updated.ID)
		}
	}

	return nil
}

// updateSubWorkflowReferences updates Execute Workflow node references
// to use new IDs when creating copies of workflows
func (p *Pusher) updateSubWorkflowReferences(wf *api.Workflow) {
	for i, node := range wf.Nodes {
		nodeType, ok := node["type"].(string)
		if !ok {
			continue
		}

		if nodeType != "n8n-nodes-base.executeWorkflow" {
			continue
		}

		params, ok := node["parameters"].(map[string]interface{})
		if !ok {
			continue
		}

		// Update workflowId parameter
		if oldID, ok := params["workflowId"].(string); ok {
			if newID, exists := p.idMapping[oldID]; exists {
				params["workflowId"] = newID
				wf.Nodes[i]["parameters"] = params
			}
		}

		// Update workflow.id if present
		if wfObj, ok := params["workflow"].(map[string]interface{}); ok {
			if oldID, ok := wfObj["id"].(string); ok {
				if newID, exists := p.idMapping[oldID]; exists {
					wfObj["id"] = newID
					params["workflow"] = wfObj
					wf.Nodes[i]["parameters"] = params
				}
			}
		}
	}
}
