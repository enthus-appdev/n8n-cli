package workflow

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/enthus-appdev/n8n-cli/internal/api"
	"github.com/enthus-appdev/n8n-cli/internal/config"
	"github.com/enthus-appdev/n8n-cli/internal/workflow"
)

func NewWorkflowCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "workflow",
		Aliases: []string{"wf"},
		Short:   "Manage n8n workflows",
		Long:    `List, view, pull, push, and manage n8n workflows.`,
	}

	cmd.AddCommand(newListCmd())
	cmd.AddCommand(newViewCmd())
	cmd.AddCommand(newPullCmd())
	cmd.AddCommand(newPushCmd())
	cmd.AddCommand(newRunCmd())
	cmd.AddCommand(newActivateCmd())
	cmd.AddCommand(newDeactivateCmd())

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
		active   bool
		inactive bool
		tags     []string
		limit    int
		cursor   string
	)

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List all workflows",
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := getClient()
			if err != nil {
				return err
			}

			opts := api.ListWorkflowsOptions{
				Limit:  limit,
				Tags:   tags,
				Cursor: cursor,
			}

			if active && !inactive {
				opts.Active = boolPtr(true)
			} else if inactive && !active {
				opts.Active = boolPtr(false)
			}

			result, err := client.ListWorkflows(opts)
			if err != nil {
				return fmt.Errorf("failed to list workflows: %w", err)
			}

			jsonFlag, _ := cmd.Flags().GetBool("json")
			if jsonFlag {
				return printJSON(result)
			}

			if len(result.Data) == 0 {
				fmt.Println("No workflows found.")
				return nil
			}

			// Table output
			fmt.Printf("%-18s  %-6s  %s\n", "ID", "ACTIVE", "NAME")
			fmt.Printf("%-18s  %-6s  %s\n", strings.Repeat("-", 18), "------", strings.Repeat("-", 50))
			for _, wf := range result.Data {
				activeStr := "no"
				if wf.Active {
					activeStr = "yes"
				}
				fmt.Printf("%-18s  %-6s  %s\n", wf.ID, activeStr, wf.Name)
			}

			if result.NextCursor != "" {
				fmt.Printf("\nMore results available. Use --cursor %s to continue.\n", result.NextCursor)
			}

			return nil
		},
	}

	cmd.Flags().BoolVar(&active, "active", false, "Show only active workflows")
	cmd.Flags().BoolVar(&inactive, "inactive", false, "Show only inactive workflows")
	cmd.Flags().StringSliceVar(&tags, "tag", nil, "Filter by tag (can be repeated)")
	cmd.Flags().IntVar(&limit, "limit", 100, "Maximum number of workflows to return")
	cmd.Flags().StringVar(&cursor, "cursor", "", "Pagination cursor for next page")

	return cmd
}

func newViewCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "view <workflow-id>",
		Short: "View a workflow's JSON definition",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := getClient()
			if err != nil {
				return err
			}

			wf, err := client.GetWorkflow(args[0])
			if err != nil {
				return fmt.Errorf("failed to get workflow: %w", err)
			}

			return printJSON(wf)
		},
	}
}

func newPullCmd() *cobra.Command {
	var (
		recursive bool
		dir       string
		force     bool
	)

	cmd := &cobra.Command{
		Use:   "pull <workflow-id>",
		Short: "Pull a workflow to local files",
		Long: `Download a workflow JSON to local filesystem.

With --recursive, also downloads all sub-workflows referenced
by Execute Workflow nodes, creating a manifest.json that tracks
the relationships.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := getClient()
			if err != nil {
				return err
			}

			workflowID := args[0]

			// Create output directory if specified
			if dir != "" {
				if err := os.MkdirAll(dir, 0755); err != nil {
					return fmt.Errorf("failed to create directory: %w", err)
				}
			}

			if recursive {
				return pullRecursive(client, workflowID, dir, force)
			}

			// Simple single workflow pull
			wf, err := client.GetWorkflow(workflowID)
			if err != nil {
				return fmt.Errorf("failed to get workflow: %w", err)
			}

			filename := workflow.SanitizeFilename(wf.Name) + ".json"
			if dir != "" {
				filename = filepath.Join(dir, filename)
			}

			if !force {
				if _, err := os.Stat(filename); err == nil {
					return fmt.Errorf("file %s already exists. Use --force to overwrite", filename)
				}
			}

			data, err := json.MarshalIndent(wf, "", "  ")
			if err != nil {
				return fmt.Errorf("failed to marshal workflow: %w", err)
			}

			if err := os.WriteFile(filename, data, 0644); err != nil {
				return fmt.Errorf("failed to write file: %w", err)
			}

			fmt.Printf("Pulled workflow to %s\n", filename)
			return nil
		},
	}

	cmd.Flags().BoolVarP(&recursive, "recursive", "r", false, "Also pull sub-workflows")
	cmd.Flags().StringVarP(&dir, "dir", "d", "", "Output directory")
	cmd.Flags().BoolVarP(&force, "force", "f", false, "Overwrite existing files")

	return cmd
}

func pullRecursive(client *api.Client, workflowID, dir string, force bool) error {
	puller := workflow.NewRecursivePuller(client)
	result, err := puller.Pull(workflowID)
	if err != nil {
		return err
	}

	// Write all workflows
	for id, wf := range result.Workflows {
		filename := workflow.SanitizeFilename(wf.Name) + ".json"
		if dir != "" {
			filename = filepath.Join(dir, filename)
		}

		if !force {
			if _, err := os.Stat(filename); err == nil {
				return fmt.Errorf("file %s already exists. Use --force to overwrite", filename)
			}
		}

		data, err := json.MarshalIndent(wf, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal workflow %s: %w", id, err)
		}

		if err := os.WriteFile(filename, data, 0644); err != nil {
			return fmt.Errorf("failed to write file: %w", err)
		}

		fmt.Printf("Pulled: %s -> %s\n", wf.Name, filename)
	}

	// Write manifest
	manifestPath := "manifest.json"
	if dir != "" {
		manifestPath = filepath.Join(dir, manifestPath)
	}

	manifestData, err := json.MarshalIndent(result.Manifest, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal manifest: %w", err)
	}

	if err := os.WriteFile(manifestPath, manifestData, 0644); err != nil {
		return fmt.Errorf("failed to write manifest: %w", err)
	}

	fmt.Printf("\nPulled %d workflow(s). Manifest: %s\n", len(result.Workflows), manifestPath)
	return nil
}

func newPushCmd() *cobra.Command {
	var (
		create bool
	)

	cmd := &cobra.Command{
		Use:   "push <file-or-directory>",
		Short: "Push workflow(s) to n8n",
		Long: `Upload workflow JSON file(s) to n8n.

If a directory is specified and contains a manifest.json,
all workflows in the manifest will be pushed in the correct order.

By default, updates existing workflows. Use --create to create new ones.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := getClient()
			if err != nil {
				return err
			}

			path := args[0]
			info, err := os.Stat(path)
			if err != nil {
				return fmt.Errorf("failed to access path: %w", err)
			}

			if info.IsDir() {
				return pushDirectory(client, path, create)
			}

			return pushFile(client, path, create)
		},
	}

	cmd.Flags().BoolVar(&create, "create", false, "Create new workflows instead of updating")

	return cmd
}

