package cmd

import (
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/mauricejumelet/jira-cli/internal/adf"
	"github.com/mauricejumelet/jira-cli/internal/api"
)

type IssuesCmd struct {
	Search     IssuesSearchCmd     `cmd:"" help:"Search issues with JQL"`
	Get        IssuesGetCmd        `cmd:"" help:"Get an issue by key"`
	Create     IssuesCreateCmd     `cmd:"" help:"Create a new issue"`
	Update     IssuesUpdateCmd     `cmd:"" help:"Update an issue"`
	Delete     IssuesDeleteCmd     `cmd:"" help:"Delete an issue"`
	Transition IssuesTransitionCmd `cmd:"" help:"Transition an issue to a new status"`
}

// --- Search ---

type IssuesSearchCmd struct {
	JQL       string `arg:"" optional:"" help:"JQL query string"`
	Project   string `short:"p" help:"Filter by project key"`
	Status    string `short:"s" help:"Filter by status name"`
	Assignee  string `short:"a" help:"Filter by assignee (account ID or 'currentUser()')"`
	MyIssues  bool   `short:"m" help:"Show only issues assigned to current user"`
	Type      string `short:"t" help:"Filter by issue type (Bug, Task, Story, etc.)"`
	Max       int    `short:"n" default:"50" help:"Maximum results"`
	JSON      bool   `short:"j" help:"Output as JSON"`
}

func (c *IssuesSearchCmd) Run(client *api.Client) error {
	jql := c.JQL

	// Build JQL from flags if no raw JQL given
	if jql == "" {
		var parts []string
		if c.Project != "" {
			parts = append(parts, fmt.Sprintf("project = %s", c.Project))
		}
		if c.Status != "" {
			parts = append(parts, fmt.Sprintf("status = \"%s\"", c.Status))
		}
		if c.MyIssues {
			parts = append(parts, "assignee = currentUser()")
		} else if c.Assignee != "" {
			parts = append(parts, fmt.Sprintf("assignee = \"%s\"", c.Assignee))
		}
		if c.Type != "" {
			parts = append(parts, fmt.Sprintf("issuetype = \"%s\"", c.Type))
		}
		if len(parts) == 0 {
			parts = append(parts, "order by updated DESC")
		} else {
			parts = append(parts, "order by updated DESC")
		}
		jql = strings.Join(parts[:len(parts)-1], " AND ")
		if len(parts) > 1 {
			jql += " " + parts[len(parts)-1]
		} else {
			jql = parts[0]
		}
	}

	result, err := client.SearchIssues(jql, nil, c.Max, "")
	if err != nil {
		return err
	}

	if c.JSON {
		return printJSON(result)
	}

	if len(result.Issues) == 0 {
		fmt.Println("No issues found.")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "KEY\tTYPE\tSTATUS\tPRIORITY\tASSIGNEE\tSUMMARY")
	fmt.Fprintln(w, "---\t----\t------\t--------\t--------\t-------")

	for _, issue := range result.Issues {
		issueType := "-"
		if issue.Fields.IssueType != nil {
			issueType = issue.Fields.IssueType.Name
		}
		status := "-"
		if issue.Fields.Status != nil {
			status = issue.Fields.Status.Name
		}
		priority := "-"
		if issue.Fields.Priority != nil {
			priority = issue.Fields.Priority.Name
		}
		assignee := "-"
		if issue.Fields.Assignee != nil {
			assignee = issue.Fields.Assignee.DisplayName
		}
		summary := truncate(issue.Fields.Summary, 50)
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\n", issue.Key, issueType, status, priority, assignee, summary)
	}
	w.Flush()

	if len(result.Issues) >= c.Max {
		fmt.Printf("\n(Showing %d issues, use -n to increase limit)\n", c.Max)
	}

	return nil
}

// --- Get ---

type IssuesGetCmd struct {
	IssueKey string `arg:"" help:"Issue key (e.g., PROJ-123)"`
	Comments bool   `help:"Include comments"`
	JSON     bool   `short:"j" help:"Output as JSON"`
}

