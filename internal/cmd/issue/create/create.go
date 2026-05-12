package create

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/AlecAivazis/survey/v2"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/ankitpokhrel/jira-cli/api"
	"github.com/ankitpokhrel/jira-cli/internal/cmdcommon"
	"github.com/ankitpokhrel/jira-cli/internal/cmdutil"
	"github.com/ankitpokhrel/jira-cli/internal/query"
	"github.com/ankitpokhrel/jira-cli/pkg/jira"
	"github.com/ankitpokhrel/jira-cli/pkg/surveyext"
	"github.com/ankitpokhrel/jira-cli/pkg/tui"
)

const (
	helpText = `Create an issue in a given project with minimal information.`
	examples = `$ jira issue create

# Create issue in the configured project
$ jira issue create -tBug -s"New Bug" -yHigh -lbug -lurgent -b"Bug description"

# Create issue in another project
$ jira issue create -pPRJ -tBug -yHigh -s"New Bug" -b$'Bug description\n\nSome more text'

# Create issue and set custom fields
# See https://github.com/ankitpokhrel/jira-cli/discussions/346
$ jira issue create -tStory -s"Issue with custom fields" --custom story-points=3

# Load description from template file
$ jira issue create --template /path/to/template.tmpl

# Get description from standard input
$ jira issue create --template -

	# Create issue from a Deviniti Issue Template
	$ jira issue create --jira-template 14866 --template-var TITLE="My Task" -tStory --no-input

# Create issue in the configured project with JSON output
$ jira issue create --raw

# Or, use pipe to read input directly from standard input
$ echo "Description from stdin" | jira issue create -s"Summary" -tTask

# For issue description, the flag --body/-b takes precedence over the --template flag
# The example below will add "Body from flag" as an issue description
$ jira issue create -tTask -sSummary -b"Body from flag" --template /path/to/template.tpl`

	flagRaw = "raw"
)

// NewCmdCreate is a create command.
func NewCmdCreate() *cobra.Command {
	cmd := cobra.Command{
		Use:     "create",
		Short:   "Create an issue in a project",
		Long:    helpText,
		Example: examples,
		Run:     create,
	}

	cmd.Flags().Bool(flagRaw, false, "Print output in JSON format")

	return &cmd
}

// SetFlags sets flags supported by create command.
func SetFlags(cmd *cobra.Command) {
	cmdcommon.SetCreateFlags(cmd, "Issue")
}

func create(cmd *cobra.Command, _ []string) {
	server := viper.GetString("server")
	project := viper.GetString("project.key")
	projectType := viper.GetString("project.type")
	installation := viper.GetString("installation")

	params := parseFlags(cmd.Flags())
	client := api.DefaultClient(params.Debug)
	if params.JiraTemplate != "" {
		if params.IssueType == "" {
			cmdutil.Failed("Param `--type` is mandatory when using `--jira-template`")
		}
		s := cmdutil.Info("Fetching template...")
		err := applyIssueTemplate(client, project, params)
		s.Stop()
		cmdutil.ExitIfError(err)
	}

	cc := createCmd{
		client: client,
		params: params,
	}

	if cc.isNonInteractive() || cc.params.NoInput || tui.IsDumbTerminal() {
		cc.params.NoInput = true

		if cc.isMandatoryParamsMissing() {
			cmdutil.Failed(
				"Params `--summary` and `--type` is mandatory when using a non-interactive mode",
			)
		}
	}

	cmdutil.ExitIfError(cc.setIssueTypes())
	cmdutil.ExitIfError(cc.askQuestions())

	if !params.NoInput {
		err := cmdcommon.HandleNoInput(params)
		cmdutil.ExitIfError(err)
	}

	params.Reporter = cmdcommon.GetRelevantUser(client, project, params.Reporter)
	params.Assignee = cmdcommon.GetRelevantUser(client, project, params.Assignee)
	cc.params = params

	if err := cc.warnOnDuplicateSubtaskType(); err != nil {
		cmdutil.ExitIfError(err)
	}

	issue, err := func() (*jira.CreateResponse, error) {
		s := cmdutil.Info("Creating an issue...")
		defer s.Stop()

		var body any = params.Body
		if params.BodyIsJiraMarkup {
			body = jira.JiraMarkup(params.Body)
		}

		cr := jira.CreateRequest{
			Project:          project,
			IssueType:        params.IssueType,
			ParentIssueKey:   params.ParentIssueKey,
			Summary:          params.Summary,
			Body:             body,
			Reporter:         params.Reporter,
			Assignee:         params.Assignee,
			Priority:         params.Priority,
			Labels:           params.Labels,
			Components:       params.Components,
			FixVersions:      params.FixVersions,
			AffectsVersions:  params.AffectsVersions,
			OriginalEstimate: params.OriginalEstimate,
			CustomFields:     params.CustomFields,
			EpicField:        viper.GetString("epic.link"),
		}
		cr.JiraTemplateID = params.JiraTemplate
		cr.ForProjectType(projectType)
		cr.ForInstallationType(installation)
		if configuredCustomFields, err := cmdcommon.GetConfiguredCustomFields(); err == nil {
			cmdcommon.ValidateCustomFields(cr.CustomFields, configuredCustomFields)
			cr.WithCustomFields(configuredCustomFields)
		}

		if handle := cmdutil.GetSubtaskHandle(params.IssueType, cc.issueTypes); handle != "" {
			cr.SubtaskField = handle
		}

		return client.CreateV2(&cr)
	}()

	cmdutil.ExitIfError(err)

	jsonFlag, err := cmd.Flags().GetBool(flagRaw)
	cmdutil.ExitIfError(err)
	if jsonFlag {
		jsonData, err := json.Marshal(issue)
		cmdutil.ExitIfError(err)
		fmt.Println(string(jsonData))
		return
	}

	cmdutil.Success("Issue created\n%s", cmdutil.GenerateServerBrowseURL(server, issue.Key))

	if web, _ := cmd.Flags().GetBool("web"); web {
		err := cmdutil.Navigate(server, issue.Key)
		cmdutil.ExitIfError(err)
	}
}

