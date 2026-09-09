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

func runMonitorsCmd(ctx context.Context, svcs app.Services, cfg app.Config, args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		writeError(stderr, fail.NewValidation("missing monitors subcommand", "usage: ddctl monitors <list|get|validate|create|update|mute|unmute> [flags]"), cfg)
		return fail.CodeValidation
	}
	switch args[0] {
	case "list":
		return runMonitorsListCmd(ctx, svcs, cfg, args[1:], stdout, stderr)
	case "get":
		return runMonitorsGetCmd(ctx, svcs, cfg, args[1:], stdout, stderr)
	case "validate":
		return runMonitorsValidateCmd(ctx, svcs, cfg, args[1:], stdout, stderr)
	case "create":
		return runMonitorsCreateCmd(ctx, svcs, cfg, args[1:], stdout, stderr)
	case "update":
		return runMonitorsUpdateCmd(ctx, svcs, cfg, args[1:], stdout, stderr)
	case "mute":
		return runMonitorsMuteCmd(ctx, svcs, cfg, args[1:], stdout, stderr)
	case "unmute":
		return runMonitorsUnmuteCmd(ctx, svcs, cfg, args[1:], stdout, stderr)
	default:
		writeError(stderr, fail.NewValidation("unknown monitors subcommand", "valid: list, get, validate, create, update, mute, unmute"), cfg)
		return fail.CodeValidation
	}
}