func (c *IssuesGetCmd) Run(client *api.Client) error {
	issue, err := client.GetIssue(c.IssueKey, nil, nil)
	if err != nil {
		return err
	}

	var comments *api.CommentPage
	if c.Comments {
		comments, err = client.GetComments(c.IssueKey, 100)
		if err != nil {
			return err
		}
	}

	if c.JSON {
		if c.Comments {
			return printJSON(map[string]interface{}{
				"issue":    issue,
				"comments": comments,
			})
		}
		return printJSON(issue)
	}

	url := issueURL(client.BaseURL(), issue.Key)

	fmt.Printf("Key: %s\n", makeHyperlink(url, issue.Key))
	fmt.Printf("Summary: %s\n", issue.Fields.Summary)

	if issue.Fields.IssueType != nil {
		fmt.Printf("Type: %s\n", issue.Fields.IssueType.Name)
	}
	if issue.Fields.Status != nil {
		fmt.Printf("Status: %s\n", issue.Fields.Status.Name)
	}
	if issue.Fields.Priority != nil {
		fmt.Printf("Priority: %s\n", issue.Fields.Priority.Name)
	}
	if issue.Fields.Assignee != nil {
		fmt.Printf("Assignee: %s\n", issue.Fields.Assignee.DisplayName)
	}
	if issue.Fields.Reporter != nil {
		fmt.Printf("Reporter: %s\n", issue.Fields.Reporter.DisplayName)
	}
	if issue.Fields.Project != nil {
		fmt.Printf("Project: %s (%s)\n", issue.Fields.Project.Name, issue.Fields.Project.Key)
	}
	if len(issue.Fields.Labels) > 0 {
		fmt.Printf("Labels: %s\n", strings.Join(issue.Fields.Labels, ", "))
	}

	fmt.Printf("Created: %s\n", formatTimestamp(issue.Fields.Created))
	fmt.Printf("Updated: %s\n", formatTimestamp(issue.Fields.Updated))
	fmt.Printf("URL: %s\n", url)

	if issue.Fields.Description != nil {
		descText := adf.ExtractText(issue.Fields.Description)
		if descText != "" {
			fmt.Printf("\nDescription:\n%s\n", descText)
		}
	}

	if c.Comments && comments != nil && len(comments.Comments) > 0 {
		fmt.Printf("\nComments (%d):\n", comments.Total)
		fmt.Println(strings.Repeat("-", 40))
		for _, comment := range comments.Comments {
			author := "Unknown"
			if comment.Author != nil {
				author = comment.Author.DisplayName
			}
			fmt.Printf("[%s] %s\n", formatTimestamp(comment.Created), author)
			if comment.Body != nil {
				text := adf.ExtractText(comment.Body)
				if text != "" {
					fmt.Printf("  %s\n", text)
				}
			}
			fmt.Println()
		}
	}

	return nil
}

// --- Create ---

type IssuesCreateCmd struct {
	Project     string `short:"p" required:"" help:"Project key (e.g., ED)"`
	Type        string `short:"t" default:"Task" help:"Issue type (Bug, Task, Story, etc.)"`
	Summary     string `short:"s" required:"" help:"Issue summary/title"`
	Description string `short:"d" help:"Issue description (supports markdown)"`
	Priority    string `help:"Priority (Highest, High, Medium, Low, Lowest)"`
	Assignee    string `short:"a" help:"Assignee account ID"`
	Labels      string `short:"l" help:"Comma-separated labels"`
	JSON        bool   `short:"j" help:"Output as JSON"`
}

func (c *IssuesCreateCmd) Run(client *api.Client) error {
	var description map[string]interface{}
	if c.Description != "" {
		description = adf.MarkdownToADF(c.Description)
	}

	var labels []string
	if c.Labels != "" {
		labels = strings.Split(c.Labels, ",")
		for i := range labels {
			labels[i] = strings.TrimSpace(labels[i])
		}
	}

	issue, err := client.CreateIssue(c.Project, c.Summary, c.Type, description, c.Priority, c.Assignee, labels)
	if err != nil {
		return err
	}

	if c.JSON {
		return printJSON(issue)
	}

	url := issueURL(client.BaseURL(), issue.Key)
	fmt.Printf("Issue created: %s\n", makeHyperlink(url, issue.Key))
	fmt.Printf("URL: %s\n", url)

	return nil
}

// --- Update ---

