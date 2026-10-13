package datatable

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/enthus-appdev/n8n-cli/internal/api"
	"github.com/enthus-appdev/n8n-cli/internal/cmd/cmdutil"
)

func NewDataTableCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "datatable",
		Aliases: []string{"dt"},
		Short:   "Read and write n8n data tables",
		Long: `List data tables and read, insert, update, upsert, and delete rows.

Select rows with repeatable --where <column><op><value> conditions, where op is
one of = != > >= < <= ~ (~ is a case-insensitive LIKE; use % as wildcard).
Conditions are ANDed unless --or is set. For typed values or other shapes,
pass the API filter object with --filter instead.`,
	}

	cmd.AddCommand(newListCmd())
	cmd.AddCommand(newViewCmd())
	cmd.AddCommand(newRowsCmd())
	cmd.AddCommand(newInsertCmd())
	cmd.AddCommand(newUpdateCmd())
	cmd.AddCommand(newUpsertCmd())
	cmd.AddCommand(newDeleteRowsCmd())

	return cmd
}

type filterFlags struct {
	wheres []string
	filter string
	or     bool
}

func (f *filterFlags) register(cmd *cobra.Command) {
	cmd.Flags().StringArrayVar(&f.wheres, "where", nil, "Row condition <column><op><value> (repeatable)")
	cmd.Flags().StringVar(&f.filter, "filter", "", `Raw filter JSON, e.g. '{"type":"and","filters":[{"columnName":"x","condition":"eq","value":1}]}'`)
	cmd.Flags().BoolVar(&f.or, "or", false, "Match rows meeting any --where condition instead of all")
}

func (f *filterFlags) build(required bool) (*api.DataTableFilter, error) {
	filter, err := buildFilter(f.wheres, f.filter, f.or)
	if err != nil {
		return nil, err
	}
	if filter == nil && required {
		return nil, fmt.Errorf("select rows with --where or --filter")
	}
	return filter, nil
}

func printRaw(raw json.RawMessage) error {
	var v interface{}
	if err := json.Unmarshal(raw, &v); err != nil {
		fmt.Println(string(raw))
		return nil
	}
	return cmdutil.PrintJSON(v)
}

func newListCmd() *cobra.Command {
	var (
		limit  int
		cursor string
	)

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List data tables",
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := cmdutil.GetClient()
			if err != nil {
				return err
			}

			result, err := client.ListDataTables(limit, cursor)
			if err != nil {
				return fmt.Errorf("failed to list data tables: %w", err)
			}

			if cmdutil.JSONOutput(cmd) {
				return cmdutil.PrintJSON(result)
			}

			if len(result.Data) == 0 {
				fmt.Println("No data tables found.")
				return nil
			}

			fmt.Printf("%-18s  %-18s  %-8s  %s\n", "ID", "PROJECT", "COLUMNS", "NAME")
			for _, t := range result.Data {
				fmt.Printf("%-18s  %-18s  %-8d  %s\n", t.ID, t.ProjectID, len(t.Columns), t.Name)
			}
			cmdutil.PrintNextCursor(result.NextCursor)
			return nil
		},
	}

	cmd.Flags().IntVar(&limit, "limit", 100, "Maximum number of tables to return (max 250)")
	cmd.Flags().StringVar(&cursor, "cursor", "", "Pagination cursor for next page")

	return cmd
}

func newViewCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "view <table-id>",
		Short: "Show a data table and its columns",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := cmdutil.GetClient()
			if err != nil {
				return err
			}

			t, err := client.GetDataTable(args[0])
			if err != nil {
				return fmt.Errorf("failed to get data table: %w", err)
			}

			if cmdutil.JSONOutput(cmd) {
				return cmdutil.PrintJSON(t)
			}

			fmt.Printf("Data table: %s (%s)\n", t.Name, t.ID)
			fmt.Printf("Project: %s\n", t.ProjectID)
			fmt.Printf("Size: %d bytes\n", t.SizeBytes)
			fmt.Println("Columns:")
			for _, c := range t.Columns {
				fmt.Printf("  %-30s  %s\n", c.Name, c.Type)
			}
			return nil
		},
	}
}

