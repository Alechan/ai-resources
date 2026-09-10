package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/Alechan/ai-resources/tools/ddctl/src/internal/app"
	"github.com/Alechan/ai-resources/tools/ddctl/src/internal/fail"
	"github.com/Alechan/ai-resources/tools/ddctl/src/internal/service"
)

func runNotebooksCmd(ctx context.Context, svcs app.Services, cfg app.Config, args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		writeError(stderr, fail.NewValidation("missing notebooks subcommand", "usage: ddctl notebooks <get|create|update|validate> [flags]"), cfg)
		return fail.CodeValidation
	}

	switch args[0] {
	case "get":
		return runNotebooksGetCmd(ctx, svcs, cfg, args[1:], stdout, stderr)
	case "create":
		return runNotebooksCreateCmd(ctx, svcs, cfg, args[1:], stdout, stderr)
	case "update":
		return runNotebooksUpdateCmd(ctx, svcs, cfg, args[1:], stdout, stderr)
	case "validate":
		return runNotebooksValidateCmd(ctx, svcs, cfg, args[1:], stdout, stderr)
	default:
		writeError(stderr, fail.NewValidation("unknown notebooks subcommand", "usage: ddctl notebooks <get|create|update|validate> [flags]"), cfg)
		return fail.CodeValidation
	}
}

