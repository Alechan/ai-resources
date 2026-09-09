package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"strings"

	"github.com/Alechan/ai-resources/tools/ddctl/src/internal/app"
	"github.com/Alechan/ai-resources/tools/ddctl/src/internal/fail"
	"github.com/Alechan/ai-resources/tools/ddctl/src/internal/service"
)

type repeatableStrings []string

func (r *repeatableStrings) String() string { return strings.Join(*r, ",") }

func (r *repeatableStrings) Set(v string) error {
	*r = append(*r, v)
	return nil
}

func parseTemplateVariableFlags(pairs []string) (map[string]string, error) {
	out := map[string]string{}
	for _, p := range pairs {
		name, val, ok := strings.Cut(p, "=")
		name = strings.TrimSpace(name)
		if !ok || name == "" {
			return nil, fail.NewValidation("invalid --template-variable", "use name=value (repeatable)")
		}
		out[name] = val
	}
	if len(out) == 0 {
		return nil, nil
	}
	return out, nil
}

func mergeUnmodifiedSince(expected, ifUnmodified string) (string, error) {
	expected = strings.TrimSpace(expected)
	ifUnmodified = strings.TrimSpace(ifUnmodified)
	if expected != "" && ifUnmodified != "" && expected != ifUnmodified {
		return "", fail.NewValidation("conflicting concurrency flags", "pass only one of --if-unmodified-since and --expected-modified-at")
	}
	if ifUnmodified != "" {
		return ifUnmodified, nil
	}
	return expected, nil
}

func runDashboardsCmd(ctx context.Context, svcs app.Services, cfg app.Config, args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		writeError(stderr, fail.NewValidation("missing dashboards subcommand", "usage: ddctl dashboards <get|validate|create|update> [flags]"))
		return fail.CodeValidation
	}

	switch args[0] {
	case "get":
		return runDashboardsGetCmd(ctx, svcs, cfg, args[1:], stdout, stderr)
	case "create":
		return runDashboardsCreateCmd(ctx, svcs, cfg, args[1:], stdout, stderr)
	case "update":
		return runDashboardsUpdateCmd(ctx, svcs, cfg, args[1:], stdout, stderr)
	case "validate":
		return runDashboardsValidateCmd(ctx, svcs, cfg, args[1:], stdout, stderr)
	default:
		writeError(stderr, fail.NewValidation("unknown dashboards subcommand", "usage: ddctl dashboards <get|validate|create|update> [flags]"))
		return fail.CodeValidation
	}
}

