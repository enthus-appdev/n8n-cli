package user

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/enthus-appdev/n8n-cli/internal/api"
	"github.com/enthus-appdev/n8n-cli/internal/cmd/cmdutil"
)

func NewUserCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "user",
		Short: "Look up n8n users",
		Long:  `List and look up users. Project membership is managed with 'n8nctl project member'.`,
	}

	cmd.AddCommand(newListCmd())
	cmd.AddCommand(newViewCmd())

	return cmd
}

// PrintUsers prints users as a table; role is the global or project role.
func PrintUsers(users []api.User) {
	fmt.Printf("%-36s  %-24s  %-34s  %s\n", "ID", "ROLE", "EMAIL", "NAME")
	for _, u := range users {
		name := strings.TrimSpace(u.FirstName + " " + u.LastName)
		if u.IsPending {
			name += " (pending)"
		}
		fmt.Printf("%-36s  %-24s  %-34s  %s\n", u.ID, u.Role, u.Email, name)
	}
}

func newListCmd() *cobra.Command {
	var (
		limit     int
		cursor    string
		projectID string
	)

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List users with their global role",
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := cmdutil.GetClient()
			if err != nil {
				return err
			}

			result, err := client.ListUsers(limit, cursor, projectID)
			if err != nil {
				return fmt.Errorf("failed to list users: %w", err)
			}

			if cmdutil.JSONOutput(cmd) {
				return cmdutil.PrintJSON(result)
			}

			if len(result.Data) == 0 {
				fmt.Println("No users found.")
				return nil
			}

			PrintUsers(result.Data)
			cmdutil.PrintNextCursor(result.NextCursor)
			return nil
		},
	}

	cmd.Flags().IntVar(&limit, "limit", 100, "Maximum number of users to return (max 250)")
	cmd.Flags().StringVar(&cursor, "cursor", "", "Pagination cursor for next page")
	cmd.Flags().StringVar(&projectID, "project", "", "Only users that are members of this project")

	return cmd
}

func newViewCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "view <user-id-or-email>",
		Short: "Show a user",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := cmdutil.GetClient()
			if err != nil {
				return err
			}

			u, err := client.GetUser(args[0])
			if err != nil {
				return fmt.Errorf("failed to get user: %w", err)
			}

			if cmdutil.JSONOutput(cmd) {
				return cmdutil.PrintJSON(u)
			}

			fmt.Printf("User: %s %s <%s>\n", u.FirstName, u.LastName, u.Email)
			fmt.Printf("ID: %s\n", u.ID)
			fmt.Printf("Role: %s\n", u.Role)
			fmt.Printf("Pending invite: %t\n", u.IsPending)
			fmt.Printf("MFA enabled: %t\n", u.MFAEnabled)
			return nil
		},
	}
}