func applyIssueTemplate(client *jira.Client, project string, params *cmdcommon.CreateParams) error {
	projectResp, err := client.GetProjectV2(project)
	if err != nil {
		return err
	}

	issueTypeID, err := resolveIssueTypeID(params.IssueType)
	if err != nil {
		return err
	}

	templateResp, err := client.GetIssueTemplate(params.JiraTemplate, projectResp.ID, issueTypeID)
	if err != nil {
		return err
	}

	for _, variable := range templateResp.UserVariables {
		if !variable.Required {
			continue
		}
		if strings.TrimSpace(params.TemplateVars[variable.Key]) == "" {
			return fmt.Errorf("required template variable %q is missing", variable.Key)
		}
	}

	for _, field := range templateResp.Fields {
		substituted := jira.SubstituteTemplateVars(field.Text1, params.TemplateVars)

		switch field.FieldType {
		case jira.TemplateFieldSummary:
			if params.Summary == "" || field.Overwritable {
				params.Summary = substituted
			}
		case jira.TemplateFieldDescription:
			if params.Body == "" || field.Overwritable {
				params.Body = substituted
				params.BodyIsJiraMarkup = true
			}
		case jira.TemplateFieldLabels:
			labels := strings.FieldsFunc(substituted, func(r rune) bool {
				return r == ',' || r == '\n'
			})
			for i := range labels {
				labels[i] = strings.TrimSpace(labels[i])
			}
			filtered := labels[:0]
			for _, label := range labels {
				if label != "" {
					filtered = append(filtered, label)
				}
			}
			params.Labels = mergeLabels(filtered, params.Labels)
		}
	}

	return nil
}

func resolveIssueTypeID(issueType string) (string, error) {
	availableTypes, ok := viper.Get("issue.types").([]any)
	if !ok {
		return "", fmt.Errorf("invalid issue types in config")
	}

	for _, availableType := range availableTypes {
		typeMap, ok := availableType.(map[string]any)
		if !ok {
			continue
		}

		name, _ := typeMap["name"].(string)
		handle, _ := typeMap["handle"].(string)
		id, _ := typeMap["id"].(string)

		if strings.EqualFold(issueType, handle) || strings.EqualFold(issueType, name) {
			return id, nil
		}
	}

	return "", fmt.Errorf("issue type %q not found in config", issueType)
}

func mergeLabels(templateLabels, userLabels []string) []string {
	if len(templateLabels) == 0 {
		return userLabels
	}

	merged := make([]string, 0, len(templateLabels)+len(userLabels))
	seen := make(map[string]struct{}, len(templateLabels)+len(userLabels))

	for _, label := range templateLabels {
		if _, ok := seen[label]; ok {
			continue
		}
		seen[label] = struct{}{}
		merged = append(merged, label)
	}
	for _, label := range userLabels {
		if _, ok := seen[label]; ok {
			continue
		}
		seen[label] = struct{}{}
		merged = append(merged, label)
	}

	return merged
}

type createCmd struct {
	client     *jira.Client
	issueTypes []*jira.IssueType
	params     *cmdcommon.CreateParams
}