func runDashboardsGetCmd(ctx context.Context, svcs app.Services, cfg app.Config, args []string, stdout, stderr io.Writer) int {
	leadingID, parseArgs := splitLeadingPositional(args)
	fs := flag.NewFlagSet("dashboards get", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	if err := fs.Parse(parseArgs); err != nil {
		writeError(stderr, fail.NewValidation(err.Error(), "usage: ddctl dashboards get <id>"))
		return fail.CodeValidation
	}
	dashboardID := leadingID
	if dashboardID == "" && fs.NArg() > 0 {
		dashboardID = fs.Arg(0)
	}
	if dashboardID == "" {
		writeError(stderr, fail.NewValidation("missing dashboard ID", "usage: ddctl dashboards get <id>"))
		return fail.CodeValidation
	}

	result, err := svcs.Dashboards.Get(ctx, service.DashboardGetInput{ID: dashboardID})
	if err != nil {
		writeError(stderr, err)
		return fail.ExitCode(err)
	}
	return writeDashboardResult(svcs, cfg, stdout, stderr, result)
}

func runDashboardsCreateCmd(ctx context.Context, svcs app.Services, cfg app.Config, args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("dashboards create", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	fromFile := fs.String("from-file", "", "path to dashboard JSON payload")
	title := fs.String("title", "", "override dashboard title")
	skipValidate := fs.Bool("skip-validate", false, "skip query preflight before create")
	allowEmpty := fs.Bool("allow-empty-series", false, "no-data is always a warning; flag kept for compatibility")
	from := fs.String("from", "now-30d", "query preflight start time")
	to := fs.String("to", "now", "query preflight end time")
	dryRun := fs.Bool("dry-run", false, "validate and print summary; do not POST")
	var tvs repeatableStrings
	fs.Var(&tvs, "template-variable", "substitute $name.value in queries (name=value, repeatable)")
	if err := fs.Parse(args); err != nil {
		writeError(stderr, fail.NewValidation(err.Error(), "usage: ddctl dashboards create --from-file <path> [--title <title>]"))
		return fail.CodeValidation
	}
	vars, err := parseTemplateVariableFlags(tvs)
	if err != nil {
		writeError(stderr, err)
		return fail.ExitCode(err)
	}

	result, err := svcs.Dashboards.Create(ctx, service.DashboardMutationInput{
		FilePath:          *fromFile,
		Title:             *title,
		SkipValidate:      *skipValidate,
		AllowEmptySeries:  *allowEmpty,
		From:              *from,
		To:                *to,
		DryRun:            *dryRun,
		TemplateVariables: vars,
	})
	if err != nil {
		writeError(stderr, err)
		return fail.ExitCode(err)
	}
	if dry, _ := result["dry_run"].(bool); dry && !cfg.JSON {
		fmt.Fprintln(stdout, "dry-run: would create dashboard")
		printDashboardSummary(stdout, result)
		return fail.CodeOK
	}
	return writeDashboardResult(svcs, cfg, stdout, stderr, result)
}

func runDashboardsUpdateCmd(ctx context.Context, svcs app.Services, cfg app.Config, args []string, stdout, stderr io.Writer) int {
	leadingID, parseArgs := splitLeadingPositional(args)
	fs := flag.NewFlagSet("dashboards update", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	fromFile := fs.String("from-file", "", "path to dashboard JSON payload")
	replaceAll := fs.Bool("replace-all", false, "confirm full replacement update")
	dryRun := fs.Bool("dry-run", false, "show semantic diff against current dashboard and do not PUT")
	showDiff := fs.Bool("diff", false, "show semantic diff against the current dashboard")
	expectedModified := fs.String("expected-modified-at", "", "abort if remote modified_at does not match")
	ifUnmodified := fs.String("if-unmodified-since", "", "abort if remote modified_at does not match")
	skipValidate := fs.Bool("skip-validate", false, "skip query preflight before update")
	allowEmpty := fs.Bool("allow-empty-series", false, "no-data is always a warning; flag kept for compatibility")
	from := fs.String("from", "now-30d", "query preflight start time")
	to := fs.String("to", "now", "query preflight end time")
	var tvs repeatableStrings
	fs.Var(&tvs, "template-variable", "substitute $name.value in queries (name=value, repeatable)")
	if err := fs.Parse(parseArgs); err != nil {
		writeError(stderr, fail.NewValidation(err.Error(), "usage: ddctl dashboards update <id> --from-file <path> --replace-all"))
		return fail.CodeValidation
	}
	dashboardID := leadingID
	if dashboardID == "" && fs.NArg() > 0 {
		dashboardID = fs.Arg(0)
	}
	if dashboardID == "" {
		writeError(stderr, fail.NewValidation("missing dashboard ID", "usage: ddctl dashboards update <id> --from-file <path> --replace-all"))
		return fail.CodeValidation
	}
	expected, err := mergeUnmodifiedSince(*expectedModified, *ifUnmodified)
	if err != nil {
		writeError(stderr, err)
		return fail.ExitCode(err)
	}
	vars, err := parseTemplateVariableFlags(tvs)
	if err != nil {
		writeError(stderr, err)
		return fail.ExitCode(err)
	}

	result, err := svcs.Dashboards.Update(ctx, service.DashboardMutationInput{
		FilePath:           *fromFile,
		ID:                 dashboardID,
		SkipValidate:       *skipValidate,
		AllowEmptySeries:   *allowEmpty,
		From:               *from,
		To:                 *to,
		DryRun:             *dryRun,
		ShowDiff:           *showDiff,
		ExpectedModifiedAt: expected,
		TemplateVariables:  vars,
	}, *replaceAll)
	if err != nil {
		writeError(stderr, err)
		return fail.ExitCode(err)
	}
	if dry, _ := result["dry_run"].(bool); dry && !cfg.JSON {
		fmt.Fprintf(stdout, "dry-run: would update dashboard %s\n", dashboardID)
		if url, _ := result["url"].(string); url != "" {
			fmt.Fprintf(stdout, "URL: %s\n", url)
		}
		if diff, _ := result["diff"].(string); diff != "" {
			fmt.Fprint(stdout, diff)
		}
		return fail.CodeOK
	}
	if !cfg.JSON {
		if diff, _ := result["diff"].(string); diff != "" {
			fmt.Fprint(stdout, diff)
		}
	}
	return writeDashboardResult(svcs, cfg, stdout, stderr, result)
}

func runDashboardsValidateCmd(ctx context.Context, svcs app.Services, cfg app.Config, args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("dashboards validate", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	fromFile := fs.String("from-file", "", "path to dashboard JSON payload")
	from := fs.String("from", "now-30d", "query preflight start time")
	to := fs.String("to", "now", "query preflight end time")
	allowEmpty := fs.Bool("allow-empty-series", false, "no-data is always a warning; flag kept for compatibility")
	var tvs repeatableStrings
	fs.Var(&tvs, "template-variable", "substitute $name.value in queries (name=value, repeatable)")
	if err := fs.Parse(args); err != nil {
		writeError(stderr, fail.NewValidation(err.Error(), "usage: ddctl dashboards validate --from-file <path>"))
		return fail.CodeValidation
	}
	vars, err := parseTemplateVariableFlags(tvs)
	if err != nil {
		writeError(stderr, err)
		return fail.ExitCode(err)
	}

	result, err := svcs.Dashboards.Validate(ctx, service.DashboardValidateInput{
		FilePath:          *fromFile,
		From:              *from,
		To:                *to,
		AllowEmptySeries:  *allowEmpty,
		TemplateVariables: vars,
	})
	if err != nil {
		writeError(stderr, err)
		return fail.ExitCode(err)
	}
	if cfg.JSON {
		if err := svcs.Output.JSON(stdout, result); err != nil {
			writeError(stderr, fail.NewAPI(err.Error(), "unable to encode dashboard validation result", ""))
			return fail.CodeAPI
		}
		return fail.CodeOK
	}
	printDashboardValidate(stdout, result)
	return fail.CodeOK
}

func writeDashboardResult(svcs app.Services, cfg app.Config, stdout, stderr io.Writer, result map[string]any) int {
	if cfg.JSON {
		if err := svcs.Output.JSON(stdout, result); err != nil {
			writeError(stderr, fail.NewAPI(err.Error(), "unable to encode dashboard result", ""))
			return fail.CodeAPI
		}
		return fail.CodeOK
	}
	printDashboardSummary(stdout, result)
	return fail.CodeOK
}

func printDashboardSummary(w io.Writer, payload map[string]any) {
	id := stringsOrSprint(payload["id"])
	title, _ := payload["title"].(string)
	layout, _ := payload["layout_type"].(string)
	widgets, _ := payload["widgets"].([]any)
	url, _ := payload["url"].(string)
	if url == "" && id != "" {
		url = dashboardURL("datadoghq.com", id)
	}

	fmt.Fprintf(w, "ID:      %s\n", id)
	fmt.Fprintf(w, "Title:   %s\n", title)
	fmt.Fprintf(w, "Layout:  %s\n", layout)
	fmt.Fprintf(w, "Widgets: %d\n", widgetCount(payload, widgets))
	if url != "" {
		fmt.Fprintf(w, "URL:     %s\n", url)
	}
}

func printDashboardValidate(w io.Writer, result service.DashboardValidateResult) {
	structure := result.Structure
	if structure == "" {
		structure = "valid"
	}
	fmt.Fprintf(w, "dashboard structure: %s\n", structure)
	fmt.Fprintf(w, "metric queries: %d valid, %d no-data\n", result.MetricsValid, result.MetricsNoData)
	fmt.Fprintf(w, "log queries: %d valid, %d no-data\n", result.LogsValid, result.LogsNoData)
	fmt.Fprintf(w, "monitor references: %d valid\n", result.MonitorsValid)
	if len(result.Skipped) > 0 {
		fmt.Fprintln(w, "skipped:")
		for _, q := range result.Skipped {
			fmt.Fprintf(w, "  - %s: %s\n", q.WidgetType, q.SkippedReason)
		}
	}
	if len(result.Warnings) > 0 {
		fmt.Fprintln(w, "warnings:")
		for _, wmsg := range result.Warnings {
			fmt.Fprintf(w, "  - %s\n", wmsg)
		}
	}
}

func dashboardURL(site, id string) string {
	if site == "" {
		site = "datadoghq.com"
	}
	if id == "" {
		return ""
	}
	return fmt.Sprintf("https://app.%s/dashboard/%s", site, id)
}

func widgetCount(payload map[string]any, widgets []any) int {
	if len(widgets) > 0 {
		return len(widgets)
	}
	switch n := payload["widget_count"].(type) {
	case int:
		return n
	case float64:
		return int(n)
	default:
		return len(widgets)
	}
}

func stringsOrSprint(v any) string {
	if v == nil {
		return ""
	}
	if s, ok := v.(string); ok {
		return s
	}
	return fmt.Sprint(v)
}
