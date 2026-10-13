package credential

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/enthus-appdev/n8n-cli/internal/cmd/cmdutil"
)

func NewCredentialCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "credential",
		Aliases: []string{"cred"},
		Short:   "Manage n8n credentials",
		Long:    `List, test, and delete credentials. Secrets are never returned by the API.`,
	}

	cmd.AddCommand(newListCmd())
	cmd.AddCommand(newTestCmd())
	cmd.AddCommand(newSchemaCmd())
	cmd.AddCommand(newDeleteCmd())

	return cmd
}

func newListCmd() *cobra.Command {
	var (
		limit  int
		cursor string
	)

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List credentials",
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := cmdutil.GetClient()
			if err != nil {
				return err
			}

			result, err := client.ListCredentials(limit, cursor)
			if err != nil {
				return fmt.Errorf("failed to list credentials: %w", err)
			}

			if cmdutil.JSONOutput(cmd) {
				return cmdutil.PrintJSON(result)
			}

			if len(result.Data) == 0 {
				fmt.Println("No credentials found.")
				return nil
			}

			fmt.Printf("%-18s  %-28s  %-30s  %s\n", "ID", "TYPE", "PROJECT", "NAME")
			for _, c := range result.Data {
				var projects []string
				for _, s := range c.Shared {
					projects = append(projects, s.Name)
				}
				fmt.Printf("%-18s  %-28s  %-30s  %s\n", c.ID, c.Type, strings.Join(projects, ", "), c.Name)
			}
			cmdutil.PrintNextCursor(result.NextCursor)
			return nil
		},
	}

	cmd.Flags().IntVar(&limit, "limit", 100, "Maximum number of credentials to return (max 250)")
	cmd.Flags().StringVar(&cursor, "cursor", "", "Pagination cursor for next page")

	return cmd
}

func newTestCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "test <credential-id>",
		Short: "Test a credential against its service",
		Long:  `Test a credential. Exits non-zero when the test fails.`,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := cmdutil.GetClient()
			if err != nil {
				return err
			}

			result, err := client.TestCredential(args[0])
			if err != nil {
				return fmt.Errorf("failed to test credential: %w", err)
			}

			if cmdutil.JSONOutput(cmd) {
				if err := cmdutil.PrintJSON(result); err != nil {
					return err
				}
			} else {
				fmt.Printf("%s: %s\n", result.Status, result.Message)
			}
			if result.Status != "OK" {
				return fmt.Errorf("credential test failed")
			}
			return nil
		},
	}
}

func newSchemaCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "schema <credential-type>",
		Short: "Show the JSON schema of a credential type's data",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := cmdutil.GetClient()
			if err != nil {
				return err
			}

			schema, err := client.GetCredentialSchema(args[0])
			if err != nil {
				return fmt.Errorf("failed to get credential schema: %w", err)
			}

			return cmdutil.PrintJSON(schema)
		},
	}
}

func newDeleteCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "delete <credential-id>",
		Short: "Delete a credential",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := cmdutil.GetClient()
			if err != nil {
				return err
			}

			if err := client.DeleteCredential(args[0]); err != nil {
				return fmt.Errorf("failed to delete credential: %w", err)
			}

			fmt.Println("Credential deleted.")
			return nil
		},
	}
}
