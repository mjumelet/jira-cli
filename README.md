# jira-cli

A command-line interface for Jira Cloud, built in Go.

## Installation

### Homebrew

```bash
brew tap mjumelet/tap
brew install jira
```

### From source

```bash
go install github.com/mauricejumelet/jira-cli@latest
```

## Configuration

Create a `.env` file at `~/.config/jira/.env`:

```
JIRA_BASE_URL=https://yourcompany.atlassian.net
JIRA_EMAIL=you@example.com
JIRA_API_TOKEN=your_api_token
```

Get your API token at: https://id.atlassian.com/manage-profile/security/api-tokens

Config is loaded from (in order):
1. Environment variables
2. `.env` in current directory
3. `~/.config/jira/.env`
4. Custom file via `--config` flag

Run `jira configure` to see setup help.

## Usage

### Issues

```bash
# Search issues
jira issues search "project = ED ORDER BY updated DESC"
jira issues search -p ED -s "In Progress"
jira issues search --my-issues

# Get issue details
jira issues get ED-123
jira issues get ED-123 --comments
jira issues get ED-123 --json

# Create issue
jira issues create -p ED -t Task -s "Fix the login bug"
jira issues create -p ED -t Bug -s "Crash on save" -d "Steps to reproduce: ..."

# Update issue
jira issues update ED-123 -s "Updated title"
jira issues update ED-123 -d "New description with **markdown**"
jira issues update ED-123 -a <account-id>
jira issues update ED-123 --unassign

# Delete issue
jira issues delete ED-123
jira issues delete ED-123 --force

# Transition issue
jira issues transition ED-123 "In Progress"
jira issues transition ED-123 "Done"
jira issues transition ED-123 --list
```

### Comments

```bash
jira comments list ED-123
jira comments add ED-123 "This is a **markdown** comment"
jira comments add ED-123 --file comment.md
jira comments update ED-123 <comment-id> "Updated text"
jira comments delete ED-123 <comment-id>
```

### Attachments

```bash
jira attachments add ED-123 ./screenshot.png
jira attachments add ED-123 ./report.pdf --filename "Q4 Report.pdf"
```

### Projects

```bash
jira projects list
jira projects get ED
```

### Users

```bash
jira users me
jira users search "john"
jira users assignable -p ED
```

## Markdown-to-ADF

Descriptions and comments support Markdown, which is automatically converted to Atlassian Document Format (ADF). Supported syntax:

- **Bold** (`**text**`)
- *Italic* (`*text*`)
- `Inline code` (`` `code` ``)
- [Links](url) (`[text](url)`)
- Headings (`# H1` through `###### H6`)
- Bullet lists (`- item`)
- Numbered lists (`1. item`)
- Code blocks (triple backticks with optional language)
- Tables (pipe syntax)
- Horizontal rules (`---`)

## License

MIT
