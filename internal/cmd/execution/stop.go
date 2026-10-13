package execution

import (
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/enthus-appdev/n8n-cli/internal/api"
	"github.com/enthus-appdev/n8n-cli/internal/cmd/cmdutil"
)

var stoppableStatuses = []string{"queued", "running", "waiting"}

func newStopCmd() *cobra.Command {
	var (
		statuses      []string
		workflowID    string
		startedAfter  string
		startedBefore string
	)

	cmd := &cobra.Command{
		Use:   "stop [execution-id]",
		Short: "Stop one execution, or every execution matching filters",
		Long: `Stop a single execution by ID, or stop in bulk by status.

Bulk stop needs --status and can be narrowed with --workflow and a start-time
window (RFC 3339 timestamps).`,
		Example: `  n8nctl execution stop 12345
  n8nctl execution stop --status running,waiting --workflow 1QHLOQM9piQTOb9Y
  n8nctl execution stop --status queued --started-before 2026-10-13T08:00:00Z`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			bulk := len(statuses) > 0 || workflowID != "" || startedAfter != "" || startedBefore != ""
			if len(args) == 1 && bulk {
				return fmt.Errorf("pass either an execution ID or bulk filters, not both")
			}
			if len(args) == 0 && len(statuses) == 0 {
				return fmt.Errorf("pass an execution ID, or --status for a bulk stop")
			}
			for flag, v := range map[string]string{"--started-after": startedAfter, "--started-before": startedBefore} {
				if v == "" {
					continue
				}
				if _, err := time.Parse(time.RFC3339, v); err != nil {
					return fmt.Errorf("invalid %s %q (use RFC 3339, e.g. 2026-10-13T08:00:00Z)", flag, v)
				}
			}
			for _, s := range statuses {
				if !slices.Contains(stoppableStatuses, s) {
					return fmt.Errorf("invalid --status %q (must be one of: %s)", s, strings.Join(stoppableStatuses, ", "))
				}
			}

			client, err := cmdutil.GetClient()
			if err != nil {
				return err
			}

			if len(args) == 1 {
				exec, err := client.StopExecution(args[0])
				if err != nil {
					return fmt.Errorf("failed to stop execution: %w", err)
				}
				if cmdutil.JSONOutput(cmd) {
					return cmdutil.PrintJSON(exec)
				}
				fmt.Printf("Stop requested for execution %s; status is now %s.\n", args[0], exec.Status)
				return nil
			}

			stopped, err := client.StopExecutions(api.StopExecutionsOptions{
				Status:        statuses,
				WorkflowID:    workflowID,
				StartedAfter:  startedAfter,
				StartedBefore: startedBefore,
			})
			if err != nil {
				return fmt.Errorf("failed to stop executions: %w", err)
			}
			if cmdutil.JSONOutput(cmd) {
				return cmdutil.PrintJSON(map[string]int{"stopped": stopped})
			}
			fmt.Printf("Stopped %d executions.\n", stopped)
			return nil
		},
	}

	cmd.Flags().StringSliceVar(&statuses, "status", nil, "Bulk stop executions with these statuses: "+strings.Join(stoppableStatuses, ", "))
	cmd.Flags().StringVar(&workflowID, "workflow", "", "Bulk stop only executions of this workflow")
	cmd.Flags().StringVar(&startedAfter, "started-after", "", "Bulk stop only executions started after this time (RFC 3339)")
	cmd.Flags().StringVar(&startedBefore, "started-before", "", "Bulk stop only executions started before this time (RFC 3339)")

	return cmd
}
