# QIX Go CLI

QIX is a Go-based command line application for managing projects, modules, and tasks. It provides a rich CLI output with lipgloss styling and supports features like time tracking, Jira integration, and detailed task reports.

## Features

- Create/list/show projects and modules
- Task management with statuses, priorities, tags, and recurrence
- Time tracking commands (start/stop/status/log/switch)
- Detailed task views with colored sections
- Jira issue linking and integration (`qix jira open`)
- Configurable output colors, logging, and shell completions

## Requirements

- Go 1.21+
- Terminal with ANSI color support

## Installation

```bash
# Clone the repository
git clone https://github.com/mrbooshehri/qix-go.git
cd qix-go

# Build the CLI
go build -o qix .

# Optional: install globally
go install ./...
```

## Usage

Run `./qix --help` to see all commands. Key examples:

```bash
./qix project create myproject "Project description"
./qix module create myproject/backend "Backend module"
./qix task create myproject/backend "Implement feature" --priority high --estimated 4
./qix task show myproject 1234abcd
./qix track start myproject 1234abcd
./qix track stop
./qix jira open myproject 1234abcd
```

Sample output for a task detail view:

```
TASK-1234 :: Implement feature
═══════════════════════════════
Details
  ID:          TASK-1234
  Status:      ✅ done
  Priority:    🟡 medium
  Jira Issue:  ACME-42
  Description: Build the feature toggle flow

⏱️  Time Tracking
  Estimated:  4.00h
  Actual:     3.50h
  Variance:   -0.50h (12.5% under)

🏷️  Tags
  backend, feature-toggle

Timestamps
  Created: 2025-11-17 10:19:11
  Updated: 2025-11-18 15:04:02

📍 Location
  myproject/backend
```

Sample terminal interactions:

```bash
$ ./qix project create myproject "Project description"
✓ Project created: myproject
  Description: Project description

$ ./qix module create myproject/backend "Backend module"
✓ Module 'backend' created in project 'myproject'
  Description: Backend module

$ ./qix task create myproject/backend "Implement feature" --priority high --estimated 4
✓ Task created: Implement feature
  ID: abcd1234
  Location: myproject/backend
  Status: todo | Priority: high
  Estimated: 4.00h

$ ./qix task show myproject abcd1234
TASK-ABCD1234 :: Implement feature
═════════════════════════════════
Details
  ID:          abcd1234
  Status:      ⭕ todo
  Priority:    🔴 high
  Description: Implement feature

⏱️  Time Tracking
  Estimated:  4.00h
  Actual:     0.00h

📍 Location
  myproject/backend

$ ./qix track start myproject abcd1234
⏺️  Tracking started
  Task: [abcd1234] Implement feature
  Path: myproject (project level)
  Started: 15:42:01

$ ./qix track stop
⏹️  Tracking stopped
  Task: [abcd1234] Implement feature
  Duration: 00h05m12s
  Logged: 0.09h

$ ./qix jira open myproject abcd1234
✓ Opening Jira issue: https://your-domain.atlassian.net/browse/ACME-42
```

### Shell completions

Generate bash completions:

```bash
./qix completion bash > /etc/bash_completion.d/qix
source /etc/bash_completion.d/qix
```

Generate zsh completions:

```bash
mkdir -p ~/.zsh/completions
./qix completion zsh > ~/.zsh/completions/_qix
autoload -U compinit && compinit
```

## Configuration

Configuration is stored in `~/.qix/config`. Example entries:

```
QIX_DATE_FORMAT="2006-01-02"
QIX_DATETIME_FORMAT="2006-01-02T15:04:05Z07:00"
QIX_BACKUP_RETENTION_DAYS=30
QIX_COLOR_OUTPUT=true
QIX_LOG_LEVEL=debug
JIRA_BASE_URL=https://your-domain.atlassian.net/browse
```

## Logging

Logs are written to `~/.qix/qix.log`. Set `QIX_LOG_LEVEL` to `debug`, `info`, `warn`, or `error` to control verbosity.

## License

MIT