func newRowsCmd() *cobra.Command {
	var (
		ff     filterFlags
		search string
		sortBy string
		limit  int
		cursor string
	)

	cmd := &cobra.Command{
		Use:   "rows <table-id>",
		Short: "List rows as a JSON array (--json adds the pagination cursor)",
		Example: `  n8nctl dt rows m7h0yVfJWk2hr9Mv --where 'displayname=Jane Doe' --limit 10
  n8nctl dt rows m7h0yVfJWk2hr9Mv --where 'requestduration>1000' --sort requestduration:desc
  n8nctl dt rows m7h0yVfJWk2hr9Mv --where 'input~%invoice%' --where 'input~%rechnung%' --or`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			filter, err := ff.build(false)
			if err != nil {
				return err
			}

			client, err := cmdutil.GetClient()
			if err != nil {
				return err
			}

			result, err := client.ListDataTableRows(args[0], api.ListRowsOptions{
				Filter: filter, Search: search, SortBy: sortBy, Limit: limit, Cursor: cursor,
			})
			if err != nil {
				return fmt.Errorf("failed to list rows: %w", err)
			}

			if cmdutil.JSONOutput(cmd) {
				return cmdutil.PrintJSON(result)
			}
			if err := cmdutil.PrintJSON(result.Data); err != nil {
				return err
			}
			if result.NextCursor != "" {
				fmt.Fprintf(os.Stderr, "More rows available. Use --cursor %s to continue.\n", result.NextCursor)
			}
			return nil
		},
	}

	ff.register(cmd)
	cmd.Flags().StringVar(&search, "search", "", "Text to search for in all string columns")
	cmd.Flags().StringVar(&sortBy, "sort", "", "Sort as <column>:asc or <column>:desc")
	cmd.Flags().IntVar(&limit, "limit", 100, "Maximum number of rows to return (max 250)")
	cmd.Flags().StringVar(&cursor, "cursor", "", "Pagination cursor for next page")

	return cmd
}

func newInsertCmd() *cobra.Command {
	var data string

	cmd := &cobra.Command{
		Use:   "insert <table-id>",
		Short: "Insert one or more rows",
		Example: `  n8nctl dt insert <table-id> --data '{"name":"a","count":1}'
  n8nctl dt insert <table-id> --data '[{"name":"a"},{"name":"b"}]'
  jq -c '.' rows.json | n8nctl dt insert <table-id> --data -`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			raw, err := readData(data, stdin)
			if err != nil {
				return err
			}
			rows, err := parseRows(raw)
			if err != nil {
				return err
			}

			client, err := cmdutil.GetClient()
			if err != nil {
				return err
			}

			inserted, err := client.InsertDataTableRows(args[0], rows)
			if err != nil {
				return fmt.Errorf("failed to insert rows: %w", err)
			}
			return printRaw(inserted)
		},
	}

	cmd.Flags().StringVar(&data, "data", "", "Row object or array of row objects as JSON, or - to read stdin")

	return cmd
}

func newWriteCmd(use, short, example string, call func(*api.Client, string, api.RowWriteOptions) (json.RawMessage, error)) *cobra.Command {
	var (
		ff     filterFlags
		data   string
		dryRun bool
	)

	cmd := &cobra.Command{
		Use:     use + " <table-id>",
		Short:   short,
		Example: example,
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			filter, err := ff.build(true)
			if err != nil {
				return err
			}
			raw, err := readData(data, stdin)
			if err != nil {
				return err
			}
			row, err := parseRow(raw)
			if err != nil {
				return err
			}

			client, err := cmdutil.GetClient()
			if err != nil {
				return err
			}

			result, err := call(client, args[0], api.RowWriteOptions{Filter: *filter, Data: row, DryRun: dryRun})
			if err != nil {
				return fmt.Errorf("failed to %s rows: %w", use, err)
			}
			return printRaw(result)
		},
	}

	ff.register(cmd)
	cmd.Flags().StringVar(&data, "data", "", "Column values as a JSON object, or - to read stdin")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Show the rows that would change without writing")

	return cmd
}

func newUpdateCmd() *cobra.Command {
	return newWriteCmd("update", "Update every row matching the filter",
		`  n8nctl dt update <table-id> --where 'status=pending' --data '{"status":"done"}' --dry-run`,
		(*api.Client).UpdateDataTableRows)
}

func newUpsertCmd() *cobra.Command {
	return newWriteCmd("upsert", "Update the rows matching the filter, or insert one if none match",
		`  n8nctl dt upsert <table-id> --where 'cache_key=dashboard' --data '{"cache_key":"dashboard","payload":"..."}'`,
		(*api.Client).UpsertDataTableRow)
}

func newDeleteRowsCmd() *cobra.Command {
	var (
		ff     filterFlags
		dryRun bool
	)

	cmd := &cobra.Command{
		Use:   "delete-rows <table-id>",
		Short: "Delete the rows matching the filter",
		Long: `Delete the rows matching --where or --filter and print them.

A filter is required; this command cannot clear a whole table.`,
		Example: `  n8nctl dt delete-rows <table-id> --where 'status=obsolete' --dry-run`,
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			filter, err := ff.build(true)
			if err != nil {
				return err
			}

			client, err := cmdutil.GetClient()
			if err != nil {
				return err
			}

			deleted, err := client.DeleteDataTableRows(args[0], api.RowWriteOptions{Filter: *filter, DryRun: dryRun})
			if err != nil {
				return fmt.Errorf("failed to delete rows: %w", err)
			}
			return printRaw(deleted)
		},
	}

	ff.register(cmd)
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Show the rows that would be deleted without deleting")

	return cmd
}
