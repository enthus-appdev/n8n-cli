package workflow

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/enthus-appdev/n8n-cli/internal/api"
	"github.com/enthus-appdev/n8n-cli/internal/cmd/cmdutil"
)

func newStateCmd(use, short, done string, call func(*api.Client, string) (*api.WorkflowState, error)) *cobra.Command {
	return &cobra.Command{
		Use:   use + " <workflow-id>",
		Short: short,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := cmdutil.GetClient()
			if err != nil {
				return err
			}

			state, err := call(client, args[0])
			if err != nil {
				return fmt.Errorf("failed to %s workflow: %w", use, err)
			}

			if cmdutil.JSONOutput(cmd) {
				return cmdutil.PrintJSON(state)
			}
			fmt.Println(done)
			return nil
		},
	}
}

func newArchiveCmd() *cobra.Command {
	return newStateCmd("archive", "Archive a workflow (soft delete; deactivates it)", "Workflow archived.", (*api.Client).ArchiveWorkflow)
}

func newUnarchiveCmd() *cobra.Command {
	return newStateCmd("unarchive", "Restore an archived workflow", "Workflow unarchived.", (*api.Client).UnarchiveWorkflow)
}

func newUnpublishCmd() *cobra.Command {
	return newStateCmd("unpublish", "Unpublish a workflow so its triggers stop", "Workflow unpublished.", (*api.Client).UnpublishWorkflow)
}

func newPublishCmd() *cobra.Command {
	var versionID string

	cmd := &cobra.Command{
		Use:   "publish <workflow-id>",
		Short: "Publish the current draft, or a specific version, of a workflow",
		Long: `Publish a workflow version so it becomes the version that runs.

Without --version the current draft is published. Use 'workflow view' to see
whether the draft differs from the published version, and 'workflow history'
to list version IDs.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := cmdutil.GetClient()
			if err != nil {
				return err
			}

			state, err := client.PublishWorkflow(args[0], versionID)
			if err != nil {
				return fmt.Errorf("failed to publish workflow: %w", err)
			}

			if cmdutil.JSONOutput(cmd) {
				return cmdutil.PrintJSON(state)
			}
			if state.ActiveVersionID != nil {
				fmt.Printf("Workflow published (version %s).\n", *state.ActiveVersionID)
			} else {
				fmt.Println("Workflow published.")
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&versionID, "version", "", "Version ID to publish (default: current draft)")

	return cmd
}

func newDeleteCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "delete <workflow-id>",
		Short: "Permanently delete a workflow",
		Long: `Permanently delete a workflow and its version history.

A published workflow must be unpublished or archived first. Use
'workflow archive' alone for a reversible soft delete.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := cmdutil.GetClient()
			if err != nil {
				return err
			}

			if err := client.DeleteWorkflow(args[0]); err != nil {
				return fmt.Errorf("failed to delete workflow: %w", err)
			}

			fmt.Println("Workflow deleted.")
			return nil
		},
	}
}

func newHistoryCmd() *cobra.Command {
	var (
		limit  int
		cursor string
	)

	cmd := &cobra.Command{
		Use:   "history <workflow-id>",
		Short: "List a workflow's saved versions, newest first",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := cmdutil.GetClient()
			if err != nil {
				return err
			}

			result, err := client.ListWorkflowHistory(args[0], limit, cursor)
			if err != nil {
				return fmt.Errorf("failed to get workflow history: %w", err)
			}

			if cmdutil.JSONOutput(cmd) {
				return cmdutil.PrintJSON(result)
			}

			if len(result.Data) == 0 {
				fmt.Println("No versions found.")
				return nil
			}

			fmt.Printf("%-36s  %-19s  %s\n", "VERSION", "CREATED", "AUTHORS")
			for _, v := range result.Data {
				label := v.Authors
				if v.Name != nil && *v.Name != "" {
					label += " (" + *v.Name + ")"
				}
				fmt.Printf("%-36s  %-19s  %s\n", v.VersionID, cmdutil.FormatTime(v.CreatedAt), label)
			}
			cmdutil.PrintNextCursor(result.NextCursor)
			return nil
		},
	}

	cmd.Flags().IntVar(&limit, "limit", 20, "Maximum number of versions to return (max 250)")
	cmd.Flags().StringVar(&cursor, "cursor", "", "Pagination cursor for next page")

	return cmd
}

func newVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version <workflow-id> <version-id>",
		Short: "Show one saved version of a workflow as JSON",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := cmdutil.GetClient()
			if err != nil {
				return err
			}

			v, err := client.GetWorkflowVersion(args[0], args[1])
			if err != nil {
				return fmt.Errorf("failed to get workflow version: %w", err)
			}

			return cmdutil.PrintJSON(v)
		},
	}
}