func pushFile(client *api.Client, path string, create bool) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	var wf api.Workflow
	if err := json.Unmarshal(data, &wf); err != nil {
		return fmt.Errorf("failed to parse workflow JSON: %w", err)
	}

	if create {
		created, err := client.CreateWorkflow(&wf)
		if err != nil {
			return fmt.Errorf("failed to create workflow: %w", err)
		}
		fmt.Printf("Created workflow: %s (ID: %s)\n", created.Name, created.ID)
	} else {
		if wf.ID == "" {
			return fmt.Errorf("workflow has no ID. Use --create to create a new workflow")
		}
		updated, err := client.UpdateWorkflow(wf.ID, &wf)
		if err != nil {
			return fmt.Errorf("failed to update workflow: %w", err)
		}
		fmt.Printf("Updated workflow: %s (ID: %s)\n", updated.Name, updated.ID)
	}

	return nil
}

func pushDirectory(client *api.Client, dir string, create bool) error {
	manifestPath := filepath.Join(dir, "manifest.json")
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		return fmt.Errorf("no manifest.json found in directory")
	}

	var manifest workflow.Manifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return fmt.Errorf("failed to parse manifest: %w", err)
	}

	// Push in dependency order (sub-workflows first)
	pusher := workflow.NewPusher(client, dir)
	if err := pusher.Push(&manifest, create); err != nil {
		return err
	}

	fmt.Printf("\nPushed %d workflow(s) successfully.\n", len(manifest.Workflows))
	return nil
}

func newRunCmd() *cobra.Command {
	var (
		inputJSON string
		wait      bool
	)

	cmd := &cobra.Command{
		Use:   "run <workflow-id>",
		Short: "Execute a workflow",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := getClient()
			if err != nil {
				return err
			}

			var inputData map[string]interface{}
			if inputJSON != "" {
				if err := json.Unmarshal([]byte(inputJSON), &inputData); err != nil {
					return fmt.Errorf("invalid input JSON: %w", err)
				}
			}

			execution, err := client.ExecuteWorkflow(args[0], inputData, wait)
			if err != nil {
				return fmt.Errorf("failed to execute workflow: %w", err)
			}

			jsonFlag, _ := cmd.Flags().GetBool("json")
			if jsonFlag {
				return printJSON(execution)
			}

			fmt.Printf("Execution ID: %s\n", execution.ID)
			fmt.Printf("Status: %s\n", execution.Status)
			if execution.Finished {
				fmt.Printf("Finished: yes\n")
			}

			return nil
		},
	}

	cmd.Flags().StringVarP(&inputJSON, "input", "i", "", "Input data as JSON")
	cmd.Flags().BoolVarP(&wait, "wait", "w", false, "Wait for execution to complete")

	return cmd
}

func newActivateCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "activate <workflow-id>",
		Short: "Activate a workflow",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := getClient()
			if err != nil {
				return err
			}

			if err := client.ActivateWorkflow(args[0]); err != nil {
				return fmt.Errorf("failed to activate workflow: %w", err)
			}

			fmt.Println("Workflow activated.")
			return nil
		},
	}
}

func newDeactivateCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "deactivate <workflow-id>",
		Short: "Deactivate a workflow",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := getClient()
			if err != nil {
				return err
			}

			if err := client.DeactivateWorkflow(args[0]); err != nil {
				return fmt.Errorf("failed to deactivate workflow: %w", err)
			}

			fmt.Println("Workflow deactivated.")
			return nil
		},
	}
}

func boolPtr(b bool) *bool {
	return &b
}

func printJSON(v interface{}) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}
