package sourcecontrol

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"slices"
	"strings"

	"github.com/spf13/cobra"

	"github.com/enthus-appdev/n8n-cli/internal/api"
	"github.com/enthus-appdev/n8n-cli/internal/config"
)

func NewSourceControlCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "source-control",
		Aliases: []string{"sc"},
		Short:   "Push and pull changes through the instance's Git connection",
		Long: `Use n8n's built-in source control (Environments) feature.

push commits selected items to the Git branch connected to the instance.
pull imports that branch into the instance and overwrites local drafts.
Both act on the shared branch that other instances may pull from.`,
	}

	cmd.AddCommand(newStatusCmd())
	cmd.AddCommand(newPushCmd())
	cmd.AddCommand(newPullCmd())

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

func newStatusCmd() *cobra.Command {
	var direction string

	cmd := &cobra.Command{
		Use:   "status",
		Short: "Preview pending source control changes",
		Long: `Preview the changes a push or pull would apply.

Use the IDs from 'status --direction push' as push targets.`,
		Example: `  n8nctl source-control status
  n8nctl sc status --direction pull --json`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if direction != "push" && direction != "pull" {
				return fmt.Errorf("invalid --direction %q (must be push or pull)", direction)
			}

			client, err := getClient()
			if err != nil {
				return err
			}

			files, err := client.SourceControlStatus(direction)
			if err != nil {
				return fmt.Errorf("failed to get source control status: %w", err)
			}

			jsonFlag, _ := cmd.Flags().GetBool("json")
			if jsonFlag {
				return printJSON(files)
			}

			if len(files) == 0 {
				fmt.Printf("No pending %s changes.\n", direction)
				return nil
			}
			printFiles(os.Stdout, files)
			return nil
		},
	}

	cmd.Flags().StringVar(&direction, "direction", "push", "Direction to preview: push or pull")

	return cmd
}

func newPushCmd() *cobra.Command {
	var (
		message   string
		workflows []string
		items     []string
		force     bool
	)

	cmd := &cobra.Command{
		Use:   "push",
		Short: "Commit and push selected items to the connected Git branch",
		Long: `Commit the selected items and push them to the Git branch connected to the instance.

Select items explicitly with --workflow or --item; there is no push-everything
option, so other people's pending changes on a shared instance are not swept in.
Run 'n8nctl source-control status' to list pending items and their IDs.`,
		Example: `  n8nctl source-control push -m "Fix invoice routing" --workflow 1QHLOQM9piQTOb9Y
  n8nctl sc push -m "Rotate SharePoint credential" --item credential:WjE8kQkvvV0zDFrN`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if strings.TrimSpace(message) == "" {
				return fmt.Errorf("a commit message is required (--message)")
			}

			fileNames, err := buildPushItems(workflows, items)
			if err != nil {
				return err
			}

			client, err := getClient()
			if err != nil {
				return err
			}

			pushed, err := client.SourceControlPush(api.SourceControlPushOptions{
				CommitMessage: message,
				FileNames:     fileNames,
				Force:         force,
			})
			if err != nil {
				return conflictError("push", err)
			}

			jsonFlag, _ := cmd.Flags().GetBool("json")
			if jsonFlag {
				return printJSON(pushed)
			}

			fmt.Printf("Pushed %d items.\n", len(pushed))
			if len(pushed) > 0 {
				printFiles(os.Stdout, pushed)
			}
			return nil
		},
	}

	cmd.Flags().StringVarP(&message, "message", "m", "", "Commit message (required)")
	cmd.Flags().StringArrayVar(&workflows, "workflow", nil, "Workflow ID to push (repeatable)")
	cmd.Flags().StringArrayVar(&items, "item", nil, "Item to push as <type>:<id> (repeatable); types: "+strings.Join(api.SourceControlTypes, ", "))
	cmd.Flags().BoolVar(&force, "force", false, "Push even if the selected items conflict with the remote branch")

	return cmd
}

