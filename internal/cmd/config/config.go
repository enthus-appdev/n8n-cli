package config

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/enthus-appdev/n8n-cli/internal/config"
)

func NewConfigCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Manage n8n CLI configuration",
		Long:  `Configure n8n instances, authentication, and CLI settings.`,
	}

	cmd.AddCommand(newInitCmd())
	cmd.AddCommand(newListCmd())
	cmd.AddCommand(newUseCmd())
	cmd.AddCommand(newRemoveCmd())

	return cmd
}

func newInitCmd() *cobra.Command {
	var (
		name    string
		url     string
		apiKey  string
		setDefault bool
	)

	cmd := &cobra.Command{
		Use:   "init",
		Short: "Configure a new n8n instance",
		Long: `Interactively configure a new n8n instance connection.

You can also provide flags for non-interactive setup:
  n8n config init --name prod --url https://n8n.example.com --api-key YOUR_KEY`,
		RunE: func(cmd *cobra.Command, args []string) error {
			reader := bufio.NewReader(os.Stdin)

			// Interactive prompts for missing values
			if name == "" {
				fmt.Print("Instance name (e.g., 'local', 'prod'): ")
				name, _ = reader.ReadString('\n')
				name = strings.TrimSpace(name)
			}

			if url == "" {
				fmt.Print("n8n URL (e.g., 'http://localhost:5678'): ")
				url, _ = reader.ReadString('\n')
				url = strings.TrimSpace(url)
			}

			if apiKey == "" {
				fmt.Print("API Key: ")
				apiKey, _ = reader.ReadString('\n')
				apiKey = strings.TrimSpace(apiKey)
			}

			// Validate inputs
			if name == "" || url == "" || apiKey == "" {
				return fmt.Errorf("name, URL, and API key are required")
			}

			// Normalize URL (remove trailing slash)
			url = strings.TrimSuffix(url, "/")

			instance := config.Instance{
				Name:   name,
				URL:    url,
				APIKey: apiKey,
			}

			cfg, err := config.Load()
			if err != nil {
				// Create new config if doesn't exist
				cfg = &config.Config{
					Instances: make(map[string]config.Instance),
				}
			}

			cfg.Instances[name] = instance

			// Set as default if it's the first instance or explicitly requested
			if len(cfg.Instances) == 1 || setDefault {
				cfg.CurrentInstance = name
			}

			if err := config.Save(cfg); err != nil {
				return fmt.Errorf("failed to save config: %w", err)
			}

			fmt.Printf("Instance '%s' configured successfully.\n", name)
			if cfg.CurrentInstance == name {
				fmt.Printf("Set as active instance.\n")
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&name, "name", "", "Instance name")
	cmd.Flags().StringVar(&url, "url", "", "n8n instance URL")
	cmd.Flags().StringVar(&apiKey, "api-key", "", "API key for authentication")
	cmd.Flags().BoolVar(&setDefault, "default", false, "Set as default instance")

	return cmd
}

func newListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List configured n8n instances",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return fmt.Errorf("no configuration found. Run 'n8n config init' first")
			}

			if len(cfg.Instances) == 0 {
				fmt.Println("No instances configured. Run 'n8n config init' to add one.")
				return nil
			}

			jsonFlag, _ := cmd.Flags().GetBool("json")
			if jsonFlag {
				// Output JSON (hide API keys for security)
				type instanceInfo struct {
					Name   string `json:"name"`
					URL    string `json:"url"`
					Active bool   `json:"active"`
				}
				instances := make([]instanceInfo, 0, len(cfg.Instances))
				for name, inst := range cfg.Instances {
					instances = append(instances, instanceInfo{
						Name:   name,
						URL:    inst.URL,
						Active: name == cfg.CurrentInstance,
					})
				}
				return printJSON(map[string]interface{}{
					"instances": instances,
					"current":   cfg.CurrentInstance,
				})
			}

			fmt.Println("Configured instances:")
			for name, inst := range cfg.Instances {
				marker := "  "
				if name == cfg.CurrentInstance {
					marker = "* "
				}
				fmt.Printf("%s%s (%s)\n", marker, name, inst.URL)
			}

			return nil
		},
	}
}

func newUseCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "use <instance-name>",
		Short: "Switch to a different n8n instance",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]

			cfg, err := config.Load()
			if err != nil {
				return fmt.Errorf("no configuration found. Run 'n8n config init' first")
			}

			if _, exists := cfg.Instances[name]; !exists {
				return fmt.Errorf("instance '%s' not found", name)
			}

			cfg.CurrentInstance = name
			if err := config.Save(cfg); err != nil {
				return fmt.Errorf("failed to save config: %w", err)
			}

			fmt.Printf("Switched to instance '%s'\n", name)
			return nil
		},
	}
}

func newRemoveCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "remove <instance-name>",
		Short: "Remove a configured n8n instance",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]

			cfg, err := config.Load()
			if err != nil {
				return fmt.Errorf("no configuration found")
			}

			if _, exists := cfg.Instances[name]; !exists {
				return fmt.Errorf("instance '%s' not found", name)
			}

			delete(cfg.Instances, name)

			// Clear current if it was the removed instance
			if cfg.CurrentInstance == name {
				cfg.CurrentInstance = ""
				// Set first available as current
				for n := range cfg.Instances {
					cfg.CurrentInstance = n
					break
				}
			}

			if err := config.Save(cfg); err != nil {
				return fmt.Errorf("failed to save config: %w", err)
			}

			fmt.Printf("Instance '%s' removed.\n", name)
			return nil
		},
	}
}

func printJSON(v interface{}) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}
