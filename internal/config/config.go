package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// Config represents the CLI configuration
type Config struct {
	CurrentInstance string              `json:"currentInstance"`
	Instances       map[string]Instance `json:"instances"`
}

// Instance represents an n8n instance configuration
type Instance struct {
	Name   string `json:"name"`
	URL    string `json:"url"`
	APIKey string `json:"apiKey"`
}

// GetCurrentInstance returns the currently active instance
func (c *Config) GetCurrentInstance() (*Instance, error) {
	if c.CurrentInstance == "" {
		return nil, fmt.Errorf("no instance selected. Run 'n8n config use <name>'")
	}

	instance, exists := c.Instances[c.CurrentInstance]
	if !exists {
		return nil, fmt.Errorf("instance '%s' not found", c.CurrentInstance)
	}

	return &instance, nil
}

// configDir returns the configuration directory path
func configDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}

	return filepath.Join(home, ".config", "n8n-cli"), nil
}

// configPath returns the configuration file path
func configPath() (string, error) {
	dir, err := configDir()
	if err != nil {
		return "", err
	}

	return filepath.Join(dir, "config.json"), nil
}

// Load loads the configuration from disk
func Load() (*Config, error) {
	path, err := configPath()
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("no configuration found. Run 'n8n config init' first")
		}
		return nil, fmt.Errorf("failed to read config: %w", err)
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	return &cfg, nil
}

// Save saves the configuration to disk
func Save(cfg *Config) error {
	dir, err := configDir()
	if err != nil {
		return err
	}

	// Create config directory if it doesn't exist
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	path, err := configPath()
	if err != nil {
		return err
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	// Write with restricted permissions (owner read/write only)
	if err := os.WriteFile(path, data, 0600); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}

	return nil
}

// Exists checks if a configuration file exists
func Exists() bool {
	path, err := configPath()
	if err != nil {
		return false
	}

	_, err = os.Stat(path)
	return err == nil
}
