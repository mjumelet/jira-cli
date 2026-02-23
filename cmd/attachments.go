package cmd

import (
	"fmt"

	"github.com/mauricejumelet/jira-cli/internal/api"
)

type AttachmentsCmd struct {
	Add AttachmentsAddCmd `cmd:"" help:"Add an attachment to an issue"`
}

type AttachmentsAddCmd struct {
	IssueKey string `arg:"" help:"Issue key (e.g., PROJ-123)"`
	FilePath string `arg:"" help:"Path to file to attach"`
	Filename string `help:"Display name for the attachment (defaults to file's basename)"`
	JSON     bool   `short:"j" help:"Output as JSON"`
}

func (c *AttachmentsAddCmd) Run(client *api.Client) error {
	attachments, err := client.AddAttachment(c.IssueKey, c.FilePath, c.Filename)
	if err != nil {
		return err
	}

	if c.JSON {
		return printJSON(attachments)
	}

	for _, a := range attachments {
		fmt.Printf("Attachment added to %s: %s (ID: %s, %d bytes)\n", c.IssueKey, a.Filename, a.ID, a.Size)
	}

	return nil
}
