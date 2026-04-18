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
	fs := flag.NewFlagSet("logs-query", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	query := fs.String("query", "*", "search query string")
	fs.String("q", "*", "search query string (shorthand)")
	from := fs.String("from", "now-1h", "start time (relative or ISO-8601)")
	to := fs.String("to", "now", "end time (relative or ISO-8601)")
	limit := fs.Int("limit", 50, "max results (1-1000)")

	if err := fs.Parse(args); err != nil {
		err = fail.NewValidation(err.Error(), "usage: ddctl logs-query [flags]")
		writeError(stderr, err)
		return fail.ExitCode(err)
	}

	// -q shorthand takes precedence if provided and differs from default
	if q := fs.Lookup("q"); q != nil && q.Value.String() != "*" {
		query = &[]string{q.Value.String()}[0]
	}

	if *limit < 1 || *limit > 1000 {
		err := fail.NewValidation("--limit must be between 1 and 1000", "provide a value between 1 and 1000")
		writeError(stderr, err)
		return fail.ExitCode(err)
	}

	input := service.LogsQueryInput{
		Query: *query,
		From:  *from,
		To:    *to,
		Limit: *limit,
	}

	result, err := svcs.LogsQuery.Run(ctx, input)
	if err != nil {
		writeError(stderr, err)
		return fail.ExitCode(err)
	}

	if cfg.JSON {
		if err := svcs.Output.JSON(stdout, result); err != nil {
			writeError(stderr, fail.NewAPI(err.Error(), "unable to encode logs result", ""))
			return fail.CodeAPI
		}
		return fail.CodeOK
	}

	for _, event := range result.Data {
		fmt.Fprintf(stdout, "%s [%s] %s: %s\n",
			event.Attributes.Timestamp,
			event.Attributes.Status,
			event.Attributes.Service,
			event.Attributes.Message,
		)
	}
	return fail.CodeOK
}
