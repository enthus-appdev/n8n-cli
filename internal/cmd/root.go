package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"runtime/debug"

	"github.com/spf13/cobra"

	auditcmd "github.com/enthus-appdev/n8n-cli/internal/cmd/audit"
	configcmd "github.com/enthus-appdev/n8n-cli/internal/cmd/config"
	credentialcmd "github.com/enthus-appdev/n8n-cli/internal/cmd/credential"
	datatablecmd "github.com/enthus-appdev/n8n-cli/internal/cmd/datatable"
	executioncmd "github.com/enthus-appdev/n8n-cli/internal/cmd/execution"
	foldercmd "github.com/enthus-appdev/n8n-cli/internal/cmd/folder"
	projectcmd "github.com/enthus-appdev/n8n-cli/internal/cmd/project"
	sourcecontrolcmd "github.com/enthus-appdev/n8n-cli/internal/cmd/sourcecontrol"
	tagcmd "github.com/enthus-appdev/n8n-cli/internal/cmd/tag"
	usercmd "github.com/enthus-appdev/n8n-cli/internal/cmd/user"
	variablecmd "github.com/enthus-appdev/n8n-cli/internal/cmd/variable"
	workflowcmd "github.com/enthus-appdev/n8n-cli/internal/cmd/workflow"
)

var (
	version    string
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

func Execute(ver string) error {
	version = ver
	return rootCmd.Execute()
}

func init() {
	rootCmd.PersistentFlags().BoolVar(&jsonOutput, "json", false, "Output in JSON format")

	rootCmd.AddCommand(configcmd.NewConfigCmd())
	rootCmd.AddCommand(workflowcmd.NewWorkflowCmd())
	rootCmd.AddCommand(executioncmd.NewExecutionCmd())
	rootCmd.AddCommand(projectcmd.NewProjectCmd())
	rootCmd.AddCommand(variablecmd.NewVariableCmd())
	rootCmd.AddCommand(sourcecontrolcmd.NewSourceControlCmd())
	rootCmd.AddCommand(credentialcmd.NewCredentialCmd())
	rootCmd.AddCommand(tagcmd.NewTagCmd())
	rootCmd.AddCommand(foldercmd.NewFolderCmd())
	rootCmd.AddCommand(datatablecmd.NewDataTableCmd())
	rootCmd.AddCommand(usercmd.NewUserCmd())
	rootCmd.AddCommand(auditcmd.NewAuditCmd())
	rootCmd.AddCommand(newVersionCmd())
}

func newVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print the version number",
		Run: func(cmd *cobra.Command, args []string) {
			commit, date := vcsInfo()
			if jsonOutput {
				out, _ := json.Marshal(map[string]string{"version": version, "commit": commit, "date": date})
				fmt.Println(string(out))
			} else {
				fmt.Printf("n8n-cli %s\ncommit: %s\nbuilt:  %s\n", version, commit, date)
			}
		},
	}
}

func vcsInfo() (commit, date string) {
	commit, date = "unknown", "unknown"
	var modified bool
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return
	}
	for _, s := range info.Settings {
		switch s.Key {
		case "vcs.revision":
			if len(s.Value) > 7 {
				commit = s.Value[:7]
			} else {
				commit = s.Value
			}
		case "vcs.time":
			date = s.Value
		case "vcs.modified":
			modified = s.Value == "true"
		}
	}
	if modified {
		commit += "-dirty"
	}
	return
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
		_ = PrintJSON(map[string]string{"error": err.Error()})
	} else {
		fmt.Fprintf(os.Stderr, "Error: %s\n", err)
	}
}
