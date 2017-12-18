package events

import (
	"bytes"
	"fmt"
	"strings"
	"text/template"
)

// MarkdownRenderer renders responses as markdown.
type MarkdownRenderer struct{}

// CommonData is data that all responses have.
type CommonData struct {
	Command string
	Verbose bool
	Log     string
}

// ErrData is data about an error response.
type ErrData struct {
	Error string
	CommonData
}

// FailureData is data about a failure response.
type FailureData struct {
	Failure string
	CommonData
}

// ResultData is data about a successful response.
type ResultData struct {
	Results map[string]string
	CommonData
}

// Render formats the data into a markdown string.
// nolint: interfacer
func (g *MarkdownRenderer) Render(res CommandResponse, cmdName CommandName, log string, verbose bool) string {
	if cmdName == Help {
		return g.renderTemplate(helpTmpl, nil)
	}
	commandStr := strings.Title(cmdName.String())
	common := CommonData{commandStr, verbose, log}
	if res.Error != nil {
		return g.renderTemplate(errWithLogTmpl, ErrData{res.Error.Error(), common})
	}
	if res.Failure != "" {
		return g.renderTemplate(failureWithLogTmpl, FailureData{res.Failure, common})
	}
	return g.renderProjectResults(res.ProjectResults, common)
}

func (g *MarkdownRenderer) renderProjectResults(pathResults []ProjectResult, common CommonData) string {
	results := make(map[string]string)
	for _, result := range pathResults {
		if result.Error != nil {
			results[result.Path] = g.renderTemplate(errTmpl, struct {
				Command string
				Error   string
			}{
				Command: common.Command,
				Error:   result.Error.Error(),
			})
		} else if result.Failure != "" {
			results[result.Path] = g.renderTemplate(failureTmpl, struct {
				Command string
				Failure string
			}{
				Command: common.Command,
				Failure: result.Failure,
			})
		} else if result.PlanSuccess != nil {
			results[result.Path] = g.renderTemplate(planSuccessTmpl, *result.PlanSuccess)
		} else if result.ApplySuccess != "" {
			results[result.Path] = g.renderTemplate(applySuccessTmpl, struct{ Output string }{result.ApplySuccess})
		} else {
			results[result.Path] = "Found no template. This is a bug!"
		}
	}

	var tmpl *template.Template
	if len(results) == 1 {
		tmpl = singleProjectTmpl
	} else {
		tmpl = multiProjectTmpl
	}
	return g.renderTemplate(tmpl, ResultData{results, common})
}

func (g *MarkdownRenderer) renderTemplate(tmpl *template.Template, data interface{}) string {
	buf := &bytes.Buffer{}
	if err := tmpl.Execute(buf, data); err != nil {
		return fmt.Sprintf("Failed to render template, this is a bug: %v", err)
	}
	return buf.String()
}

var helpTmpl = template.Must(template.New("").Parse("```cmake\n" +
	`atlantis - Terraform collaboration tool that enables you to collaborate on infrastructure
safely and securely.

Usage: atlantis <command> [workspace] [--verbose]

Commands:
plan           Runs 'terraform plan' on the files changed in the pull request
apply          Runs 'terraform apply' using the plans generated by 'atlantis plan'
help           Get help

Examples:

# Generates a plan for staging workspace
atlantis plan staging

# Generates a plan for a standalone terraform project
atlantis plan

# Applies a plan for staging workspace
atlantis apply staging

# Applies a plan for a standalone terraform project
atlantis apply
`))
var singleProjectTmpl = template.Must(template.New("").Parse("{{ range $result := .Results }}{{$result}}{{end}}\n" + logTmpl))
var multiProjectTmpl = template.Must(template.New("").Parse(
	"Ran {{.Command}} in {{ len .Results }} directories:\n" +
		"{{ range $path, $result := .Results }}" +
		" * `{{$path}}`\n" +
		"{{end}}\n" +
		"{{ range $path, $result := .Results }}" +
		"## {{$path}}/\n" +
		"{{$result}}\n" +
		"---\n{{end}}" +
		logTmpl))
var planSuccessTmpl = template.Must(template.New("").Parse(
	"```diff\n" +
		"{{.TerraformOutput}}\n" +
		"```\n\n" +
		"* To **discard** this plan click [here]({{.LockURL}})."))
var applySuccessTmpl = template.Must(template.New("").Parse(
	"```diff\n" +
		"{{.Output}}\n" +
		"```"))
var errTmplText = "**{{.Command}} Error**\n" +
	"```\n" +
	"{{.Error}}\n" +
	"```\n"
var errTmpl = template.Must(template.New("").Parse(errTmplText))
var errWithLogTmpl = template.Must(template.New("").Parse(errTmplText + logTmpl))
var failureTmplText = "**{{.Command}} Failed**: {{.Failure}}\n"
var failureTmpl = template.Must(template.New("").Parse(failureTmplText))
var failureWithLogTmpl = template.Must(template.New("").Parse(failureTmplText + logTmpl))
var logTmpl = "{{if .Verbose}}\n<details><summary>Log</summary>\n  <p>\n\n```\n{{.Log}}```\n</p></details>{{end}}\n"
