package cmd

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/mauricejumelet/jira-cli/internal/api"
)

type UsersCmd struct {
	Me         UsersMeCmd         `cmd:"" help:"Show current user"`
	Search     UsersSearchCmd     `cmd:"" help:"Search for users"`
	Assignable UsersAssignableCmd `cmd:"" help:"List users assignable to a project"`
}

// --- Me ---

type UsersMeCmd struct {
	JSON bool `short:"j" help:"Output as JSON"`
}

func (c *UsersMeCmd) Run(client *api.Client) error {
	user, err := client.GetMyself()
	if err != nil {
		return err
	}

	if c.JSON {
		return printJSON(user)
	}

	fmt.Printf("Name: %s\n", user.DisplayName)
	fmt.Printf("Email: %s\n", user.Email)
	fmt.Printf("Account ID: %s\n", user.AccountID)

	return nil
}

// --- Search ---

type UsersSearchCmd struct {
	Query string `arg:"" help:"Search query (name or email)"`
	Max   int    `short:"n" default:"50" help:"Maximum results"`
	JSON  bool   `short:"j" help:"Output as JSON"`
}

func (c *UsersSearchCmd) Run(client *api.Client) error {
	users, err := client.SearchUsers(c.Query, c.Max)
	if err != nil {
		return err
	}

	if c.JSON {
		return printJSON(users)
	}

	if len(users) == 0 {
		fmt.Println("No users found.")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ACCOUNT ID\tNAME\tEMAIL")
	fmt.Fprintln(w, "----------\t----\t-----")
	for _, u := range users {
		email := u.Email
		if email == "" {
			email = "-"
		}
		fmt.Fprintf(w, "%s\t%s\t%s\n", u.AccountID, u.DisplayName, email)
	}
	w.Flush()

	return nil
}

// --- Assignable ---

type UsersAssignableCmd struct {
	Project string `short:"p" required:"" help:"Project key"`
	Max     int    `short:"n" default:"50" help:"Maximum results"`
	JSON    bool   `short:"j" help:"Output as JSON"`
}

func (c *UsersAssignableCmd) Run(client *api.Client) error {
	users, err := client.GetAssignableUsers(c.Project, c.Max)
	if err != nil {
		return err
	}

	if c.JSON {
		return printJSON(users)
	}

	if len(users) == 0 {
		fmt.Println("No assignable users found.")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ACCOUNT ID\tNAME\tEMAIL")
	fmt.Fprintln(w, "----------\t----\t-----")
	for _, u := range users {
		email := u.Email
		if email == "" {
			email = "-"
		}
		fmt.Fprintf(w, "%s\t%s\t%s\n", u.AccountID, u.DisplayName, email)
	}
	w.Flush()

	return nil
}
