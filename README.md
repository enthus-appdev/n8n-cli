# n8n-cli

A command-line interface for managing [n8n](https://n8n.io) workflows. Perfect for version control, automation, and LLM-assisted workflow development.

## Features

- **Workflow Management**: List, view, create, update, and delete workflows
- **Recursive Pull**: Download a workflow and all its sub-workflows with a single command
- **Smart Push**: Push workflows in dependency order with automatic ID remapping
- **Execution Control**: Run workflows, view execution history, and manage running executions
- **Source Control**: Preview, push, and pull changes through the instance's Git connection
- **Multi-Instance**: Configure and switch between multiple n8n instances
- **LLM-Friendly**: JSON output for seamless AI integration

## Installation

```bash
# From source
go install github.com/enthus-appdev/n8n-cli/cmd/n8nctl@latest

# Or build locally
git clone https://github.com/enthus-appdev/n8n-cli.git
cd n8n-cli
go build -ldflags "-X github.com/enthus-appdev/n8n-cli/internal/cmd.version=dev" -o bin/n8nctl ./cmd/n8nctl
```

## Quick Start

```bash
# Configure your n8n instance
n8nctl config init

# List all workflows
n8nctl workflow list

# Pull a workflow (with sub-workflows)
n8nctl workflow pull <id> --recursive --dir ./my-workflows

# Edit the JSON files...

# Push changes back
n8nctl workflow push ./my-workflows
```

## Commands

### Configuration

```bash
n8nctl config init              # Configure a new n8n instance (interactive)
n8nctl config init --name prod --url https://n8n.example.com --api-key KEY
n8nctl config list              # List configured instances
n8nctl config use <name>        # Switch active instance
n8nctl config remove <name>     # Remove an instance
```

### Workflows

```bash
n8nctl workflow list [--active] [--json]     # List workflows
n8nctl workflow view <id>                     # View workflow JSON
n8nctl workflow pull <id>                     # Download to file
n8nctl workflow pull <id> -r -d ./dir         # Recursive pull with sub-workflows
n8nctl workflow push <file>                   # Update workflow from file
n8nctl workflow push <dir>                    # Push from manifest
n8nctl workflow push <file> --create          # Create new workflow
n8nctl workflow run <id> [-i '{"key":"val"}'] # Execute workflow
n8nctl workflow activate <id>                 # Activate workflow
n8nctl workflow deactivate <id>               # Deactivate workflow
```

### Executions

```bash
n8nctl execution list [--workflow <id>]  # List executions
n8nctl execution view <id>               # View execution details
n8nctl execution retry <id>              # Retry a failed execution
n8nctl execution delete <id>             # Delete execution
```

### Source Control

Drives n8n's built-in source control (Environments): the Git branch connected to the
instance, not local files.

```bash
n8nctl source-control status                         # Pending changes a push would commit
n8nctl source-control status --direction pull        # Changes a pull would import
n8nctl source-control push -m "msg" --workflow <id>  # Commit and push one workflow
n8nctl source-control push -m "msg" --workflow <id> --workflow <id2> --item credential:<id>
n8nctl source-control push -m "msg" --workflow <id> --force   # Push despite conflicts
n8nctl source-control pull                           # Import the connected branch
n8nctl source-control pull --auto-publish published  # Republish workflows that were published
n8nctl source-control pull --force                   # Discard conflicting local changes
```

`sc` is an alias for `source-control`. `--item <type>:<id>` accepts the types `credential`,
`workflow`, `tags`, `variables`, `file`, `folders`, `project`, and `datatable`; take the IDs
from `status`. Push has no push-everything option, so pending changes from other people on
a shared instance are only committed when you name them.

Things to know before using it:

- A push commits to the connected branch. Every instance connected to the same branch
  receives the change on its next pull.
- A pull overwrites the drafts of every changed item with the Git version. `--auto-publish`
  controls publishing after import: `none` (server default) keeps each workflow's published
  state, `all` publishes everything imported, `published` republishes only workflows that
  were published before the pull.
- On a 409 the CLI lists the conflicting items and exits non-zero; rerun with `--force` only
  after checking them.
- The API key needs the `sourceControl:read` (status), `sourceControl:push`, or
  `sourceControl:pull` scope, and the instance needs source control connected. A missing
  scope returns 403.

## Recursive Pull & Push

The killer feature: pull a workflow and all its sub-workflows at once.

```bash
n8nctl workflow pull abc123 --recursive --dir ./workflows
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
n8nctl workflow push ./workflows
```

## For LLMs

Use `--json` flag for structured output:

```bash
n8nctl workflow list --json
n8nctl workflow view abc123 --json
n8nctl execution list --json
n8nctl source-control status --json
```

Typical workflow for LLM-assisted development:
1. `n8nctl workflow pull <id> -r -d ./wf` - Pull workflow tree
2. LLM reads and modifies JSON files
3. `n8nctl workflow push ./wf` - Push changes back

## Configuration

Config is stored in `~/.config/n8n-cli/config.json`

## Getting an API Key

1. Go to your n8n instance
2. Click on your user icon → Settings
3. Navigate to API → API Keys
4. Create a new API key

## License

MIT License - See [LICENSE](LICENSE) for details.
