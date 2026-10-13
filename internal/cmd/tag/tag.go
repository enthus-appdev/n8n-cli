package tag

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/enthus-appdev/n8n-cli/internal/cmd/cmdutil"
)

func NewTagCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "tag",
		Short: "Manage workflow tags",
	}

	cmd.AddCommand(newListCmd())
	cmd.AddCommand(newCreateCmd())
	cmd.AddCommand(newRenameCmd())
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
		Short: "List tags",
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := cmdutil.GetClient()
			if err != nil {
				return err
			}

			result, err := client.ListTagsPage(limit, cursor)
			if err != nil {
				return fmt.Errorf("failed to list tags: %w", err)
			}

			if cmdutil.JSONOutput(cmd) {
				return cmdutil.PrintJSON(result)
			}

			if len(result.Data) == 0 {
				fmt.Println("No tags found.")
				return nil
			}

			fmt.Printf("%-18s  %s\n", "ID", "NAME")
			for _, t := range result.Data {
				fmt.Printf("%-18s  %s\n", t.ID, t.Name)
			}
			cmdutil.PrintNextCursor(result.NextCursor)
			return nil
		},
	}

	cmd.Flags().IntVar(&limit, "limit", 100, "Maximum number of tags to return (max 250)")
	cmd.Flags().StringVar(&cursor, "cursor", "", "Pagination cursor for next page")

	return cmd
}

func newCreateCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "create <name>",
		Short: "Create a tag",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := cmdutil.GetClient()
			if err != nil {
				return err
			}

			tag, err := client.CreateTag(args[0])
			if err != nil {
				return fmt.Errorf("failed to create tag: %w", err)
			}

			if cmdutil.JSONOutput(cmd) {
				return cmdutil.PrintJSON(tag)
			}
			fmt.Printf("Created tag %s (%s).\n", tag.Name, tag.ID)
			return nil
		},
	}
}

func newRenameCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "rename <tag-id> <new-name>",
		Short: "Rename a tag",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := cmdutil.GetClient()
			if err != nil {
				return err
			}

			tag, err := client.UpdateTag(args[0], args[1])
			if err != nil {
				return fmt.Errorf("failed to rename tag: %w", err)
			}

			if cmdutil.JSONOutput(cmd) {
				return cmdutil.PrintJSON(tag)
			}
			fmt.Printf("Renamed tag %s to %s.\n", tag.ID, tag.Name)
			return nil
		},
	}
}

func newDeleteCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "delete <tag-id>",
		Short: "Delete a tag (removes it from every workflow)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := cmdutil.GetClient()
			if err != nil {
				return err
			}

			if err := client.DeleteTag(args[0]); err != nil {
				return fmt.Errorf("failed to delete tag: %w", err)
			}

			fmt.Println("Tag deleted.")
			return nil
		},
	}
}
