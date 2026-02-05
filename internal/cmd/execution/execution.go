package execution

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/enthus-appdev/n8n-cli/internal/api"
	"github.com/enthus-appdev/n8n-cli/internal/config"
)

func NewExecutionCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "execution",
		Aliases: []string{"exec"},
		Short:   "Manage workflow executions",
		Long:    `List, view, and manage workflow execution history.`,
	}

	cmd.AddCommand(newListCmd())
	cmd.AddCommand(newViewCmd())
	cmd.AddCommand(newRetryCmd())
	cmd.AddCommand(newDeleteCmd())

	return cmd
}

func getClient() (*api.Client, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, fmt.Errorf("not configured. Run 'n8n config init' first")
	}

	instance, err := cfg.GetCurrentInstance()
	if err != nil {
		return nil, err
	}

	return api.NewClient(instance.URL, instance.APIKey), nil
}

func newListCmd() *cobra.Command {
	var (
		workflowID   string
		status       string
		limit        int
		cursor       string
		resolveNames bool
	)

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List workflow executions",
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := getClient()
			if err != nil {
				return err
			}

			opts := api.ListExecutionsOptions{
				WorkflowID: workflowID,
				Status:     status,
				Limit:      limit,
				Cursor:     cursor,
			}

			result, err := client.ListExecutions(opts)
			if err != nil {
				return fmt.Errorf("failed to list executions: %w", err)
			}

			executions := result.Data

			// Optionally resolve workflow names
			workflowNames := make(map[string]string)
			if resolveNames && len(executions) > 0 {
				// Collect unique workflow IDs
				workflowIDs := make(map[string]bool)
				for _, exec := range executions {
					workflowIDs[exec.WorkflowID] = true
				}
				// Fetch workflow names
				for wfID := range workflowIDs {
					wf, err := client.GetWorkflow(wfID)
					if err == nil {
						workflowNames[wfID] = wf.Name
					}
				}
			}

			jsonFlag, _ := cmd.Flags().GetBool("json")
			if jsonFlag {
				// Enrich with workflow names if resolved
				if resolveNames {
					type enrichedExecution struct {
						api.Execution
						WorkflowName string `json:"workflowName,omitempty"`
					}
					enriched := make([]enrichedExecution, len(executions))
					for i, exec := range executions {
						enriched[i] = enrichedExecution{
							Execution:    exec,
							WorkflowName: workflowNames[exec.WorkflowID],
						}
					}
					output := struct {
						Data       []enrichedExecution `json:"data"`
						NextCursor string              `json:"nextCursor,omitempty"`
					}{
						Data:       enriched,
						NextCursor: result.NextCursor,
					}
					return printJSON(output)
				}
				return printJSON(result)
			}

			if len(executions) == 0 {
				fmt.Println("No executions found.")
				return nil
			}

			// Table output
			if resolveNames {
				fmt.Printf("%-10s  %-10s  %-20s  %s\n", "ID", "STATUS", "STARTED", "WORKFLOW")
				fmt.Printf("%-10s  %-10s  %-20s  %s\n",
					strings.Repeat("-", 10),
					strings.Repeat("-", 10),
					strings.Repeat("-", 20),
					strings.Repeat("-", 40))

				for _, exec := range executions {
					startedAt := formatTime(exec.StartedAt)
					name := workflowNames[exec.WorkflowID]
					if name == "" {
						name = exec.WorkflowID
					}
					fmt.Printf("%-10s  %-10s  %-20s  %s\n",
						exec.ID,
						exec.Status,
						startedAt,
						truncate(name, 40))
				}
			} else {
				fmt.Printf("%-10s  %-18s  %-10s  %s\n", "ID", "WORKFLOW ID", "STATUS", "STARTED")
				fmt.Printf("%-10s  %-18s  %-10s  %s\n",
					strings.Repeat("-", 10),
					strings.Repeat("-", 18),
					strings.Repeat("-", 10),
					strings.Repeat("-", 20))

				for _, exec := range executions {
					startedAt := formatTime(exec.StartedAt)
					fmt.Printf("%-10s  %-18s  %-10s  %s\n",
						exec.ID,
						exec.WorkflowID,
						exec.Status,
						startedAt)
				}
			}

			if result.NextCursor != "" {
				fmt.Printf("\nMore results available. Use --cursor %s to continue.\n", result.NextCursor)
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&workflowID, "workflow", "", "Filter by workflow ID")
	cmd.Flags().StringVar(&status, "status", "", "Filter by status (running, success, error, waiting)")
	cmd.Flags().IntVar(&limit, "limit", 20, "Maximum number of executions to return")
	cmd.Flags().StringVar(&cursor, "cursor", "", "Pagination cursor for next page")
	cmd.Flags().BoolVar(&resolveNames, "resolve-names", false, "Fetch workflow names (slower, extra API calls)")

	return cmd
}