func runMonitorsListCmd(ctx context.Context, svcs app.Services, cfg app.Config, args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("monitors list", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	tag := fs.String("tag", "", "filter monitors by tag")
	if err := fs.Parse(args); err != nil {
		writeError(stderr, fail.NewValidation(err.Error(), "usage: ddctl monitors list [--tag <tag>]"), cfg)
		return fail.CodeValidation
	}
	result, err := svcs.Monitors.List(ctx, *tag)
	if err != nil {
		writeError(stderr, err, cfg)
		return fail.ExitCode(err)
	}
	if cfg.JSON {
		return writeJSONResult(svcs, cfg, stdout, stderr, result)
	}
	for _, m := range result.Monitors {
		fmt.Fprintf(stdout, "[%d] %-10s %-8s %s  tags:%s\n  %s\n",
			m.ID, m.OverallState, m.Type, m.Name, strings.Join(m.Tags, ","), m.URL)
	}
	return fail.CodeOK
}

func runMonitorsGetCmd(ctx context.Context, svcs app.Services, cfg app.Config, args []string, stdout, stderr io.Writer) int {
	leadingID, parseArgs := splitLeadingPositional(args)
	fs := flag.NewFlagSet("monitors get", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	if err := fs.Parse(parseArgs); err != nil {
		writeError(stderr, fail.NewValidation(err.Error(), "usage: ddctl monitors get <id>"), cfg)
		return fail.CodeValidation
	}
	idStr := leadingID
	if idStr == "" && fs.NArg() > 0 {
		idStr = fs.Arg(0)
	}
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil || id <= 0 {
		writeError(stderr, fail.NewValidation("monitor ID must be a number", "usage: ddctl monitors get <id>"), cfg)
		return fail.CodeValidation
	}
	result, err := svcs.Monitors.Get(ctx, id)
	if err != nil {
		writeError(stderr, err, cfg)
		return fail.ExitCode(err)
	}
	if cfg.JSON {
		return writeJSONResult(svcs, cfg, stdout, stderr, result)
	}
	printMonitorSummary(stdout, result)
	return fail.CodeOK
}

func runMonitorsValidateCmd(ctx context.Context, svcs app.Services, cfg app.Config, args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("monitors validate", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	fromFile := fs.String("from-file", "", "path to monitor JSON payload")
	from := fs.String("from", "now-24h", "query preflight start time")
	to := fs.String("to", "now", "query preflight end time")
	if err := fs.Parse(args); err != nil {
		writeError(stderr, fail.NewValidation(err.Error(), "usage: ddctl monitors validate --from-file <path>"), cfg)
		return fail.CodeValidation
	}
	result, err := svcs.Monitors.Validate(ctx, service.MonitorValidateInput{
		FilePath: *fromFile,
		From:     *from,
		To:       *to,
	})
	if err != nil {
		writeError(stderr, err, cfg)
		return fail.ExitCode(err)
	}
	if cfg.JSON {
		return writeJSONResult(svcs, cfg, stdout, stderr, result)
	}
	fmt.Fprintf(stdout, "monitor structure: %s\n", result.Structure)
	fmt.Fprintf(stdout, "query: %s\n", result.Query)
	if result.QueryValid {
		fmt.Fprintln(stdout, "query valid")
	}
	if result.QueryNoData {
		fmt.Fprintln(stdout, "series currently absent")
	}
	if result.RemoteValid {
		fmt.Fprintln(stdout, "remote validate: ok")
	}
	for _, w := range result.Warnings {
		fmt.Fprintf(stdout, "warning: %s\n", w)
	}
	return fail.CodeOK
}

func runMonitorsCreateCmd(ctx context.Context, svcs app.Services, cfg app.Config, args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("monitors create", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	fromFile := fs.String("from-file", "", "path to monitor JSON payload")
	skipValidate := fs.Bool("skip-validate", false, "skip validation before create")
	dryRun := fs.Bool("dry-run", false, "validate and print summary; do not POST")
	muted := fs.Bool("muted", false, "mute monitor after create")
	from := fs.String("from", "now-24h", "query preflight start time")
	to := fs.String("to", "now", "query preflight end time")
	if err := fs.Parse(args); err != nil {
		writeError(stderr, fail.NewValidation(err.Error(), "usage: ddctl monitors create --from-file <path>"), cfg)
		return fail.CodeValidation
	}
	result, err := svcs.Monitors.Create(ctx, service.MonitorMutationInput{
		FilePath:     *fromFile,
		SkipValidate: *skipValidate,
		DryRun:       *dryRun,
		Muted:        *muted,
		From:         *from,
		To:           *to,
	})
	if err != nil {
		writeError(stderr, err, cfg)
		return fail.ExitCode(err)
	}
	if dry, _ := result["dry_run"].(bool); dry && !cfg.JSON {
		fmt.Fprintln(stdout, "dry-run: would create monitor")
		printMonitorSummary(stdout, result)
		return fail.CodeOK
	}
	if cfg.JSON {
		return writeJSONResult(svcs, cfg, stdout, stderr, result)
	}
	printMonitorSummary(stdout, result)
	return fail.CodeOK
}

func runMonitorsUpdateCmd(ctx context.Context, svcs app.Services, cfg app.Config, args []string, stdout, stderr io.Writer) int {
	leadingID, parseArgs := splitLeadingPositional(args)
	fs := flag.NewFlagSet("monitors update", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	fromFile := fs.String("from-file", "", "path to monitor JSON payload")
	replaceAll := fs.Bool("replace-all", false, "confirm full replacement update")
	dryRun := fs.Bool("dry-run", false, "show diff and do not PUT")
	showDiff := fs.Bool("diff", false, "show semantic diff")
	ifUnmodified := fs.String("if-unmodified-since", "", "abort if remote modified timestamp does not match")
	skipValidate := fs.Bool("skip-validate", false, "skip validation before update")
	from := fs.String("from", "now-24h", "query preflight start time")
	to := fs.String("to", "now", "query preflight end time")
	if err := fs.Parse(parseArgs); err != nil {
		writeError(stderr, fail.NewValidation(err.Error(), "usage: ddctl monitors update <id> --from-file <path> --replace-all"), cfg)
		return fail.CodeValidation
	}
	id, err := parseMonitorIDArg(leadingID, fs)
	if err != nil {
		writeError(stderr, err, cfg)
		return fail.ExitCode(err)
	}
	result, err := svcs.Monitors.Update(ctx, service.MonitorMutationInput{
		FilePath:          *fromFile,
		ID:                id,
		SkipValidate:      *skipValidate,
		DryRun:            *dryRun,
		ShowDiff:          *showDiff,
		IfUnmodifiedSince: *ifUnmodified,
		From:              *from,
		To:                *to,
	}, *replaceAll)
	if err != nil {
		writeError(stderr, err, cfg)
		return fail.ExitCode(err)
	}
	if dry, _ := result["dry_run"].(bool); dry && !cfg.JSON {
		fmt.Fprintf(stdout, "dry-run: would update monitor %d\n", id)
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
	if cfg.JSON {
		return writeJSONResult(svcs, cfg, stdout, stderr, result)
	}
	printMonitorSummary(stdout, result)
	return fail.CodeOK
}

func runMonitorsMuteCmd(ctx context.Context, svcs app.Services, cfg app.Config, args []string, stdout, stderr io.Writer) int {
	leadingID, parseArgs := splitLeadingPositional(args)
	fs := flag.NewFlagSet("monitors mute", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	until := fs.String("until", "", "mute until RFC3339 timestamp")
	if err := fs.Parse(parseArgs); err != nil {
		writeError(stderr, fail.NewValidation(err.Error(), "usage: ddctl monitors mute <id> [--until <rfc3339>]"), cfg)
		return fail.CodeValidation
	}
	id, err := parseMonitorIDArg(leadingID, fs)
	if err != nil {
		writeError(stderr, err, cfg)
		return fail.ExitCode(err)
	}
	result, err := svcs.Monitors.Mute(ctx, service.MonitorMuteInput{ID: id, Until: *until})
	if err != nil {
		writeError(stderr, err, cfg)
		return fail.ExitCode(err)
	}
	if cfg.JSON {
		return writeJSONResult(svcs, cfg, stdout, stderr, result)
	}
	printMonitorSummary(stdout, result)
	return fail.CodeOK
}

func runMonitorsUnmuteCmd(ctx context.Context, svcs app.Services, cfg app.Config, args []string, stdout, stderr io.Writer) int {
	leadingID, parseArgs := splitLeadingPositional(args)
	fs := flag.NewFlagSet("monitors unmute", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	confirm := fs.String("confirm", "", "required for production monitors; must equal monitor ID")
	if err := fs.Parse(parseArgs); err != nil {
		writeError(stderr, fail.NewValidation(err.Error(), "usage: ddctl monitors unmute <id> [--confirm <id>]"), cfg)
		return fail.CodeValidation
	}
	id, err := parseMonitorIDArg(leadingID, fs)
	if err != nil {
		writeError(stderr, err, cfg)
		return fail.ExitCode(err)
	}
	result, err := svcs.Monitors.Unmute(ctx, service.MonitorMuteInput{ID: id, Confirm: *confirm})
	if err != nil {
		writeError(stderr, err, cfg)
		return fail.ExitCode(err)
	}
	if cfg.JSON {
		return writeJSONResult(svcs, cfg, stdout, stderr, result)
	}
	printMonitorSummary(stdout, result)
	return fail.CodeOK
}

func parseMonitorIDArg(leading string, fs *flag.FlagSet) (int64, error) {
	idStr := leading
	if idStr == "" && fs.NArg() > 0 {
		idStr = fs.Arg(0)
	}
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil || id <= 0 {
		return 0, fail.NewValidation("monitor ID must be a number", "usage: ddctl monitors <command> <id>")
	}
	return id, nil
}

func printMonitorSummary(w io.Writer, payload map[string]any) {
	id := monitorIDFromMap(payload)
	name, _ := payload["name"].(string)
	mtype, _ := payload["type"].(string)
	state, _ := payload["overall_state"].(string)
	url, _ := payload["url"].(string)
	if url == "" && id > 0 {
		url = fmt.Sprintf("https://app.datadoghq.com/monitors/%d", id)
	}
	fmt.Fprintf(w, "ID:     %d\n", id)
	fmt.Fprintf(w, "Name:   %s\n", name)
	if mtype != "" {
		fmt.Fprintf(w, "Type:   %s\n", mtype)
	}
	if state != "" {
		fmt.Fprintf(w, "State:  %s\n", state)
	}
	if url != "" {
		fmt.Fprintf(w, "URL:    %s\n", url)
	}
	if q, _ := payload["query"].(string); q != "" {
		fmt.Fprintf(w, "Query:  %s\n", q)
	}
}

func monitorIDFromMap(payload map[string]any) int64 {
	switch v := payload["id"].(type) {
	case float64:
		return int64(v)
	case int64:
		return v
	case int:
		return int64(v)
	case string:
		n, _ := strconv.ParseInt(v, 10, 64)
		return n
	default:
		return 0
	}
}

func writeJSONResult(svcs app.Services, cfg app.Config, stdout, stderr io.Writer, result any) int {
	if err := svcs.Output.JSON(stdout, result); err != nil {
		writeError(stderr, fail.NewAPI(err.Error(), "unable to encode JSON result", ""), cfg)
		return fail.CodeAPI
	}
	return fail.CodeOK
}
