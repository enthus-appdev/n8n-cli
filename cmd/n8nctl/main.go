package main

import (
	"os"

	"github.com/enthus-appdev/n8n-cli/internal/cmd"
)

var version = "dev"

func main() {
	if err := cmd.Execute(version); err != nil {
		os.Exit(1)
	}
}
