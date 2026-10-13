// Package cmdutil holds helpers shared by the command packages.
package cmdutil

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"

	"github.com/enthus-appdev/n8n-cli/internal/api"
	"github.com/enthus-appdev/n8n-cli/internal/config"
)

// GetClient returns an API client for the active instance.
func GetClient() (*api.Client, error) {
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

// JSONOutput reports whether the global --json flag is set.
func JSONOutput(cmd *cobra.Command) bool {
	v, _ := cmd.Flags().GetBool("json")
	return v
}

// PrintJSON writes v as indented JSON to stdout.
func PrintJSON(v interface{}) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}

// PrintNextCursor tells the user how to fetch the next page.
func PrintNextCursor(cursor string) {
	if cursor != "" {
		fmt.Printf("\nMore results available. Use --cursor %s to continue.\n", cursor)
	}
}

// FormatTime formats a timestamp in local time, or "" when unset.
func FormatTime(t *time.Time) string {
	if t == nil {
		return ""
	}
	return t.Local().Format("2006-01-02 15:04:05")
}
