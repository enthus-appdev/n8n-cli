package audit

import (
	"fmt"
	"slices"
	"strings"

	"github.com/spf13/cobra"

	"github.com/enthus-appdev/n8n-cli/internal/api"
	"github.com/enthus-appdev/n8n-cli/internal/cmd/cmdutil"
)

var categories = []string{"credentials", "database", "nodes", "filesystem", "instance"}

func NewAuditCmd() *cobra.Command {
	var (
		cats          []string
		daysAbandoned int
	)

	cmd := &cobra.Command{
		Use:   "audit",
		Short: "Run n8n's security audit and print the report as JSON",
		Example: `  n8nctl audit
  n8nctl audit --categories credentials,nodes --days-abandoned 30`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			for _, c := range cats {
				if !slices.Contains(categories, c) {
					return fmt.Errorf("invalid --categories value %q (must be one of: %s)", c, strings.Join(categories, ", "))
				}
			}

			client, err := cmdutil.GetClient()
			if err != nil {
				return err
			}

			report, err := client.GenerateAudit(api.AuditOptions{Categories: cats, DaysAbandonedWorkflow: daysAbandoned})
			if err != nil {
				return fmt.Errorf("failed to generate audit: %w", err)
			}

			return cmdutil.PrintJSON(report)
		},
	}

	cmd.Flags().StringSliceVar(&cats, "categories", nil, "Risk categories to audit: "+strings.Join(categories, ", ")+" (default: all)")
	cmd.Flags().IntVar(&daysAbandoned, "days-abandoned", 0, "Days without executions before a workflow counts as abandoned (default: server default)")

	return cmd
}