type IssuesUpdateCmd struct {
	IssueKey    string `arg:"" help:"Issue key (e.g., PROJ-123)"`
	Summary     string `short:"s" help:"New summary"`
	Description string `short:"d" help:"New description (supports markdown)"`
	Priority    string `help:"New priority"`
	Assignee    string `short:"a" help:"New assignee account ID"`
	Unassign    bool   `help:"Remove assignee"`
	Labels      string `short:"l" help:"New comma-separated labels"`
}

func (c *IssuesUpdateCmd) Run(client *api.Client) error {
	fields := map[string]interface{}{}

	if c.Summary != "" {
		fields["summary"] = c.Summary
	}
	if c.Description != "" {
		fields["description"] = adf.MarkdownToADF(c.Description)
	}
	if c.Priority != "" {
		fields["priority"] = map[string]string{"name": c.Priority}
	}
	if c.Unassign {
		fields["assignee"] = nil
	} else if c.Assignee != "" {
		fields["assignee"] = map[string]string{"accountId": c.Assignee}
	}
	if c.Labels != "" {
		labels := strings.Split(c.Labels, ",")
		for i := range labels {
			labels[i] = strings.TrimSpace(labels[i])
		}
		fields["labels"] = labels
	}

	if len(fields) == 0 {
		return fmt.Errorf("no fields to update (use --summary, --description, --priority, --assignee, --unassign, or --labels)")
	}

	if err := client.UpdateIssue(c.IssueKey, fields); err != nil {
		return err
	}

	fmt.Printf("Issue %s updated.\n", c.IssueKey)
	return nil
}

// --- Delete ---

type IssuesDeleteCmd struct {
	IssueKey string `arg:"" help:"Issue key (e.g., PROJ-123)"`
	Force    bool   `short:"f" help:"Skip confirmation"`
}

func (c *IssuesDeleteCmd) Run(client *api.Client) error {
	if !c.Force {
		fmt.Printf("Are you sure you want to delete %s? [y/N] ", c.IssueKey)
		var response string
		fmt.Scanln(&response)
		if response != "y" && response != "Y" {
			fmt.Println("Cancelled.")
			return nil
		}
	}

	if err := client.DeleteIssue(c.IssueKey, false); err != nil {
		return err
	}

	fmt.Printf("Issue %s deleted.\n", c.IssueKey)
	return nil
}

// --- Transition ---

type IssuesTransitionCmd struct {
	IssueKey string `arg:"" help:"Issue key (e.g., PROJ-123)"`
	Status   string `arg:"" optional:"" help:"Target status name (e.g., 'In Progress', 'Done')"`
	List     bool   `help:"List available transitions"`
}

func (c *IssuesTransitionCmd) Run(client *api.Client) error {
	transitions, err := client.GetTransitions(c.IssueKey)
	if err != nil {
		return err
	}

	if c.List || c.Status == "" {
		fmt.Printf("Available transitions for %s:\n", c.IssueKey)
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "ID\tNAME\tTO STATUS")
		fmt.Fprintln(w, "--\t----\t---------")
		for _, t := range transitions {
			toStatus := "-"
			if t.To != nil {
				toStatus = t.To.Name
			}
			fmt.Fprintf(w, "%s\t%s\t%s\n", t.ID, t.Name, toStatus)
		}
		w.Flush()
		return nil
	}

	// Find transition by name (case-insensitive)
	var transitionID string
	for _, t := range transitions {
		if strings.EqualFold(t.Name, c.Status) {
			transitionID = t.ID
			break
		}
	}

	if transitionID == "" {
		// Try matching by target status name
		for _, t := range transitions {
			if t.To != nil && strings.EqualFold(t.To.Name, c.Status) {
				transitionID = t.ID
				break
			}
		}
	}

	if transitionID == "" {
		fmt.Printf("Transition '%s' not found. Available transitions:\n", c.Status)
		for _, t := range transitions {
			toStatus := ""
			if t.To != nil {
				toStatus = fmt.Sprintf(" -> %s", t.To.Name)
			}
			fmt.Printf("  - %s%s\n", t.Name, toStatus)
		}
		return fmt.Errorf("transition '%s' not found", c.Status)
	}

	if err := client.TransitionIssue(c.IssueKey, transitionID); err != nil {
		return err
	}

	fmt.Printf("Issue %s transitioned to '%s'.\n", c.IssueKey, c.Status)
	return nil
}
