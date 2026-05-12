package tree

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/ankitpokhrel/jira-cli/api"
	"github.com/ankitpokhrel/jira-cli/internal/cmdutil"
	"github.com/ankitpokhrel/jira-cli/pkg/jira"
)

const (
	helpText = `Tree displays the full issue hierarchy under an epic.

Shows the epic at root level, its child stories/tasks on the first level,
and their subtasks on the second level.`

	examples = `# Display epic hierarchy as a tree
$ jira epic tree EPIC-1

# Display as a flat table
$ jira epic tree EPIC-1 --plain

# Display as a flat table without headers
$ jira epic tree EPIC-1 --plain --no-headers`
)

// NewCmdTree is a tree command.
func NewCmdTree() *cobra.Command {
	return &cobra.Command{
		Use:     "tree EPIC-KEY",
		Short:   "Display full issue hierarchy under an epic",
		Long:    helpText,
		Example: examples,
		Args:    cobra.ExactArgs(1),
		Run:     run,
	}
}

// SetFlags sets flags supported by the tree command.
func SetFlags(cmd *cobra.Command) {
	cmd.Flags().Bool("plain", false, "Display output as a flat table")
	cmd.Flags().Bool("no-headers", false, "Don't display column headers in plain mode")
	cmd.Flags().Bool("debug", false, "Turn on debug output")
}

func run(cmd *cobra.Command, args []string) {
	project := viper.GetString("project.key")
	projectType := viper.GetString("project.type")

	debug, err := cmd.Flags().GetBool("debug")
	cmdutil.ExitIfError(err)

	client := api.DefaultClient(debug)
	key := cmdutil.GetJiraIssueKey(project, args[0])

	epic, err := func() (*jira.Issue, error) {
		s := cmdutil.Info("Fetching epic...")
		defer s.Stop()
		return api.ProxyGetIssue(client, key)
	}()
	cmdutil.ExitIfError(err)

	children, err := func() ([]*jira.Issue, error) {
		s := cmdutil.Info("Fetching epic issues...")
		defer s.Stop()

		var resp *jira.SearchResult

		if projectType == jira.ProjectTypeNextGen {
			resp, err = api.ProxySearch(client, fmt.Sprintf("parent = %s", key), 0, 100)
		} else {
			resp, err = client.EpicIssues(key, "", 0, 100)
		}
		if err != nil {
			return nil, err
		}
		return resp.Issues, nil
	}()
	cmdutil.ExitIfError(err)

	plain, err := cmd.Flags().GetBool("plain")
	cmdutil.ExitIfError(err)

	noHeaders, err := cmd.Flags().GetBool("no-headers")
	cmdutil.ExitIfError(err)

	if plain {
		renderPlain(epic, children, noHeaders)
	} else {
		renderTree(epic, children)
	}
}

func renderTree(epic *jira.Issue, children []*jira.Issue) {
	fmt.Printf("%s  %s  [%s]\n", epic.Key, epic.Fields.Summary, epic.Fields.Status.Name)

	for i, child := range children {
		last := i == len(children)-1
		prefix, childPrefix := "├── ", "│   "
		if last {
			prefix, childPrefix = "└── ", "    "
		}

		fmt.Printf("%s%s  %s  [%s]\n", prefix, child.Key, child.Fields.Summary, child.Fields.Status.Name)

		for j, sub := range child.Fields.Subtasks {
			subPrefix := childPrefix + "├── "
			if j == len(child.Fields.Subtasks)-1 {
				subPrefix = childPrefix + "└── "
			}
			fmt.Printf("%s%s  %s  [%s]\n", subPrefix, sub.Key, sub.Fields.Summary, sub.Fields.Status.Name)
		}
	}
}

func renderPlain(epic *jira.Issue, children []*jira.Issue, noHeaders bool) {
	w := tabwriter.NewWriter(os.Stdout, 0, 8, 1, '\t', 0)

	if !noHeaders {
		fmt.Fprintln(w, "TYPE\tKEY\tSUMMARY\tSTATUS\tPARENT")
	}

	fmt.Fprintf(w, "%s\t%s\t%s\t%s\t\n",
		epic.Fields.IssueType.Name, epic.Key, epic.Fields.Summary, epic.Fields.Status.Name)

	for _, child := range children {
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n",
			child.Fields.IssueType.Name, child.Key, child.Fields.Summary, child.Fields.Status.Name, epic.Key)

		for _, sub := range child.Fields.Subtasks {
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n",
				sub.Fields.IssueType.Name, sub.Key, sub.Fields.Summary, sub.Fields.Status.Name, child.Key)
		}
	}

	_ = w.Flush()
}
