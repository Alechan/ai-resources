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

	if err := fs.Parse(args); err != nil {
		writeError(stderr, fail.NewValidation(err.Error(), "usage: ddctl events-list [flags]"))
		return fail.CodeValidation
	}

	input := service.EventsListInput{
		From:    *from,
		To:      *to,
		Sources: *sources,
		Tags:    *tags,
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

	for _, ev := range result.Events {
		ts := time.Unix(ev.DateHappened, 0).UTC().Format(time.RFC3339)
		title := ev.Title
		alert := ev.AlertType
		if alert == "" {
			alert = ev.Priority
		}
		tags := strings.Join(ev.Tags, ",")
		fmt.Fprintf(stdout, "%s [%s] %s  %s\n", ts, alert, title, tags)
	}
	return fail.CodeOK
}
