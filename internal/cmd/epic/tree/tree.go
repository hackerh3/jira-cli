package tree

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/ankitpokhrel/jira-cli/api"
	"github.com/ankitpokhrel/jira-cli/internal/cmdutil"
	"github.com/ankitpokhrel/jira-cli/pkg/jira"
)

var (
	connector    = color.New(color.Faint).SprintFunc()
	keyStyle     = color.New(color.FgCyan, color.Bold).SprintFunc()
	epicKeyStyle = color.New(color.FgCyan, color.Bold, color.Underline).SprintFunc()
	doneStyle    = color.New(color.FgGreen).SprintFunc()
	activeStyle  = color.New(color.FgYellow).SprintFunc()
	blockedStyle = color.New(color.FgRed).SprintFunc()
	dimStyle     = color.New(color.Faint).SprintFunc()
)

const (
	helpText = `Tree displays the full issue hierarchy under an epic.

Shows the epic at root level, its child issues on the first level,
and their subtasks on the second level.`

	examples = `# Display epic hierarchy as a tree
$ jira epic tree EPIC-1

# Display as a flat table
$ jira epic tree EPIC-1 --plain

# Display as raw JSON
$ jira epic tree EPIC-1 --raw

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
		Annotations: map[string]string{
			"help:args": "EPIC-KEY\tKey for the issue of type epic, eg: ISSUE-1",
		},
		Args: cobra.ExactArgs(1),
		Run:  run,
	}
}

// SetFlags sets flags supported by the tree command.
func SetFlags(cmd *cobra.Command) {
	cmd.Flags().Bool("plain", false, "Display output in plain mode")
	cmd.Flags().Bool("no-headers", false, "Don't display table headers in plain mode. Works only with --plain")
	cmd.Flags().Bool("raw", false, "Print raw JSON output")
}

func run(cmd *cobra.Command, args []string) {
	project := viper.GetString("project.key")

	debug, err := cmd.Flags().GetBool("debug")
	cmdutil.ExitIfError(err)

	client := api.DefaultClient(debug)
	key := cmdutil.GetJiraIssueKey(project, args[0])

	tree, err := func() (*jira.EpicTree, error) {
		s := cmdutil.Info("Fetching epic hierarchy...")
		defer s.Stop()

		return client.EpicTree(key)
	}()
	cmdutil.ExitIfError(err)

	raw, err := cmd.Flags().GetBool("raw")
	cmdutil.ExitIfError(err)
	if raw {
		outputRawJSON(tree)
		return
	}

	plain, err := cmd.Flags().GetBool("plain")
	cmdutil.ExitIfError(err)

	noHeaders, err := cmd.Flags().GetBool("no-headers")
	cmdutil.ExitIfError(err)

	if plain {
		renderPlain(tree, noHeaders)
		return
	}

	renderTree(tree)
}

func outputRawJSON(tree *jira.EpicTree) {
	data, err := json.MarshalIndent(tree, "", "  ")
	if err != nil {
		cmdutil.Failed("Failed to marshal epic tree to JSON: %s", err)
		return
	}

	fmt.Println(string(data))
}

func statusColor(status string) string {
	s := strings.ToLower(status)

	switch {
	case s == "done" || s == "closed" || s == "resolved" || strings.Contains(s, "complete"):
		return doneStyle(status)
	case s == "blocked" || strings.Contains(s, "block"):
		return blockedStyle(status)
	case s == "in progress" || s == "in review" || s == "active" || strings.Contains(s, "progress"):
		return activeStyle(status)
	default:
		return dimStyle(status)
	}
}

func renderTree(tree *jira.EpicTree) {
	if tree == nil || tree.Epic == nil {
		return
	}

	fmt.Printf("%s [%s] %s\n", epicKeyStyle(tree.Epic.Key), statusColor(tree.Epic.Fields.Status.Name), tree.Epic.Fields.Summary)

	for i, child := range tree.Children {
		branch := "├── "
		nextIndent := "│   "
		if i == len(tree.Children)-1 {
			branch = "└── "
			nextIndent = "    "
		}

		fmt.Printf("%s%s [%s] %s\n",
			connector(branch),
			keyStyle(child.Issue.Key),
			statusColor(child.Issue.Fields.Status.Name),
			child.Issue.Fields.Summary,
		)

		for j, subtask := range child.Subtasks {
			subBranch := "├── "
			if j == len(child.Subtasks)-1 {
				subBranch = "└── "
			}

			fmt.Printf("%s%s%s [%s] %s\n",
				connector(nextIndent),
				connector(subBranch),
				keyStyle(subtask.Key),
				statusColor(subtask.Fields.Status.Name),
				subtask.Fields.Summary,
			)
		}
	}
}

func renderPlain(tree *jira.EpicTree, noHeaders bool) {
	if tree == nil || tree.Epic == nil {
		return
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 8, 1, '\t', 0)

	if !noHeaders {
		_, _ = fmt.Fprintln(w, strings.Join([]string{"KEY", "TYPE", "STATUS", "SUMMARY", "PARENT"}, "\t"))
	}

	writePlainRow(w, tree.Epic, "")

	for _, child := range tree.Children {
		writePlainRow(w, child.Issue, tree.Epic.Key)
		for _, subtask := range child.Subtasks {
			writePlainRow(w, subtask, child.Issue.Key)
		}
	}

	_ = w.Flush()
}

func writePlainRow(w *tabwriter.Writer, issue *jira.Issue, parent string) {
	if issue == nil {
		return
	}

	_, _ = fmt.Fprintf(
		w,
		"%s\t%s\t%s\t%s\t%s\n",
		issue.Key,
		issue.Fields.IssueType.Name,
		issue.Fields.Status.Name,
		issue.Fields.Summary,
		parent,
	)
}