func (cc *createCmd) setIssueTypes() error {
	issueTypes := make([]*jira.IssueType, 0)
	availableTypes, ok := viper.Get("issue.types").([]any)
	if !ok {
		return fmt.Errorf("invalid issue types in config")
	}
	for _, at := range availableTypes {
		tp := at.(map[string]any)
		name := tp["name"].(string)
		handle, _ := tp["handle"].(string)
		if handle == jira.IssueTypeEpic || name == jira.IssueTypeEpic {
			continue
		}
		issueTypes = append(issueTypes, &jira.IssueType{
			ID:      tp["id"].(string),
			Name:    name,
			Handle:  handle,
			Subtask: tp["subtask"].(bool),
		})
	}
	cc.issueTypes = issueTypes

	return nil
}

func (cc *createCmd) getIssueType() *survey.Question {
	var qs *survey.Question

	if cc.params.IssueType == "" {
		var options []string
		for _, t := range cc.issueTypes {
			if t.Handle != "" && t.Handle != t.Name {
				options = append(options, fmt.Sprintf("%s (%s)", t.Name, t.Handle))
			} else {
				options = append(options, t.Name)
			}
		}

		qs = &survey.Question{
			Name: "issueType",
			Prompt: &survey.Select{
				Message: "Issue type",
				Options: options,
			},
			Validate: survey.Required,
		}
	}

	return qs
}

func (cc *createCmd) askQuestions() error {
	it := cc.getIssueType()
	if it != nil {
		ans := struct{ IssueType string }{}
		err := survey.Ask([]*survey.Question{it}, &ans)
		if err != nil {
			return err
		}

		if cc.params.IssueType == "" {
			for _, t := range cc.issueTypes {
				if t.Handle != "" && fmt.Sprintf("%s (%s)", t.Name, t.Handle) == ans.IssueType {
					cc.params.IssueType = t.Handle
				} else if t.Name == ans.IssueType {
					cc.params.IssueType = t.Name
				}
			}
		}
	}

	qs := cc.getRemainingQuestions()
	if len(qs) == 0 {
		return nil
	}

	ans := struct{ ParentIssueKey, Summary, Body string }{}
	err := survey.Ask(qs, &ans)
	if err != nil {
		return err
	}

	project := viper.GetString("project.key")

	if cc.params.ParentIssueKey == "" {
		cc.params.ParentIssueKey = cmdutil.GetJiraIssueKey(project, ans.ParentIssueKey)
	} else {
		cc.params.ParentIssueKey = cmdutil.GetJiraIssueKey(project, cc.params.ParentIssueKey)
	}

	if cc.params.Summary == "" {
		cc.params.Summary = ans.Summary
	}
	if cc.params.Body == "" {
		cc.params.Body = ans.Body
	}

	return nil
}

func (cc *createCmd) getRemainingQuestions() []*survey.Question {
	var qs []*survey.Question

	if cc.params.ParentIssueKey == "" {
		for _, t := range cc.issueTypes {
			if t.Subtask && (t.Name == cc.params.IssueType || (t.Handle != "" && t.Handle == cc.params.IssueType)) {
				qs = append(qs, &survey.Question{
					Name:     "parentIssueKey",
					Prompt:   &survey.Input{Message: "Parent issue key"},
					Validate: survey.Required,
				})
			}
		}
	}

	if cc.params.Summary == "" {
		qs = append(qs, &survey.Question{
			Name:     "summary",
			Prompt:   &survey.Input{Message: "Summary"},
			Validate: survey.Required,
		})
	}

	var defaultBody string

	if cc.params.Template != "" || cmdutil.StdinHasData() {
		b, err := cmdutil.ReadFile(cc.params.Template)
		if err != nil {
			cmdutil.Failed("Error: %s", err)
		}
		defaultBody = string(b)
	}

	if cc.params.NoInput {
		if cc.params.Body == "" {
			cc.params.Body = defaultBody
		}
		return qs
	}

	if cc.params.Body == "" {
		qs = append(qs, &survey.Question{
			Name: "body",
			Prompt: &surveyext.JiraEditor{
				Editor: &survey.Editor{
					Message:       "Description",
					Default:       defaultBody,
					HideDefault:   true,
					AppendDefault: true,
				},
				BlankAllowed: true,
			},
		})
	}

	return qs
}

func (cc *createCmd) isNonInteractive() bool {
	return cmdutil.StdinHasData() || cc.params.Template == "-"
}

func (cc *createCmd) isMandatoryParamsMissing() bool {
	return cc.params.Summary == "" || cc.params.IssueType == ""
}

