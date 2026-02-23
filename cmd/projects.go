package cmd

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/mauricejumelet/jira-cli/internal/api"
)

type ProjectsCmd struct {
	List ProjectsListCmd `cmd:"" help:"List projects"`
	Get  ProjectsGetCmd  `cmd:"" help:"Get project details"`
}

// --- List ---

type ProjectsListCmd struct {
	Max  int  `short:"n" default:"50" help:"Maximum results"`
	JSON bool `short:"j" help:"Output as JSON"`
}

func (c *ProjectsListCmd) Run(client *api.Client) error {
	projects, err := client.GetProjects(c.Max)
	if err != nil {
		return err
	}

	if c.JSON {
		return printJSON(projects)
	}

	if len(projects) == 0 {
		fmt.Println("No projects found.")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "KEY\tNAME\tID")
	fmt.Fprintln(w, "---\t----\t--")
	for _, p := range projects {
		fmt.Fprintf(w, "%s\t%s\t%s\n", p.Key, p.Name, p.ID)
	}
	w.Flush()

	return nil
}

// --- Get ---

type ProjectsGetCmd struct {
	ProjectKey string `arg:"" help:"Project key (e.g., ED)"`
	JSON       bool   `short:"j" help:"Output as JSON"`
}

func (c *ProjectsGetCmd) Run(client *api.Client) error {
	project, err := client.GetProject(c.ProjectKey)
	if err != nil {
		return err
	}

	if c.JSON {
		return printJSON(project)
	}

	fmt.Printf("Key: %s\n", getStr(project, "key"))
	fmt.Printf("Name: %s\n", getStr(project, "name"))
	fmt.Printf("ID: %s\n", getStr(project, "id"))

	if lead, ok := project["lead"].(map[string]interface{}); ok {
		fmt.Printf("Lead: %s\n", getStr(lead, "displayName"))
	}

	if desc := getStr(project, "description"); desc != "" {
		fmt.Printf("Description: %s\n", desc)
	}

	return nil
}

func getStr(m map[string]interface{}, key string) string {
	if v, ok := m[key].(string); ok {
		return v
	}
	return ""
}
