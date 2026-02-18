package variable

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/enthus-appdev/n8n-cli/internal/api"
	"github.com/enthus-appdev/n8n-cli/internal/config"
)

func NewVariableCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "variable",
		Aliases: []string{"var"},
		Short:   "Manage n8n variables",
		Long:    `List, create, update, and delete n8n environment variables.`,
	}

	cmd.AddCommand(newListCmd())
	cmd.AddCommand(newGetCmd())
	cmd.AddCommand(newCreateCmd())
	cmd.AddCommand(newUpdateCmd())
	cmd.AddCommand(newDeleteCmd())

	return cmd
}

func getClient() (*api.Client, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, fmt.Errorf("not configured. Run 'n8nctl config init' first")
	}

	instance, err := cfg.GetCurrentInstance()
	if err != nil {
		return nil, err
	}

	return api.NewClient(instance.URL, instance.APIKey), nil
}

func newListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List all variables",
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := getClient()
			if err != nil {
				return err
			}

			vars, err := client.ListVariables(0, "")
			if err != nil {
				return fmt.Errorf("failed to list variables: %w", err)
			}

			jsonFlag, _ := cmd.Flags().GetBool("json")
			if jsonFlag {
				return printJSON(vars)
			}

			if len(vars) == 0 {
				fmt.Println("No variables found.")
				return nil
			}

			fmt.Printf("%-8s  %-40s  %s\n", "ID", "KEY", "VALUE")
			fmt.Printf("%-8s  %-40s  %s\n", strings.Repeat("-", 8), strings.Repeat("-", 40), strings.Repeat("-", 50))
			for _, v := range vars {
				value := v.Value
				if len(value) > 50 {
					value = value[:47] + "..."
				}
				fmt.Printf("%-8s  %-40s  %s\n", v.ID, v.Key, value)
			}

			return nil
		},
	}

	return cmd
}

func newGetCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get <key>",
		Short: "Get a variable by key",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := getClient()
			if err != nil {
				return err
			}

			vars, err := client.ListVariables(0, "")
			if err != nil {
				return fmt.Errorf("failed to list variables: %w", err)
			}

			key := args[0]
			for _, v := range vars {
				if v.Key == key {
					jsonFlag, _ := cmd.Flags().GetBool("json")
					if jsonFlag {
						return printJSON(v)
					}
					fmt.Println(v.Value)
					return nil
				}
			}

			return fmt.Errorf("variable %q not found", key)
		},
	}

	return cmd
}

func newCreateCmd() *cobra.Command {
	var valueFlag string

	cmd := &cobra.Command{
		Use:   "create <key> [value]",
		Short: "Create a new variable",
		Long: `Create a new variable. The value can be passed as a positional argument
or via the --value flag. The flag form is useful for values containing
special shell characters (e.g. n8nctl var create key --value 'b!xyz').`,
		Args: cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := getClient()
			if err != nil {
				return err
			}

			key := args[0]
			var value string
			switch {
			case len(args) == 2:
				value = args[1]
			case valueFlag != "":
				value = valueFlag
			default:
				return fmt.Errorf("value is required: pass as second argument or use --value")
			}

			if err := client.CreateVariable(key, value); err != nil {
				return fmt.Errorf("failed to create variable: %w", err)
			}

			fmt.Printf("Created variable: %s\n", key)
			return nil
		},
	}

	cmd.Flags().StringVar(&valueFlag, "value", "", "Variable value (alternative to positional argument)")

	return cmd
}

func newUpdateCmd() *cobra.Command {
	var valueFlag string

	cmd := &cobra.Command{
		Use:   "update <key> [value]",
		Short: "Update a variable by key",
		Long: `Update a variable. The value can be passed as a positional argument
or via the --value flag. The flag form is useful for values containing
special shell characters (e.g. n8nctl var update key --value 'b!xyz').`,
		Args: cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := getClient()
			if err != nil {
				return err
			}

			key := args[0]
			var value string
			switch {
			case len(args) == 2:
				value = args[1]
			case valueFlag != "":
				value = valueFlag
			default:
				return fmt.Errorf("value is required: pass as second argument or use --value")
			}

			// Resolve key to ID
			vars, err := client.ListVariables(0, "")
			if err != nil {
				return fmt.Errorf("failed to list variables: %w", err)
			}

			for _, v := range vars {
				if v.Key == key {
					if err := client.UpdateVariable(v.ID, key, value); err != nil {
						return fmt.Errorf("failed to update variable: %w", err)
					}
					fmt.Printf("Updated variable: %s\n", key)
					return nil
				}
			}

			return fmt.Errorf("variable %q not found", key)
		},
	}

	cmd.Flags().StringVar(&valueFlag, "value", "", "Variable value (alternative to positional argument)")

	return cmd
}

func newDeleteCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "delete <key>",
		Short: "Delete a variable by key",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := getClient()
			if err != nil {
				return err
			}

			// Resolve key to ID
			vars, err := client.ListVariables(0, "")
			if err != nil {
				return fmt.Errorf("failed to list variables: %w", err)
			}

			key := args[0]
			for _, v := range vars {
				if v.Key == key {
					if err := client.DeleteVariable(v.ID); err != nil {
						return fmt.Errorf("failed to delete variable: %w", err)
					}
					fmt.Printf("Deleted variable: %s\n", key)
					return nil
				}
			}

			return fmt.Errorf("variable %q not found", key)
		},
	}

	return cmd
}

func printJSON(v interface{}) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}
