package cmd

import (
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/mauricejumelet/jira-cli/internal/adf"
	"github.com/mauricejumelet/jira-cli/internal/api"
)

type CommentsCmd struct {
	List   CommentsListCmd   `cmd:"" help:"List comments on an issue"`
	Add    CommentsAddCmd    `cmd:"" help:"Add a comment to an issue"`
	Update CommentsUpdateCmd `cmd:"" help:"Update a comment"`
	Delete CommentsDeleteCmd `cmd:"" help:"Delete a comment"`
}

// --- List ---

type CommentsListCmd struct {
	IssueKey string `arg:"" help:"Issue key (e.g., PROJ-123)"`
	Max      int    `short:"n" default:"50" help:"Maximum results"`
	JSON     bool   `short:"j" help:"Output as JSON"`
}

func (c *CommentsListCmd) Run(client *api.Client) error {
	comments, err := client.GetComments(c.IssueKey, c.Max)
	if err != nil {
		return err
	}

	if c.JSON {
		return printJSON(comments)
	}

	if len(comments.Comments) == 0 {
		fmt.Printf("No comments on %s.\n", c.IssueKey)
		return nil
	}

	fmt.Printf("Comments on %s (%d total):\n\n", c.IssueKey, comments.Total)

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	for _, comment := range comments.Comments {
		author := "Unknown"
		if comment.Author != nil {
			author = comment.Author.DisplayName
		}
		text := ""
		if comment.Body != nil {
			text = adf.ExtractText(comment.Body)
		}
		fmt.Fprintf(w, "[%s] %s (ID: %s)\n", formatTimestamp(comment.Created), author, comment.ID)
		fmt.Fprintf(w, "  %s\n\n", truncate(text, 200))
	}
	w.Flush()

	return nil
}

// --- Add ---

type CommentsAddCmd struct {
	IssueKey string `arg:"" help:"Issue key (e.g., PROJ-123)"`
	Text     string `arg:"" optional:"" help:"Comment text (supports markdown)"`
	File     string `help:"Read comment from file"`
	JSON     bool   `short:"j" help:"Output as JSON"`
}

func (c *CommentsAddCmd) Run(client *api.Client) error {
	text := c.Text

	if c.File != "" {
		data, err := os.ReadFile(c.File)
		if err != nil {
			return fmt.Errorf("reading file: %w", err)
		}
		text = string(data)
	}

	if text == "" {
		return fmt.Errorf("comment text is required (provide as argument or via --file)")
	}

	adfBody := adf.MarkdownToADF(text)

	comment, err := client.AddComment(c.IssueKey, adfBody)
	if err != nil {
		return err
	}

	if c.JSON {
		return printJSON(comment)
	}

	fmt.Printf("Comment added to %s (ID: %s)\n", c.IssueKey, comment.ID)
	return nil
}

// --- Update ---

type CommentsUpdateCmd struct {
	IssueKey  string `arg:"" help:"Issue key (e.g., PROJ-123)"`
	CommentID string `arg:"" help:"Comment ID"`
	Text      string `arg:"" help:"New comment text (supports markdown)"`
	JSON      bool   `short:"j" help:"Output as JSON"`
}

func (c *CommentsUpdateCmd) Run(client *api.Client) error {
	adfBody := adf.MarkdownToADF(c.Text)

	comment, err := client.UpdateComment(c.IssueKey, c.CommentID, adfBody)
	if err != nil {
		return err
	}

	if c.JSON {
		return printJSON(comment)
	}

	fmt.Printf("Comment %s updated on %s.\n", c.CommentID, c.IssueKey)
	return nil
}

// --- Delete ---

type CommentsDeleteCmd struct {
	IssueKey  string `arg:"" help:"Issue key (e.g., PROJ-123)"`
	CommentID string `arg:"" help:"Comment ID"`
	Force     bool   `short:"f" help:"Skip confirmation"`
}

func (c *CommentsDeleteCmd) Run(client *api.Client) error {
	if !c.Force {
		fmt.Printf("Are you sure you want to delete comment %s on %s? [y/N] ", c.CommentID, c.IssueKey)
		var response string
		fmt.Scanln(&response)
		if strings.ToLower(response) != "y" {
			fmt.Println("Cancelled.")
			return nil
		}
	}

	if err := client.DeleteComment(c.IssueKey, c.CommentID); err != nil {
		return err
	}

	fmt.Printf("Comment %s deleted from %s.\n", c.CommentID, c.IssueKey)
	return nil
}