// buildPushItems turns --workflow and --item values into push targets.
func buildPushItems(workflows, items []string) ([]api.SourceControlItem, error) {
	var out []api.SourceControlItem
	for _, id := range workflows {
		if id == "" {
			return nil, fmt.Errorf("--workflow needs a non-empty ID")
		}
		out = append(out, api.SourceControlItem{ID: id, Type: "workflow"})
	}
	for _, item := range items {
		typ, id, ok := strings.Cut(item, ":")
		if !ok || typ == "" || id == "" {
			return nil, fmt.Errorf("invalid --item %q (expected <type>:<id>)", item)
		}
		if !slices.Contains(api.SourceControlTypes, typ) {
			return nil, fmt.Errorf("invalid --item type %q (must be one of: %s)", typ, strings.Join(api.SourceControlTypes, ", "))
		}
		out = append(out, api.SourceControlItem{ID: id, Type: typ})
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("select at least one item with --workflow or --item")
	}
	return out, nil
}

func newPullCmd() *cobra.Command {
	var (
		force       bool
		autoPublish string
	)

	cmd := &cobra.Command{
		Use:   "pull",
		Short: "Import the connected Git branch into the instance",
		Long: `Import the Git branch connected to the instance.

A pull overwrites the drafts of every changed item with the version from Git.
Run 'n8nctl source-control status --direction pull' first to preview it.`,
		Example: `  n8nctl source-control pull
  n8nctl sc pull --auto-publish published`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if autoPublish != "" && autoPublish != "none" && autoPublish != "all" && autoPublish != "published" {
				return fmt.Errorf("invalid --auto-publish %q (must be none, all, or published)", autoPublish)
			}

			client, err := getClient()
			if err != nil {
				return err
			}

			pulled, err := client.SourceControlPull(api.SourceControlPullOptions{
				Force:       force,
				AutoPublish: autoPublish,
			})
			if err != nil {
				return conflictError("pull", err)
			}

			jsonFlag, _ := cmd.Flags().GetBool("json")
			if jsonFlag {
				return printJSON(pulled)
			}

			fmt.Printf("Pulled %d items.\n", len(pulled))
			if len(pulled) > 0 {
				printFiles(os.Stdout, pulled)
			}
			return nil
		},
	}

	cmd.Flags().BoolVar(&force, "force", false, "Discard local changes that conflict with the pull")
	cmd.Flags().StringVar(&autoPublish, "auto-publish", "", "Publish imported workflows: none (server default), all, or published (only those published before the pull)")

	return cmd
}

// conflictError lists the conflicting files of a rejected push or pull and
// points at --force.
func conflictError(action string, err error) error {
	var conflict *api.SourceControlConflictError
	if !errors.As(err, &conflict) {
		return fmt.Errorf("failed to %s: %w", action, err)
	}
	if len(conflict.Conflicts) > 0 {
		fmt.Fprintln(os.Stderr, "Conflicting items:")
		printFiles(os.Stderr, conflict.Conflicts)
	}
	return fmt.Errorf("%s rejected: %w; rerun with --force to override", action, err)
}

func printFiles(w io.Writer, files []api.SourceControlFile) {
	fmt.Fprintf(w, "%-10s  %-10s  %-8s  %-18s  %s\n", "TYPE", "STATUS", "LOCATION", "ID", "NAME")
	fmt.Fprintf(w, "%-10s  %-10s  %-8s  %-18s  %s\n", strings.Repeat("-", 10), strings.Repeat("-", 10), strings.Repeat("-", 8), strings.Repeat("-", 18), strings.Repeat("-", 30))
	for _, f := range files {
		name := f.Name
		if f.Conflict {
			name += " [conflict]"
		}
		fmt.Fprintf(w, "%-10s  %-10s  %-8s  %-18s  %s\n", f.Type, f.Status, f.Location, f.ID, name)
		if f.PublishingError != "" || f.PublishingErrorDetails != nil {
			reason := ""
			if f.PublishingErrorDetails != nil {
				reason = " (" + f.PublishingErrorDetails.Reason + ")"
			}
			fmt.Fprintf(w, "            not published: %s%s\n", f.PublishingError, reason)
		}
	}
}

func printJSON(v interface{}) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}
