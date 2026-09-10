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

func runLogsGetCmd(ctx context.Context, svcs app.Services, cfg app.Config, args []string, stdout, stderr io.Writer) int {
	leadingID, parseArgs := splitLeadingPositional(args)

	fs := flag.NewFlagSet("logs get", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	from := fs.String("from", "now-1h", "start time (relative or ISO-8601)")
	to := fs.String("to", "now", "end time (relative or ISO-8601)")
	fields := fs.String("fields", "", "comma-separated JSON field projection")
	if err := fs.Parse(parseArgs); err != nil {
		writeError(stderr, fail.NewValidation(err.Error(), "usage: ddctl logs get <event_id> [--from <time>] [--to <time>]"), cfg)
		return fail.CodeValidation
	}
	eventID := leadingID
	if eventID == "" && fs.NArg() > 0 {
		eventID = fs.Arg(0)
	}
	if eventID == "" {
		writeError(stderr, fail.NewValidation("missing log event ID", "usage: ddctl logs get <event_id> [--from <time>] [--to <time>]"), cfg)
		return fail.CodeValidation
	}
	if *fields != "" && !cfg.JSON {
		writeError(stderr, fail.NewValidation("--fields requires --json", "pass --json for field projection"), cfg)
		return fail.CodeValidation
	}

	event, err := svcs.LogsQuery.Get(ctx, service.LogsGetInput{
		ID:   eventID,
		From: *from,
		To:   *to,
	})
	if err != nil {
		writeError(stderr, err, cfg)
		return fail.ExitCode(err)
	}

	if cfg.JSON {
		payload := service.ProjectLogEventFields(event, service.ParseLogFieldList(*fields))
		if err := svcs.Output.JSON(stdout, payload); err != nil {
			writeError(stderr, fail.NewAPI(err.Error(), "unable to encode log event", ""), cfg)
			return fail.CodeAPI
		}
		return fail.CodeOK
	}

	fmt.Fprintf(stdout, "%s [%s] %s: %s\n", event.Timestamp, event.Status, event.Service, event.Message)
	for _, line := range service.VerboseCustomLines(event, nil) {
		fmt.Fprintf(stdout, "  %s\n", line)
	}
	return fail.CodeOK
}