func newViewCmd() *cobra.Command {
	var showData bool

	cmd := &cobra.Command{
		Use:   "view <execution-id>",
		Short: "View execution details",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := getClient()
			if err != nil {
				return err
			}

			exec, err := client.GetExecution(args[0], showData)
			if err != nil {
				return fmt.Errorf("failed to get execution: %w", err)
			}

			// Fetch workflow name
			workflowName := ""
			if wf, err := client.GetWorkflow(exec.WorkflowID); err == nil {
				workflowName = wf.Name
			}

			jsonFlag, _ := cmd.Flags().GetBool("json")
			if jsonFlag {
				// Enrich JSON with workflow name
				enriched := struct {
					api.Execution
					WorkflowName string `json:"workflowName,omitempty"`
				}{
					Execution:    *exec,
					WorkflowName: workflowName,
				}
				return printJSON(enriched)
			}

			// Human-readable output
			fmt.Printf("Execution ID: %s\n", exec.ID)
			if workflowName != "" {
				fmt.Printf("Workflow: %s (%s)\n", workflowName, exec.WorkflowID)
			} else {
				fmt.Printf("Workflow ID: %s\n", exec.WorkflowID)
			}
			fmt.Printf("Status: %s\n", exec.Status)
			fmt.Printf("Mode: %s\n", exec.Mode)

			if exec.StartedAt != nil {
				fmt.Printf("Started: %s\n", formatTime(exec.StartedAt))
			}
			if exec.StoppedAt != nil {
				fmt.Printf("Stopped: %s\n", formatTime(exec.StoppedAt))
			}
			if exec.StartedAt != nil && exec.StoppedAt != nil {
				duration := exec.StoppedAt.Sub(*exec.StartedAt)
				fmt.Printf("Duration: %s\n", duration.Round(time.Millisecond))
			}

			if exec.Error != "" {
				fmt.Printf("\nError: %s\n", exec.Error)
			}

			if showData && exec.Data != nil {
				printNodeData(exec.Data)
			}

			return nil
		},
	}

	cmd.Flags().BoolVar(&showData, "data", false, "Include per-node execution data")

	return cmd
}

func printNodeData(data map[string]interface{}) {
	resultData, ok := data["resultData"].(map[string]interface{})
	if !ok {
		return
	}
	runData, ok := resultData["runData"].(map[string]interface{})
	if !ok {
		return
	}

	fmt.Printf("\nNode Execution Data:\n")
	fmt.Printf("────────────────────\n")

	for nodeName, nodeRuns := range runData {
		runs, ok := nodeRuns.([]interface{})
		if !ok || len(runs) == 0 {
			continue
		}

		// Use the last run entry for this node
		run, ok := runs[len(runs)-1].(map[string]interface{})
		if !ok {
			continue
		}

		// Determine status
		nodeStatus := "success"
		if _, hasError := run["error"]; hasError {
			nodeStatus = "error"
		}

		// Get execution time
		execTimeMs := ""
		if et, ok := run["executionTime"].(float64); ok {
			execTimeMs = fmt.Sprintf("%dms", int(et))
		}

		// Count input/output items
		inputItems, outputItems := countItems(run)

		fmt.Printf("\n  %s\n", nodeName)
		fmt.Printf("    Status: %s\n", nodeStatus)
		if inputItems >= 0 || outputItems >= 0 {
			parts := []string{}
			if inputItems >= 0 {
				parts = append(parts, fmt.Sprintf("%d input", inputItems))
			}
			if outputItems >= 0 {
				parts = append(parts, fmt.Sprintf("%d output", outputItems))
			}
			fmt.Printf("    Items: %s\n", strings.Join(parts, ", "))
		}
		if execTimeMs != "" {
			fmt.Printf("    Time: %s\n", execTimeMs)
		}
	}
}

func countItems(run map[string]interface{}) (input, output int) {
	input = -1
	output = -1

	// Input items from inputData
	if inputData, ok := run["inputData"].(map[string]interface{}); ok {
		if main, ok := inputData["main"].([]interface{}); ok {
			input = 0
			for _, branch := range main {
				if items, ok := branch.([]interface{}); ok {
					input += len(items)
				}
			}
		}
	}

	// Output items from data.main
	if data, ok := run["data"].(map[string]interface{}); ok {
		if main, ok := data["main"].([]interface{}); ok {
			output = 0
			for _, branch := range main {
				if items, ok := branch.([]interface{}); ok {
					output += len(items)
				}
			}
		}
	}

	return
}

func newRetryCmd() *cobra.Command {
	var loadWorkflow bool

	cmd := &cobra.Command{
		Use:   "retry <execution-id>",
		Short: "Retry a failed execution",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := getClient()
			if err != nil {
				return err
			}

			exec, err := client.RetryExecution(args[0], loadWorkflow)
			if err != nil {
				return fmt.Errorf("failed to retry execution: %w", err)
			}

			jsonFlag, _ := cmd.Flags().GetBool("json")
			if jsonFlag {
				return printJSON(exec)
			}

			fmt.Printf("Retry started. New execution ID: %s\n", exec.ID)
			return nil
		},
	}

	cmd.Flags().BoolVar(&loadWorkflow, "load-workflow", false, "Load the latest workflow version instead of the version at execution time")

	return cmd
}

func newDeleteCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "delete <execution-id>",
		Short: "Delete an execution",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := getClient()
			if err != nil {
				return err
			}

			if err := client.DeleteExecution(args[0]); err != nil {
				return fmt.Errorf("failed to delete execution: %w", err)
			}

			fmt.Println("Execution deleted.")
			return nil
		},
	}
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max-3] + "..."
}

func formatTime(t *time.Time) string {
	if t == nil {
		return ""
	}
	return t.Local().Format("2006-01-02 15:04:05")
}

func printJSON(v interface{}) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}
