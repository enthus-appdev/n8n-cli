package project

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/enthus-appdev/n8n-cli/internal/api"
	"github.com/enthus-appdev/n8n-cli/internal/config"
)

func NewProjectCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "project",
		Short: "Manage n8n projects",
		Long:  `List and manage n8n projects.`,
	}

	cmd.AddCommand(newListCmd())

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
		limit  int
		cursor string
	)

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List all projects",
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := getClient()
			if err != nil {
				return err
			}

			result, err := client.ListProjects(limit, cursor)
			if err != nil {
				return fmt.Errorf("failed to list projects: %w", err)
			}

			jsonFlag, _ := cmd.Flags().GetBool("json")
			if jsonFlag {
				return printJSON(result)
			}

			if len(result.Data) == 0 {
				fmt.Println("No projects found.")
				return nil
			}

			fmt.Printf("%-18s  %-10s  %s\n", "ID", "TYPE", "NAME")
			fmt.Printf("%-18s  %-10s  %s\n", strings.Repeat("-", 18), strings.Repeat("-", 10), strings.Repeat("-", 40))
			for _, p := range result.Data {
				fmt.Printf("%-18s  %-10s  %s\n", p.ID, p.Type, p.Name)
			}

			if result.NextCursor != "" {
				fmt.Printf("\nMore results available. Use --cursor %s to continue.\n", result.NextCursor)
			}

			return nil
		},
	}

	cmd.Flags().IntVar(&limit, "limit", 100, "Maximum number of projects to return")
	cmd.Flags().StringVar(&cursor, "cursor", "", "Pagination cursor for next page")

	return cmd
}

func printJSON(v interface{}) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}
