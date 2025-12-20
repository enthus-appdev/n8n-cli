package workflow

import (
	"regexp"
	"strings"
)

// SanitizeFilename converts a workflow name to a safe filename
func SanitizeFilename(name string) string {
	// Replace spaces with underscores
	name = strings.ReplaceAll(name, " ", "_")

	// Remove or replace unsafe characters
	reg := regexp.MustCompile(`[<>:"/\\|?*]`)
	name = reg.ReplaceAllString(name, "")

	// Trim leading/trailing dots and spaces
	name = strings.Trim(name, ". ")

	// Limit length
	if len(name) > 200 {
		name = name[:200]
	}

	// Default name if empty
	if name == "" {
		name = "workflow"
	}

	return name
}

// ExtractSubWorkflowIDs extracts workflow IDs referenced by Execute Workflow nodes
func ExtractSubWorkflowIDs(nodes []map[string]interface{}) []string {
	var ids []string
	seen := make(map[string]bool)

	for _, node := range nodes {
		nodeType, ok := node["type"].(string)
		if !ok {
			continue
		}

		// Check for Execute Workflow node (various versions)
		if nodeType == "n8n-nodes-base.executeWorkflow" ||
			nodeType == "n8n-nodes-base.executeWorkflowTrigger" {

			// Try to extract workflow ID from parameters
			params, ok := node["parameters"].(map[string]interface{})
			if !ok {
				continue
			}

			// Check different parameter structures
			// Direct workflow ID
			if wfID, ok := params["workflowId"].(string); ok && wfID != "" {
				if !seen[wfID] {
					ids = append(ids, wfID)
					seen[wfID] = true
				}
			}

			// Workflow object with id
			if wf, ok := params["workflow"].(map[string]interface{}); ok {
				if wfID, ok := wf["id"].(string); ok && wfID != "" {
					if !seen[wfID] {
						ids = append(ids, wfID)
						seen[wfID] = true
					}
				}
			}

			// Check for workflow value (expression or direct)
			if wfValue, ok := params["workflowId"].(map[string]interface{}); ok {
				if value, ok := wfValue["value"].(string); ok && value != "" {
					if !seen[value] {
						ids = append(ids, value)
						seen[value] = true
					}
				}
			}
		}
	}

	return ids
}
