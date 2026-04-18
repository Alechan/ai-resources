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
	limit := fs.Int("limit", 50, "max results per page (1-1000); when --all is set, max total results")
	all := fs.Bool("all", false, "auto-paginate until no more results or --limit is reached")
	cursor := fs.String("cursor", "", "pagination cursor from a previous result's next_cursor field")

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

	if *all {
		return runLogsQueryAll(ctx, svcs, cfg, stdout, stderr, *query, *from, *to, *limit)
	}

	input := service.LogsQueryInput{
		Query:  *query,
		From:   *from,
		To:     *to,
		Limit:  *limit,
		Cursor: *cursor,
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

	printLogEvents(stdout, result.Data)
	if result.NextCursor != "" {
		fmt.Fprintf(stdout, "# next_cursor: %s\n", result.NextCursor)
		fmt.Fprintf(stdout, "# use: ddctl logs-query --cursor '%s' to fetch the next page\n", result.NextCursor)
	}
	return fail.CodeOK
}

// runLogsQueryAll auto-paginates until no cursor remains or the total limit is reached.
func runLogsQueryAll(ctx context.Context, svcs app.Services, cfg app.Config, stdout, stderr io.Writer, query, from, to string, maxTotal int) int {
	const pageSize = 50 // fetch at most 50 per page; DataDog's v1 limit
	var allEvents []service.LogEvent
	cursor := ""

	for {
		remaining := maxTotal - len(allEvents)
		if remaining <= 0 {
			break
		}
		batchLimit := pageSize
		if remaining < batchLimit {
			batchLimit = remaining
		}

		input := service.LogsQueryInput{
			Query:  query,
			From:   from,
			To:     to,
			Limit:  batchLimit,
			Cursor: cursor,
		}
		result, err := svcs.LogsQuery.Run(ctx, input)
		if err != nil {
			writeError(stderr, err)
			return fail.ExitCode(err)
		}
		allEvents = append(allEvents, result.Data...)
		if result.NextCursor == "" || len(result.Data) == 0 {
			break
		}
		cursor = result.NextCursor
	}

	combined := service.LogsQueryResult{Data: allEvents}
	if cfg.JSON {
		if err := svcs.Output.JSON(stdout, combined); err != nil {
			writeError(stderr, fail.NewAPI(err.Error(), "unable to encode logs result", ""))
			return fail.CodeAPI
		}
		return fail.CodeOK
	}
	printLogEvents(stdout, allEvents)
	return fail.CodeOK
}

func printLogEvents(w io.Writer, events []service.LogEvent) {
	for _, event := range events {
		fmt.Fprintf(w, "%s [%s] %s: %s\n",
			event.Attributes.Timestamp,
			event.Attributes.Status,
			event.Attributes.Service,
			event.Attributes.Message,
		)
	}
}
