package main

import (
	"context"
	"flag"
	"fmt"
	"io"

	"github.com/Alechan/ai-resources/tools/ddctl/src/internal/app"
	"github.com/Alechan/ai-resources/tools/ddctl/src/internal/fail"
	"github.com/Alechan/ai-resources/tools/ddctl/src/internal/service"
)

func runLogsQueryCmd(ctx context.Context, svcs app.Services, cfg app.Config, args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("logs query", flag.ContinueOnError)
	flags, err := parseLogsQueryFlags(fs, args)
	if err != nil {
		writeError(stderr, fail.NewValidation(err.Error(), "usage: ddctl logs query [flags]"), cfg)
		return fail.CodeValidation
	}

	if flags.limit < 1 || flags.limit > 1000 {
		writeError(stderr, fail.NewValidation("--limit must be between 1 and 1000", "provide a value between 1 and 1000"), cfg)
		return fail.CodeValidation
	}
	if flags.fields != "" && flags.countOnly {
		writeError(stderr, fail.NewValidation("--fields cannot be used with --count-only", "remove --fields from count-only queries"), cfg)
		return fail.CodeValidation
	}
	if flags.fields != "" && !cfg.JSON {
		writeError(stderr, fail.NewValidation("--fields requires --json", "pass --json or use text mode with --verbose"), cfg)
		return fail.CodeValidation
	}

	fieldList := service.ParseLogFieldList(flags.fields)
	verboseKeys := service.ParseLogFieldList(flags.verboseKeys)

	if flags.countOnly {
		if flags.cursor != "" {
			writeError(stderr, fail.NewValidation("--cursor cannot be used with --count-only", "remove --cursor from count-only queries"), cfg)
			return fail.CodeValidation
		}
		return runLogsQueryCountOnly(ctx, svcs, cfg, stdout, stderr, flags.query, flags.from, flags.to)
	}

	if flags.all {
		return runLogsQueryAll(ctx, svcs, cfg, stdout, stderr, flags, fieldList, verboseKeys)
	}

	result, err := svcs.LogsQuery.Run(ctx, service.LogsQueryInput{
		Query:  flags.query,
		From:   flags.from,
		To:     flags.to,
		Limit:  flags.limit,
		Cursor: flags.cursor,
	})
	if err != nil {
		writeError(stderr, err, cfg)
		return fail.ExitCode(err)
	}
	return writeLogsQueryResult(svcs, cfg, stdout, stderr, result, fieldList, flags.verbose, verboseKeys)
}

func runLogsQueryCountOnly(ctx context.Context, svcs app.Services, cfg app.Config, stdout, stderr io.Writer, query, from, to string) int {
	result, err := svcs.LogsQuery.Run(ctx, service.LogsQueryInput{
		Query:     query,
		From:      from,
		To:        to,
		Limit:     1,
		CountOnly: true,
	})
	if err != nil {
		writeError(stderr, err, cfg)
		return fail.ExitCode(err)
	}
	out := struct {
		QueryWindow struct {
			From string `json:"from"`
			To   string `json:"to"`
		} `json:"query_window"`
		HitCount  int      `json:"hit_count"`
		Truncated bool     `json:"truncated"`
		Warnings  []string `json:"warnings,omitempty"`
	}{
		HitCount:  result.HitCount,
		Truncated: false,
		Warnings:  result.Warnings,
	}
	out.QueryWindow.From = from
	out.QueryWindow.To = to

	if cfg.JSON {
		if err := svcs.Output.JSON(stdout, out); err != nil {
			writeError(stderr, fail.NewAPI(err.Error(), "unable to encode logs result", ""), cfg)
			return fail.CodeAPI
		}
		return fail.CodeOK
	}

	fmt.Fprintf(stdout, "query window: from=%s to=%s\n", from, to)
	fmt.Fprintf(stdout, "hit_count: %d\n", result.HitCount)
	for _, warning := range result.Warnings {
		fmt.Fprintf(stdout, "warning: %s\n", warning)
	}
	return fail.CodeOK
}

func runLogsQueryAll(ctx context.Context, svcs app.Services, cfg app.Config, stdout, stderr io.Writer, flags logsQueryFlags, fieldList, verboseKeys []string) int {
	result, err := svcs.LogsQuery.RunAll(ctx, service.LogsQueryInput{
		Query: flags.query,
		From:  flags.from,
		To:    flags.to,
	}, flags.limit)
	if err != nil {
		writeError(stderr, err, cfg)
		return fail.ExitCode(err)
	}
	return writeLogsQueryResult(svcs, cfg, stdout, stderr, result, fieldList, flags.verbose, verboseKeys)
}

func writeLogsQueryResult(svcs app.Services, cfg app.Config, stdout, stderr io.Writer, result service.LogsQueryResult, fieldList []string, verbose bool, verboseKeys []string) int {
	if cfg.JSON {
		payload := service.ProjectLogsQueryResult(result, fieldList)
		if err := svcs.Output.JSON(stdout, payload); err != nil {
			writeError(stderr, fail.NewAPI(err.Error(), "unable to encode logs result", ""), cfg)
			return fail.CodeAPI
		}
		return fail.CodeOK
	}

	printLogEvents(stdout, result.Data, verbose, verboseKeys)
	fmt.Fprintf(stdout, "# hit_count: %d\n", result.HitCount)
	for _, warning := range result.Warnings {
		fmt.Fprintf(stdout, "warning: %s\n", warning)
	}
	if result.Truncated {
		if result.HitCount > 0 {
			fmt.Fprintf(stdout, "returned %d of at least %d (limit %d reached)\n", result.ReturnedCount, result.HitCount, result.Limit)
		} else {
			fmt.Fprintf(stdout, "returned %d results (limit %d reached)\n", result.ReturnedCount, result.Limit)
		}
	}
	if result.NextCursor != "" {
		fmt.Fprintf(stdout, "# next_cursor: %s\n", result.NextCursor)
		fmt.Fprintf(stdout, "# use: ddctl logs query --cursor '%s' to fetch the next page\n", result.NextCursor)
	}
	return fail.CodeOK
}

func printLogEvents(w io.Writer, events []service.LogEvent, verbose bool, verboseKeys []string) {
	for _, event := range events {
		fmt.Fprintf(w, "%s [%s] %s: %s\n", event.Timestamp, event.Status, event.Service, event.Message)
		if verbose {
			for _, line := range service.VerboseCustomLines(event, verboseKeys) {
				fmt.Fprintf(w, "  %s\n", line)
			}
		}
	}
}
