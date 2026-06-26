package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/Alechan/ai-resources/tools/ddctl/src/internal/app"
	"github.com/Alechan/ai-resources/tools/ddctl/src/internal/fail"
	"github.com/Alechan/ai-resources/tools/ddctl/src/internal/service"
)

func runEventsListCmd(ctx context.Context, svcs app.Services, cfg app.Config, args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("events-list", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	from := fs.String("from", "now-1h", "start time (relative or ISO-8601)")
	to := fs.String("to", "now", "end time (relative or ISO-8601)")
	sources := fs.String("sources", "", "filter by comma-separated event sources")
	tags := fs.String("tags", "", "filter by comma-separated tags (e.g. env:prod,service:api)")
	limit := fs.Int("limit", 50, "max events to return")
	cursor := fs.String("cursor", "", "pagination cursor from a previous result")
	countOnly := fs.Bool("count-only", false, "return only the hit count, no event data")

	if err := fs.Parse(args); err != nil {
		writeError(stderr, fail.NewValidation(err.Error(), "usage: ddctl events-list [flags]"))
		return fail.CodeValidation
	}

	input := service.EventsListInput{
		From:      *from,
		To:        *to,
		Sources:   *sources,
		Tags:      *tags,
		Limit:     *limit,
		Cursor:    *cursor,
		CountOnly: *countOnly,
	}

	result, err := svcs.EventsList.Run(ctx, input)
	if err != nil {
		writeError(stderr, err)
		return fail.ExitCode(err)
	}

	if cfg.JSON {
		if err := svcs.Output.JSON(stdout, result); err != nil {
			writeError(stderr, fail.NewAPI(err.Error(), "unable to encode events result", ""))
			return fail.CodeAPI
		}
		return fail.CodeOK
	}

	if *countOnly {
		fmt.Fprintf(stdout, "hit_count: %d\n", result.HitCount)
		return fail.CodeOK
	}

	for _, ev := range result.Events {
		ts := time.UnixMilli(ev.Timestamp).UTC().Format(time.RFC3339)
		alert := ev.Status
		if alert == "" {
			alert = ev.AlertType
		}
		title := ev.Title
		if title == "" {
			title = ev.Text
		}
		tags := strings.Join(ev.Tags, ",")
		fmt.Fprintf(stdout, "%s [%s] %s  %s\n", ts, alert, title, tags)
	}

	if result.NextCursor != "" {
		fmt.Fprintf(stdout, "\nnext_cursor: %s\n", result.NextCursor)
	}

	return fail.CodeOK
}