func runNotebooksGetCmd(ctx context.Context, svcs app.Services, cfg app.Config, args []string, stdout, stderr io.Writer) int {
	leadingID, parseArgs := splitLeadingPositional(args)

	fs := flag.NewFlagSet("notebooks get", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	includeMetadata := fs.Bool("include-metadata", true, "include notebook metadata")
	raw := fs.Bool("raw", false, "with --json, emit the full Datadog response")
	if err := fs.Parse(parseArgs); err != nil {
		writeError(stderr, fail.NewValidation(err.Error(), "usage: ddctl notebooks get <id> [--include-metadata]"), cfg)
		return fail.CodeValidation
	}
	notebookID := leadingID
	if notebookID == "" && fs.NArg() > 0 {
		notebookID = fs.Arg(0)
	}
	if notebookID == "" {
		writeError(stderr, fail.NewValidation("missing notebook ID", "usage: ddctl notebooks get <id>"), cfg)
		return fail.CodeValidation
	}

	result, err := svcs.Notebooks.Get(ctx, service.NotebookGetInput{
		ID:              notebookID,
		IncludeMetadata: *includeMetadata,
	})
	if err != nil {
		writeError(stderr, err, cfg)
		return fail.ExitCode(err)
	}
	return writeNotebookResult(svcs, cfg, stdout, stderr, result, *raw)
}

func runNotebooksCreateCmd(ctx context.Context, svcs app.Services, cfg app.Config, args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("notebooks create", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	fromFile := fs.String("from-file", "", "path to notebook JSON payload")
	name := fs.String("name", "", "override notebook name")
	timeSpan := fs.String("time", "", "override live_span (e.g. 1w)")
	skipValidate := fs.Bool("skip-validate", false, "skip query preflight before create")
	from := fs.String("from", "now-30d", "metrics validation start time")
	to := fs.String("to", "now", "metrics validation end time")
	dryRun := fs.Bool("dry-run", false, "validate and print summary without POST")
	raw := fs.Bool("raw", false, "with --json, emit the full Datadog response")
	if err := fs.Parse(args); err != nil {
		writeError(stderr, fail.NewValidation(err.Error(), "usage: ddctl notebooks create --from-file <path>"), cfg)
		return fail.CodeValidation
	}

	result, err := svcs.Notebooks.Create(ctx, service.NotebookMutationInput{
		FilePath:     *fromFile,
		Name:         *name,
		Time:         *timeSpan,
		SkipValidate: *skipValidate,
		From:         *from,
		To:           *to,
		DryRun:       *dryRun,
	})
	if err != nil {
		writeError(stderr, err, cfg)
		return fail.ExitCode(err)
	}
	if dry, _ := result["dry_run"].(bool); dry && !cfg.JSON {
		fmt.Fprintln(stdout, "dry-run: would create notebook")
		printNotebookDryRun(stdout, result)
		return fail.CodeOK
	}
	return writeNotebookResult(svcs, cfg, stdout, stderr, result, *raw)
}

func runNotebooksUpdateCmd(ctx context.Context, svcs app.Services, cfg app.Config, args []string, stdout, stderr io.Writer) int {
	leadingID, parseArgs := splitLeadingPositional(args)

	fs := flag.NewFlagSet("notebooks update", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	fromFile := fs.String("from-file", "", "path to notebook JSON payload")
	replaceAll := fs.Bool("replace-all", false, "confirm full replacement update")
	skipValidate := fs.Bool("skip-validate", false, "skip query preflight before update")
	from := fs.String("from", "now-30d", "metrics validation start time")
	to := fs.String("to", "now", "metrics validation end time")
	dryRun := fs.Bool("dry-run", false, "validate and print diff without PUT")
	showDiff := fs.Bool("diff", false, "include semantic diff in successful update output")
	ifUnmodified := fs.String("if-unmodified-since", "", "abort if remote modified_at does not match")
	raw := fs.Bool("raw", false, "with --json, emit the full Datadog response")
	if err := fs.Parse(parseArgs); err != nil {
		writeError(stderr, fail.NewValidation(err.Error(), "usage: ddctl notebooks update <id> --from-file <path> --replace-all"), cfg)
		return fail.CodeValidation
	}
	notebookID := leadingID
	if notebookID == "" && fs.NArg() > 0 {
		notebookID = fs.Arg(0)
	}
	if notebookID == "" {
		writeError(stderr, fail.NewValidation("missing notebook ID", "usage: ddctl notebooks update <id> --from-file <path> --replace-all"), cfg)
		return fail.CodeValidation
	}

	result, err := svcs.Notebooks.Update(ctx, service.NotebookMutationInput{
		ID:                notebookID,
		FilePath:          *fromFile,
		SkipValidate:      *skipValidate,
		From:              *from,
		To:                *to,
		DryRun:            *dryRun,
		ShowDiff:          *showDiff,
		IfUnmodifiedSince: *ifUnmodified,
	}, *replaceAll)
	if err != nil {
		writeError(stderr, err, cfg)
		return fail.ExitCode(err)
	}
	if dry, _ := result["dry_run"].(bool); dry && !cfg.JSON {
		fmt.Fprintf(stdout, "dry-run: would update notebook %s\n", notebookID)
		printNotebookDryRun(stdout, result)
		return fail.CodeOK
	}
	return writeNotebookResult(svcs, cfg, stdout, stderr, result, *raw)
}

func runNotebooksValidateCmd(ctx context.Context, svcs app.Services, cfg app.Config, args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("notebooks validate", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	fromFile := fs.String("from-file", "", "path to notebook JSON payload")
	from := fs.String("from", "now-30d", "metrics validation start time")
	to := fs.String("to", "now", "metrics validation end time")
	if err := fs.Parse(args); err != nil {
		writeError(stderr, fail.NewValidation(err.Error(), "usage: ddctl notebooks validate --from-file <path> [--from <time>] [--to <time>]"), cfg)
		return fail.CodeValidation
	}

	result, err := svcs.Notebooks.Validate(ctx, service.NotebookValidateInput{
		FilePath: *fromFile,
		From:     *from,
		To:       *to,
	})
	if err != nil {
		writeError(stderr, err, cfg)
		return fail.ExitCode(err)
	}
	if cfg.JSON {
		if err := svcs.Output.JSON(stdout, result); err != nil {
			writeError(stderr, fail.NewAPI(err.Error(), "unable to encode notebook validation result", ""), cfg)
			return fail.CodeAPI
		}
		return fail.CodeOK
	}

	fmt.Fprintf(stdout, "structure: %s\n", result.Structure)
	fmt.Fprintf(stdout, "metric queries: %d\n", result.QueryCount)
	for _, q := range result.Queries {
		fmt.Fprintf(stdout, "  - cell[%d] request[%d] original=%q resolved=%q", q.CellIndex, q.RequestIndex, q.Original, q.Resolved)
		if q.NoData {
			fmt.Fprint(stdout, " (no data)")
		}
		fmt.Fprintln(stdout)
	}
	if len(result.Warnings) > 0 {
		fmt.Fprintln(stdout, "warnings:")
		for _, w := range result.Warnings {
			fmt.Fprintf(stdout, "  - %s\n", w)
		}
	}
	return fail.CodeOK
}

func writeNotebookResult(svcs app.Services, cfg app.Config, stdout, stderr io.Writer, result map[string]any, raw bool) int {
	if cfg.JSON {
		payload := map[string]any(result)
		if !raw {
			if _, ok := result["data"]; ok {
				payload = service.NotebookMutationSummary(result, cfg.Site)
				if diff, ok := result["diff"].(string); ok && diff != "" {
					payload["diff"] = diff
				}
			}
		}
		if err := svcs.Output.JSON(stdout, payload); err != nil {
			writeError(stderr, fail.NewAPI(err.Error(), "unable to encode notebook result", ""), cfg)
			return fail.CodeAPI
		}
		return fail.CodeOK
	}
	printNotebookSummary(stdout, cfg.Site, result)
	if diff, _ := result["diff"].(string); diff != "" {
		fmt.Fprintf(stdout, "Diff: %s\n", diff)
	}
	return fail.CodeOK
}

func printNotebookDryRun(w io.Writer, result map[string]any) {
	if name, _ := result["name"].(string); name != "" {
		fmt.Fprintf(w, "Name:  %s\n", name)
	}
	if n, ok := result["cell_count"].(int); ok {
		fmt.Fprintf(w, "Cells: %d\n", n)
	}
	if id, _ := result["id"].(string); id != "" {
		fmt.Fprintf(w, "ID:    %s\n", id)
	}
	if url, _ := result["url"].(string); url != "" {
		fmt.Fprintf(w, "URL:   %s\n", url)
	}
	if diff, _ := result["diff"].(string); diff != "" {
		fmt.Fprintf(w, "Diff:  %s\n", diff)
	}
}

func printNotebookSummary(w io.Writer, site string, payload map[string]any) {
	if _, ok := payload["data"]; !ok {
		printNotebookSummaryFields(w, payload)
		return
	}
	summary := service.NotebookMutationSummary(payload, site)
	printNotebookSummaryFields(w, summary)
}

func printNotebookSummaryFields(w io.Writer, summary map[string]any) {
	if id := fmt.Sprint(summary["id"]); id != "" && id != "<nil>" {
		fmt.Fprintf(w, "ID:         %s\n", id)
	}
	if name, _ := summary["name"].(string); name != "" {
		fmt.Fprintf(w, "Name:       %s\n", name)
	}
	if status, _ := summary["status"].(string); status != "" {
		fmt.Fprintf(w, "Status:     %s\n", status)
	}
	if count, ok := summary["cell_count"].(int); ok {
		fmt.Fprintf(w, "Cell count: %d\n", count)
	}
	if ids, ok := summary["cell_ids"].([]string); ok && len(ids) > 0 {
		fmt.Fprintf(w, "Cell IDs:   %s\n", strings.Join(ids, ", "))
	}
	if url, _ := summary["url"].(string); url != "" {
		fmt.Fprintf(w, "URL:        %s\n", url)
	}
}

func splitLeadingPositional(args []string) (string, []string) {
	if len(args) == 0 {
		return "", args
	}
	if strings.HasPrefix(args[0], "-") {
		return "", args
	}
	return args[0], args[1:]
}

func normalizeNotebookID(v any) string {
	switch t := v.(type) {
	case string:
		return t
	case int:
		return strconv.Itoa(t)
	case int64:
		return strconv.FormatInt(t, 10)
	case float64:
		return strconv.FormatInt(int64(t), 10)
	default:
		return fmt.Sprintf("%v", v)
	}
}
