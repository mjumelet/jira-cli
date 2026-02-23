# jira-cli

Go CLI for Jira Cloud using Kong framework.

## Build & Run

```bash
go build -o jira .
./jira --version
./jira --help
```

## Project Structure

- `main.go` — Entry point, Kong CLI definition
- `cmd/` — Command implementations (issues, comments, attachments, projects, users)
- `internal/api/` — Jira REST API v3 client (Basic Auth)
- `internal/adf/` — Atlassian Document Format: builder, markdown-to-ADF converter, text extractor
- `internal/config/` — Config loading from env vars and .env files

## Config

Reads from `JIRA_BASE_URL`, `JIRA_EMAIL`, `JIRA_API_TOKEN` env vars or `~/.config/jira/.env`.

## Release

GoReleaser with Homebrew tap at `mjumelet/homebrew-tap`. Tag with `v*` to trigger release.
