package main

import (
	"os"

	"github.com/enthus-appdev/n8n-cli/internal/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
