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
	fs := flag.NewFlagSet("events list", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	from := fs.String("from", "now-1h", "start time (relative or ISO-8601)")
	to := fs.String("to", "now", "end time (relative or ISO-8601)")
	sources := fs.String("sources", "", "filter by comma-separated event sources")
	tags := fs.String("tags", "", "filter by comma-separated tags (e.g. env:prod,service:api)")
	limit := fs.Int("limit", 50, "max events to return (or total with --all)")
	all := fs.Bool("all", false, "auto-paginate until no more results or --limit is reached")
	cursor := fs.String("cursor", "", "pagination cursor from a previous result")
	countOnly := fs.Bool("count-only", false, "return only the hit count, no event data")

	if err := fs.Parse(args); err != nil {
		writeError(stderr, fail.NewValidation(err.Error(), "usage: ddctl events list [flags]"), cfg)
		return fail.CodeValidation
	}

	if *countOnly {
		if *all {
			writeError(stderr, fail.NewValidation("--all cannot be used with --count-only", "remove --all from count-only queries"), cfg)
			return fail.CodeValidation
		}
		if *cursor != "" {
			writeError(stderr, fail.NewValidation("--cursor cannot be used with --count-only", "remove --cursor from count-only queries"), cfg)
			return fail.CodeValidation
		}
	}

	if *all {
		return runEventsListAll(ctx, svcs, cfg, stdout, stderr, *from, *to, *sources, *tags, *limit)
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
		writeError(stderr, err, cfg)
		return fail.ExitCode(err)
	}

	if cfg.JSON {
		if err := svcs.Output.JSON(stdout, result); err != nil {
			writeError(stderr, fail.NewAPI(err.Error(), "unable to encode events result", ""), cfg)
			return fail.CodeAPI
		}
		return fail.CodeOK
	}

	if *countOnly {
		fmt.Fprintf(stdout, "hit_count: %d\n", result.HitCount)
		return fail.CodeOK
	}

	printEvents(stdout, result.Events)
	fmt.Fprintf(stdout, "# hit_count: %d\n", result.HitCount)
	for _, warning := range result.Warnings {
		fmt.Fprintf(stdout, "warning: %s\n", warning)
	}
	if result.NextCursor != "" {
		fmt.Fprintf(stdout, "# next_cursor: %s\n", result.NextCursor)
		fmt.Fprintf(stdout, "# use: ddctl events list --cursor '%s' to fetch the next page\n", result.NextCursor)
	}
	return fail.CodeOK
}

func runEventsListAll(ctx context.Context, svcs app.Services, cfg app.Config, stdout, stderr io.Writer, from, to, sources, tags string, maxTotal int) int {
	const pageSize = 50
	var allEvents []service.Event
	warningsSet := make(map[string]struct{})
	hitCount := 0
	truncated := false
	cursor := ""

	for {
		remaining := maxTotal - len(allEvents)
		if remaining <= 0 {
			truncated = true
			break
		}
		batchLimit := pageSize
		if remaining < batchLimit {
			batchLimit = remaining
		}

		result, err := svcs.EventsList.Run(ctx, service.EventsListInput{
			From:    from,
			To:      to,
			Sources: sources,
			Tags:    tags,
			Limit:   batchLimit,
			Cursor:  cursor,
		})
		if err != nil {
			writeError(stderr, err, cfg)
			return fail.ExitCode(err)
		}
		if result.HitCount > hitCount {
			hitCount = result.HitCount
		}
		for _, warning := range result.Warnings {
			warningsSet[warning] = struct{}{}
		}
		allEvents = append(allEvents, result.Events...)
		if result.NextCursor == "" || len(result.Events) == 0 {
			if len(allEvents) >= maxTotal && hitCount > len(allEvents) {
				truncated = true
			}
			break
		}
		cursor = result.NextCursor
	}

	warnings := make([]string, 0, len(warningsSet))
	for w := range warningsSet {
		warnings = append(warnings, w)
	}
	combined := service.EventsListResult{
		Events:        allEvents,
		HitCount:      hitCount,
		ReturnedCount: len(allEvents),
		Warnings:      warnings,
	}
	if truncated {
		combined.Truncated = true
		combined.Limit = maxTotal
	}

	if cfg.JSON {
		if err := svcs.Output.JSON(stdout, combined); err != nil {
			writeError(stderr, fail.NewAPI(err.Error(), "unable to encode events result", ""), cfg)
			return fail.CodeAPI
		}
		return fail.CodeOK
	}

	printEvents(stdout, allEvents)
	fmt.Fprintf(stdout, "# hit_count: %d\n", hitCount)
	for _, warning := range warnings {
		fmt.Fprintf(stdout, "warning: %s\n", warning)
	}
	if truncated {
		if hitCount > 0 {
			fmt.Fprintf(stdout, "returned %d of at least %d (limit %d reached)\n", len(allEvents), hitCount, maxTotal)
		} else {
			fmt.Fprintf(stdout, "returned %d events (limit %d reached)\n", len(allEvents), maxTotal)
		}
	}
	return fail.CodeOK
}

func printEvents(w io.Writer, events []service.Event) {
	for _, ev := range events {
		ts := time.UnixMilli(ev.Timestamp).UTC().Format(time.RFC3339)
		alert := ev.Status
		if alert == "" {
			alert = ev.AlertType
		}
		title := ev.Title
		if title == "" {
			title = ev.Text
		}
		tagLine := strings.Join(ev.Tags, ",")
		fmt.Fprintf(w, "%s [%s] %s  %s\n", ts, alert, title, tagLine)
	}
}
