package folder

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/enthus-appdev/n8n-cli/internal/cmd/cmdutil"
)

func NewFolderCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "folder",
		Short: "Manage workflow folders in a project",
		Long:  `List, create, rename, move, and delete workflow folders. Every subcommand needs --project.`,
	}

	cmd.PersistentFlags().String("project", "", "Project ID (required; see 'n8nctl project list')")
	_ = cmd.MarkPersistentFlagRequired("project")

	cmd.AddCommand(newListCmd())
	cmd.AddCommand(newViewCmd())
	cmd.AddCommand(newCreateCmd())
	cmd.AddCommand(newUpdateCmd())
	cmd.AddCommand(newDeleteCmd())

	return cmd
}

func projectID(cmd *cobra.Command) string {
	p, _ := cmd.Flags().GetString("project")
	return p
}

func parentName(id *string) string {
	if id == nil {
		return "-"
	}
	return *id
}

func newListCmd() *cobra.Command {
	var skip, take int

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List the folders of a project",
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := cmdutil.GetClient()
			if err != nil {
				return err
			}

			folders, count, err := client.ListFolders(projectID(cmd), skip, take)
			if err != nil {
				return fmt.Errorf("failed to list folders: %w", err)
			}

			if cmdutil.JSONOutput(cmd) {
				return cmdutil.PrintJSON(map[string]interface{}{"count": count, "data": folders})
			}

			if len(folders) == 0 {
				fmt.Println("No folders found.")
				return nil
			}

			fmt.Printf("%-18s  %-18s  %s\n", "ID", "PARENT", "NAME")
			for _, f := range folders {
				fmt.Printf("%-18s  %-18s  %s\n", f.ID, parentName(f.ParentFolderID), f.Name)
			}
			if shown := skip + len(folders); shown < count {
				fmt.Printf("\nShowing %d of %d. Use --skip %d to continue.\n", len(folders), count, shown)
			}
			return nil
		},
	}

	cmd.Flags().IntVar(&skip, "skip", 0, "Number of folders to skip")
	cmd.Flags().IntVar(&take, "take", 0, "Maximum number of folders to return (default: server default)")

	return cmd
}

func newViewCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "view <folder-id>",
		Short: "Show a folder with its sub-folder and workflow counts",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := cmdutil.GetClient()
			if err != nil {
				return err
			}

			f, err := client.GetFolder(projectID(cmd), args[0])
			if err != nil {
				return fmt.Errorf("failed to get folder: %w", err)
			}

			if cmdutil.JSONOutput(cmd) {
				return cmdutil.PrintJSON(f)
			}

			fmt.Printf("Folder: %s (%s)\n", f.Name, f.ID)
			fmt.Printf("Parent: %s\n", parentName(f.ParentFolderID))
			if f.TotalSubFolders != nil {
				fmt.Printf("Sub-folders: %d\n", *f.TotalSubFolders)
			}
			if f.TotalWorkflows != nil {
				fmt.Printf("Workflows: %d\n", *f.TotalWorkflows)
			}
			return nil
		},
	}
}

func newCreateCmd() *cobra.Command {
	var parent string

	cmd := &cobra.Command{
		Use:   "create <name>",
		Short: "Create a folder",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := cmdutil.GetClient()
			if err != nil {
				return err
			}

			f, err := client.CreateFolder(projectID(cmd), args[0], parent)
			if err != nil {
				return fmt.Errorf("failed to create folder: %w", err)
			}

			if cmdutil.JSONOutput(cmd) {
				return cmdutil.PrintJSON(f)
			}
			fmt.Printf("Created folder %s (%s).\n", f.Name, f.ID)
			return nil
		},
	}

	cmd.Flags().StringVar(&parent, "parent", "", "Parent folder ID (default: project root)")

	return cmd
}

func newUpdateCmd() *cobra.Command {
	var name, parent string

	cmd := &cobra.Command{
		Use:   "update <folder-id>",
		Short: "Rename a folder or move it under another folder",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if name == "" && parent == "" {
				return fmt.Errorf("pass --name and/or --parent")
			}

			client, err := cmdutil.GetClient()
			if err != nil {
				return err
			}

			f, err := client.UpdateFolder(projectID(cmd), args[0], name, parent)
			if err != nil {
				return fmt.Errorf("failed to update folder: %w", err)
			}

			if cmdutil.JSONOutput(cmd) {
				return cmdutil.PrintJSON(f)
			}
			fmt.Println("Folder updated.")
			return nil
		},
	}

	cmd.Flags().StringVar(&name, "name", "", "New folder name")
	cmd.Flags().StringVar(&parent, "parent", "", "New parent folder ID")

	return cmd
}

func newDeleteCmd() *cobra.Command {
	var transferTo string

	cmd := &cobra.Command{
		Use:   "delete <folder-id>",
		Short: "Delete a folder",
		Long: `Delete a folder.

With --transfer-to, its workflows and sub-folders move to that folder first.
Without it, its workflows move to the project root and are ARCHIVED, and its
sub-folders are deleted.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := cmdutil.GetClient()
			if err != nil {
				return err
			}

			if err := client.DeleteFolder(projectID(cmd), args[0], transferTo); err != nil {
				return fmt.Errorf("failed to delete folder: %w", err)
			}

			fmt.Println("Folder deleted.")
			return nil
		},
	}

	cmd.Flags().StringVar(&transferTo, "transfer-to", "", "Folder ID that receives the deleted folder's contents")

	return cmd
}
