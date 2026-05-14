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
		writeError(stderr, fail.NewValidation("missing notebooks subcommand", "usage: ddctl notebooks <get|create|update|validate> [flags]"))
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
		writeError(stderr, fail.NewValidation("unknown notebooks subcommand", "usage: ddctl notebooks <get|create|update|validate> [flags]"))
		return fail.CodeValidation
	}
}

func runNotebooksGetCmd(ctx context.Context, svcs app.Services, cfg app.Config, args []string, stdout, stderr io.Writer) int {
	leadingID, parseArgs := splitLeadingPositional(args)

	fs := flag.NewFlagSet("notebooks get", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	includeMetadata := fs.Bool("include-metadata", true, "include notebook metadata")
	if err := fs.Parse(parseArgs); err != nil {
		writeError(stderr, fail.NewValidation(err.Error(), "usage: ddctl notebooks get <id> [--include-metadata]"))
		return fail.CodeValidation
	}
	notebookID := leadingID
	if notebookID == "" && fs.NArg() > 0 {
		notebookID = fs.Arg(0)
	}
	if notebookID == "" {
		writeError(stderr, fail.NewValidation("missing notebook ID", "usage: ddctl notebooks get <id>"))
		return fail.CodeValidation
	}

	result, err := svcs.Notebooks.Get(ctx, service.NotebookGetInput{
		ID:              notebookID,
		IncludeMetadata: *includeMetadata,
	})
	if err != nil {
		writeError(stderr, err)
		return fail.ExitCode(err)
	}
	if cfg.JSON {
		if err := svcs.Output.JSON(stdout, result); err != nil {
			writeError(stderr, fail.NewAPI(err.Error(), "unable to encode notebook result", ""))
			return fail.CodeAPI
		}
		return fail.CodeOK
	}
	printNotebookSummary(stdout, result)
	return fail.CodeOK
}

func runNotebooksCreateCmd(ctx context.Context, svcs app.Services, cfg app.Config, args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("notebooks create", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	fromFile := fs.String("from-file", "", "path to notebook JSON payload")
	name := fs.String("name", "", "override notebook name")
	timeSpan := fs.String("time", "", "override live_span (e.g. 1w)")
	if err := fs.Parse(args); err != nil {
		writeError(stderr, fail.NewValidation(err.Error(), "usage: ddctl notebooks create --from-file <path> [--name <name>] [--time <live_span>]"))
		return fail.CodeValidation
	}

	result, err := svcs.Notebooks.Create(ctx, service.NotebookMutationInput{
		FilePath: *fromFile,
		Name:     *name,
		Time:     *timeSpan,
	})
	if err != nil {
		writeError(stderr, err)
		return fail.ExitCode(err)
	}
	if cfg.JSON {
		if err := svcs.Output.JSON(stdout, result); err != nil {
			writeError(stderr, fail.NewAPI(err.Error(), "unable to encode notebook result", ""))
			return fail.CodeAPI
		}
		return fail.CodeOK
	}
	printNotebookSummary(stdout, result)
	return fail.CodeOK
}

func runNotebooksUpdateCmd(ctx context.Context, svcs app.Services, cfg app.Config, args []string, stdout, stderr io.Writer) int {
	leadingID, parseArgs := splitLeadingPositional(args)

	fs := flag.NewFlagSet("notebooks update", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	fromFile := fs.String("from-file", "", "path to notebook JSON payload")
	replaceAll := fs.Bool("replace-all", false, "confirm full replacement update")
	if err := fs.Parse(parseArgs); err != nil {
		writeError(stderr, fail.NewValidation(err.Error(), "usage: ddctl notebooks update <id> --from-file <path> --replace-all"))
		return fail.CodeValidation
	}
	notebookID := leadingID
	if notebookID == "" && fs.NArg() > 0 {
		notebookID = fs.Arg(0)
	}
	if notebookID == "" {
		writeError(stderr, fail.NewValidation("missing notebook ID", "usage: ddctl notebooks update <id> --from-file <path> --replace-all"))
		return fail.CodeValidation
	}

	result, err := svcs.Notebooks.Update(ctx, service.NotebookMutationInput{
		ID:       notebookID,
		FilePath: *fromFile,
	}, *replaceAll)
	if err != nil {
		writeError(stderr, err)
		return fail.ExitCode(err)
	}
	if cfg.JSON {
		if err := svcs.Output.JSON(stdout, result); err != nil {
			writeError(stderr, fail.NewAPI(err.Error(), "unable to encode notebook result", ""))
			return fail.CodeAPI
		}
		return fail.CodeOK
	}
	printNotebookSummary(stdout, result)
	return fail.CodeOK
}

func runNotebooksValidateCmd(ctx context.Context, svcs app.Services, cfg app.Config, args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("notebooks validate", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	fromFile := fs.String("from-file", "", "path to notebook JSON payload")
	from := fs.String("from", "now-30d", "metrics validation start time")
	to := fs.String("to", "now", "metrics validation end time")
	allowEmpty := fs.Bool("allow-empty-series", false, "allow timeseries metric queries with no data")
	if err := fs.Parse(args); err != nil {
		writeError(stderr, fail.NewValidation(err.Error(), "usage: ddctl notebooks validate --from-file <path> [--from <time>] [--to <time>] [--allow-empty-series]"))
		return fail.CodeValidation
	}

	result, err := svcs.Notebooks.Validate(ctx, service.NotebookValidateInput{
		FilePath:         *fromFile,
		From:             *from,
		To:               *to,
		AllowEmptySeries: *allowEmpty,
	})
	if err != nil {
		writeError(stderr, err)
		return fail.ExitCode(err)
	}
	if cfg.JSON {
		if err := svcs.Output.JSON(stdout, result); err != nil {
			writeError(stderr, fail.NewAPI(err.Error(), "unable to encode notebook validation result", ""))
			return fail.CodeAPI
		}
		return fail.CodeOK
	}

	fmt.Fprintf(stdout, "timeseries queries: %d\n", result.QueryCount)
	for _, q := range result.Queries {
		fmt.Fprintf(stdout, "  - %s\n", q)
	}
	if len(result.Warnings) > 0 {
		fmt.Fprintln(stdout, "warnings:")
		for _, w := range result.Warnings {
			fmt.Fprintf(stdout, "  - %s\n", w)
		}
	}
	return fail.CodeOK
}

func printNotebookSummary(w io.Writer, payload map[string]any) {
	data, _ := payload["data"].(map[string]any)
	attrs, _ := data["attributes"].(map[string]any)
	name, _ := attrs["name"].(string)
	id := normalizeNotebookID(data["id"])
	cells, _ := attrs["cells"].([]any)

	fmt.Fprintf(w, "ID: %s\n", id)
	fmt.Fprintf(w, "Name: %s\n", name)
	fmt.Fprintf(w, "Cells: %d\n", len(cells))
	if id != "" {
		fmt.Fprintf(w, "URL: https://app.datadoghq.com/notebook/%s\n", id)
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
