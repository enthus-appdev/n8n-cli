package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	configcmd "github.com/enthus-appdev/n8n-cli/internal/cmd/config"
	executioncmd "github.com/enthus-appdev/n8n-cli/internal/cmd/execution"
	projectcmd "github.com/enthus-appdev/n8n-cli/internal/cmd/project"
	workflowcmd "github.com/enthus-appdev/n8n-cli/internal/cmd/workflow"
)

var (
	version = "dev"
	jsonOutput bool
)

var rootCmd = &cobra.Command{
	Use:   "n8nctl",
	Short: "CLI for interacting with n8n workflow automation",
	Long: `n8n-cli is a command-line tool for managing n8n workflows.

It allows you to list, view, pull, push, and execute workflows
directly from your terminal - perfect for version control,
automation, and LLM-assisted workflow development.`,
	SilenceUsage: true,
}

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.PersistentFlags().BoolVar(&jsonOutput, "json", false, "Output in JSON format")

	rootCmd.AddCommand(configcmd.NewConfigCmd())
	rootCmd.AddCommand(workflowcmd.NewWorkflowCmd())
	rootCmd.AddCommand(executioncmd.NewExecutionCmd())
	rootCmd.AddCommand(projectcmd.NewProjectCmd())
	rootCmd.AddCommand(newVersionCmd())
}

func newVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print the version number",
		Run: func(cmd *cobra.Command, args []string) {
			if jsonOutput {
				out, _ := json.Marshal(map[string]string{"version": version})
				fmt.Println(string(out))
			} else {
				fmt.Printf("n8n-cli %s\n", version)
			}
		},
	}
}

// IsJSONOutput returns whether JSON output is enabled
func IsJSONOutput() bool {
	return jsonOutput
}

// PrintJSON outputs data as formatted JSON
func PrintJSON(v interface{}) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}

// PrintError outputs an error in the appropriate format
func PrintError(err error) {
	if jsonOutput {
		PrintJSON(map[string]string{"error": err.Error()})
	} else {
		fmt.Fprintf(os.Stderr, "Error: %s\n", err)
	}
}
