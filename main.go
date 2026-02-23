package main

import (
	"fmt"
	"os"

	"github.com/alecthomas/kong"
	"github.com/mauricejumelet/jira-cli/cmd"
	"github.com/mauricejumelet/jira-cli/internal/api"
	"github.com/mauricejumelet/jira-cli/internal/config"
)

var version = "0.1.0"

var CLI struct {
	// Global flags
	Config string `short:"c" help:"Path to config file (.env format)" type:"path"`

	// Commands
	Issues      cmd.IssuesCmd      `cmd:"" help:"Manage issues"`
	Comments    cmd.CommentsCmd    `cmd:"" help:"Manage comments"`
	Attachments cmd.AttachmentsCmd `cmd:"" help:"Manage attachments"`
	Projects    cmd.ProjectsCmd    `cmd:"" help:"Manage projects"`
	Users       cmd.UsersCmd       `cmd:"" help:"Manage users"`
	Configure   ConfigureCmd       `cmd:"" help:"Show configuration help"`
}

type ConfigureCmd struct{}

func (c *ConfigureCmd) Run() error {
	config.PrintConfigHelp()
	return nil
}

func main() {
	// Handle version flag early
	for _, arg := range os.Args[1:] {
		if arg == "-v" || arg == "--version" {
			fmt.Printf("jira-cli v%s\n", version)
			return
		}
	}

	ctx := kong.Parse(&CLI,
		kong.Name("jira"),
		kong.Description("A command-line interface for Jira Cloud (v"+version+")"),
		kong.UsageOnError(),
		kong.ConfigureHelp(kong.HelpOptions{
			Compact: true,
		}),
	)

	// Commands that don't need the API client
	switch ctx.Command() {
	case "configure":
		err := ctx.Run()
		ctx.FatalIfErrorf(err)
		return
	}

	// Load configuration
	cfg, err := config.Load(CLI.Config)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	// Create API client
	client := api.NewClient(cfg)

	// Run the command with the client
	err = ctx.Run(client)
	ctx.FatalIfErrorf(err)
}