func parseFlags(flags query.FlagParser) *cmdcommon.CreateParams {
	issueType, err := flags.GetString("type")
	cmdutil.ExitIfError(err)

	parentIssueKey, err := flags.GetString("parent")
	cmdutil.ExitIfError(err)

	summary, err := flags.GetString("summary")
	cmdutil.ExitIfError(err)

	body, err := flags.GetString("body")
	cmdutil.ExitIfError(err)

	priority, err := flags.GetString("priority")
	cmdutil.ExitIfError(err)

	reporter, err := flags.GetString("reporter")
	cmdutil.ExitIfError(err)

	assignee, err := flags.GetString("assignee")
	cmdutil.ExitIfError(err)

	labels, err := flags.GetStringArray("label")
	cmdutil.ExitIfError(err)

	components, err := flags.GetStringArray("component")
	cmdutil.ExitIfError(err)

	fixVersions, err := flags.GetStringArray("fix-version")
	cmdutil.ExitIfError(err)

	affectsVersions, err := flags.GetStringArray("affects-version")
	cmdutil.ExitIfError(err)

	originalEstimate, err := flags.GetString("original-estimate")
	cmdutil.ExitIfError(err)

	custom, err := flags.GetStringToString("custom")
	cmdutil.ExitIfError(err)

	template, err := flags.GetString("template")
	cmdutil.ExitIfError(err)

	jiraTemplate, err := flags.GetString("jira-template")
	cmdutil.ExitIfError(err)

	templateVars, err := flags.GetStringToString("template-var")
	cmdutil.ExitIfError(err)

	noInput, err := flags.GetBool("no-input")
	cmdutil.ExitIfError(err)

	debug, err := flags.GetBool("debug")
	cmdutil.ExitIfError(err)

	noDuplicateCheck, err := flags.GetBool("no-duplicate-check")
	cmdutil.ExitIfError(err)

	return &cmdcommon.CreateParams{
		IssueType:        issueType,
		NoDuplicateCheck: noDuplicateCheck,
		ParentIssueKey:   parentIssueKey,
		Summary:          summary,
		Body:             body,
		Priority:         priority,
		Assignee:         assignee,
		Labels:           labels,
		Reporter:         reporter,
		Components:       components,
		FixVersions:      fixVersions,
		AffectsVersions:  affectsVersions,
		OriginalEstimate: originalEstimate,
		CustomFields:     custom,
		Template:         template,
		JiraTemplate:     jiraTemplate,
		TemplateVars:     templateVars,
		NoInput:          noInput,
		Debug:            debug,
	}
}

func (cc *createCmd) warnOnDuplicateSubtaskType() error {
	if cc.params.NoDuplicateCheck || cc.params.ParentIssueKey == "" || !cc.isSubtaskIssueType() {
		return nil
	}

	subtasks, err := cc.client.GetSubtasks(cc.params.ParentIssueKey)
	if err != nil {
		cmdutil.Warn(
			"Warning: unable to check existing subtasks for %s: %s",
			cc.params.ParentIssueKey,
			err,
		)
		return nil
	}

	targetType := cc.issueTypeName()
	for _, subtask := range subtasks {
		if !strings.EqualFold(subtask.Fields.IssueType.Name, targetType) {
			continue
		}

		message := fmt.Sprintf(
			"Warning: parent %s already has a subtask of type %s (%s). Proceed? [y/N]",
			cc.params.ParentIssueKey,
			targetType,
			subtask.Key,
		)
		if cc.params.NoInput {
			cmdutil.Warn(message)
			return nil
		}

		proceed, err := promptForConfirmation(message)
		if err != nil {
			return err
		}
		if !proceed {
			cmdutil.Failed("Action aborted")
		}

		return nil
	}

	return nil
}

func (cc *createCmd) issueTypeName() string {
	for _, issueType := range cc.issueTypes {
		if strings.EqualFold(cc.params.IssueType, issueType.Name) ||
			(issueType.Handle != "" && strings.EqualFold(cc.params.IssueType, issueType.Handle)) {
			return issueType.Name
		}
	}

	return cc.params.IssueType
}

func (cc *createCmd) isSubtaskIssueType() bool {
	for _, issueType := range cc.issueTypes {
		if !issueType.Subtask {
			continue
		}

		if strings.EqualFold(cc.params.IssueType, issueType.Name) ||
			(issueType.Handle != "" && strings.EqualFold(cc.params.IssueType, issueType.Handle)) {
			return true
		}
	}

	return false
}

func promptForConfirmation(message string) (bool, error) {
	_, err := fmt.Fprintf(os.Stderr, "%s ", message)
	if err != nil {
		return false, err
	}

	answer, err := bufio.NewReader(os.Stdin).ReadString('\n')
	if err != nil {
		return false, err
	}

	answer = strings.TrimSpace(answer)
	return strings.EqualFold(answer, "y") || strings.EqualFold(answer, "yes"), nil
}
