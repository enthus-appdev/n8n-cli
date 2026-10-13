package project

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/enthus-appdev/n8n-cli/internal/api"
	"github.com/enthus-appdev/n8n-cli/internal/cmd/cmdutil"
	"github.com/enthus-appdev/n8n-cli/internal/cmd/user"
)

func newMemberCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "member",
		Short: "Manage project members",
		Long: `List, add, and remove project members and change their project role.

Project roles look like project:admin, project:editor, or project:viewer.`,
	}

	cmd.AddCommand(newMemberListCmd())
	cmd.AddCommand(newMemberAddCmd())
	cmd.AddCommand(newMemberSetRoleCmd())
	cmd.AddCommand(newMemberRemoveCmd())

	return cmd
}

func newMemberListCmd() *cobra.Command {
	var (
		limit  int
		cursor string
	)

	cmd := &cobra.Command{
		Use:   "list <project-id>",
		Short: "List a project's members with their project role",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := cmdutil.GetClient()
			if err != nil {
				return err
			}

			result, err := client.ListProjectMembers(args[0], limit, cursor)
			if err != nil {
				return fmt.Errorf("failed to list project members: %w", err)
			}

			if cmdutil.JSONOutput(cmd) {
				return cmdutil.PrintJSON(result)
			}

			if len(result.Data) == 0 {
				fmt.Println("No members found.")
				return nil
			}

			user.PrintUsers(result.Data)
			cmdutil.PrintNextCursor(result.NextCursor)
			return nil
		},
	}

	cmd.Flags().IntVar(&limit, "limit", 100, "Maximum number of members to return (max 250)")
	cmd.Flags().StringVar(&cursor, "cursor", "", "Pagination cursor for next page")

	return cmd
}

func newMemberAddCmd() *cobra.Command {
	var role string

	cmd := &cobra.Command{
		Use:   "add <project-id> <user-id>...",
		Short: "Add users to a project",
		Args:  cobra.MinimumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := cmdutil.GetClient()
			if err != nil {
				return err
			}

			relations := make([]api.ProjectRelation, 0, len(args)-1)
			for _, id := range args[1:] {
				relations = append(relations, api.ProjectRelation{UserID: id, Role: role})
			}

			if err := client.AddProjectMembers(args[0], relations); err != nil {
				return fmt.Errorf("failed to add project members: %w", err)
			}

			fmt.Printf("Added %d members as %s.\n", len(relations), role)
			return nil
		},
	}

	cmd.Flags().StringVar(&role, "role", "", "Project role for the added users, e.g. project:editor (required)")
	_ = cmd.MarkFlagRequired("role")

	return cmd
}

func newMemberSetRoleCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "set-role <project-id> <user-id> <role>",
		Short: "Change a member's project role",
		Args:  cobra.ExactArgs(3),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := cmdutil.GetClient()
			if err != nil {
				return err
			}

			if err := client.SetProjectMemberRole(args[0], args[1], args[2]); err != nil {
				return fmt.Errorf("failed to change project role: %w", err)
			}

			fmt.Println("Project role changed.")
			return nil
		},
	}
}

func newMemberRemoveCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "remove <project-id> <user-id>",
		Short: "Remove a user from a project",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := cmdutil.GetClient()
			if err != nil {
				return err
			}

			if err := client.RemoveProjectMember(args[0], args[1]); err != nil {
				return fmt.Errorf("failed to remove project member: %w", err)
			}

			fmt.Println("Project member removed.")
			return nil
		},
	}
}
