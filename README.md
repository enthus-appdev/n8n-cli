# n8n-cli

A command-line interface for managing [n8n](https://n8n.io) workflows. Perfect for version control, automation, and LLM-assisted workflow development.

## Features

- **Workflow Management**: List, view, create, update, and delete workflows
- **Recursive Pull**: Download a workflow and all its sub-workflows with a single command
- **Smart Push**: Push workflows in dependency order with automatic ID remapping
- **Execution Control**: Run workflows, view execution history, and manage running executions
- **Multi-Instance**: Configure and switch between multiple n8n instances
- **LLM-Friendly**: JSON output for seamless AI integration

## Installation

```bash
# From source
go install github.com/enthus-appdev/n8n-cli/cmd/n8n@latest

# Or build locally
git clone https://github.com/enthus-appdev/n8n-cli.git
cd n8n-cli
make build
```

## Quick Start

```bash
# Configure your n8n instance
n8n config init

# List all workflows
n8n workflow list

# Pull a workflow (with sub-workflows)
n8n workflow pull <id> --recursive --dir ./my-workflows

# Edit the JSON files...

# Push changes back
n8n workflow push ./my-workflows
```

## Commands

### Configuration

```bash
n8n config init              # Configure a new n8n instance (interactive)
n8n config init --name prod --url https://n8n.example.com --api-key KEY
n8n config list              # List configured instances
n8n config use <name>        # Switch active instance
n8n config remove <name>     # Remove an instance
```

### Workflows

```bash
n8n workflow list [--active] [--json]     # List workflows
n8n workflow view <id>                     # View workflow JSON
n8n workflow pull <id>                     # Download to file
n8n workflow pull <id> -r -d ./dir         # Recursive pull with sub-workflows
n8n workflow push <file>                   # Update workflow from file
n8n workflow push <dir>                    # Push from manifest
n8n workflow push <file> --create          # Create new workflow
n8n workflow run <id> [-i '{"key":"val"}'] # Execute workflow
n8n workflow activate <id>                 # Activate workflow
n8n workflow deactivate <id>               # Deactivate workflow
```

### Executions

```bash
n8n execution list [--workflow <id>]  # List executions
n8n execution view <id>               # View execution details
n8n execution retry <id>              # Retry a failed execution
n8n execution delete <id>             # Delete execution
```

## Recursive Pull & Push

The killer feature: pull a workflow and all its sub-workflows at once.

```bash
n8n workflow pull abc123 --recursive --dir ./workflows
```

Creates:
```
./workflows/
  Main_Workflow.json
  Sub_Workflow_1.json
  Sub_Workflow_2.json
  manifest.json
```

The `manifest.json` tracks workflow relationships:
```json
{
  "rootWorkflow": "abc123",
  "workflows": {
    "abc123": {"id": "abc123", "name": "Main Workflow", "filename": "Main_Workflow.json"},
    "def456": {"id": "def456", "name": "Sub Workflow 1", "filename": "Sub_Workflow_1.json"}
  },
  "dependencies": {
    "abc123": ["def456", "ghi789"]
  }
}
```

Push back in the correct order:
```bash
n8n workflow push ./workflows
```

## For LLMs

Use `--json` flag for structured output:

```bash
n8n workflow list --json
n8n workflow view abc123 --json
n8n execution list --json
```

Typical workflow for LLM-assisted development:
1. `n8n workflow pull <id> -r -d ./wf` - Pull workflow tree
2. LLM reads and modifies JSON files
3. `n8n workflow push ./wf` - Push changes back

## Configuration

Config is stored in `~/.config/n8n-cli/config.json`

## Getting an API Key

1. Go to your n8n instance
2. Click on your user icon → Settings
3. Navigate to API → API Keys
4. Create a new API key

## License

MIT
