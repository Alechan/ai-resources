package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"strings"

	"github.com/Alechan/ai-resources/tools/ddctl/src/internal/app"
	"github.com/Alechan/ai-resources/tools/ddctl/src/internal/fail"
)

func runMonitorsListCmd(ctx context.Context, svcs app.Services, cfg app.Config, args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("monitors-list", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	tagFilter := fs.String("tag", "", "filter monitors by tag (e.g. env:prod)")

	if err := fs.Parse(args); err != nil {
		writeError(stderr, fail.NewValidation(err.Error(), "usage: ddctl monitors-list [--tag <tag>]"))
		return fail.CodeValidation
	}

	result, err := svcs.MonitorsList.Run(ctx)
	if err != nil {
		writeError(stderr, err)
		return fail.ExitCode(err)
	}

	// Apply tag filter
	monitors := result.Monitors
	if *tagFilter != "" {
		filtered := monitors[:0]
		for _, m := range monitors {
			for _, t := range m.Tags {
				if t == *tagFilter {
					filtered = append(filtered, m)
					break
				}
			}
		}
		monitors = filtered
		result.Monitors = monitors
	}

	if cfg.JSON {
		if err := svcs.Output.JSON(stdout, result); err != nil {
			writeError(stderr, fail.NewAPI(err.Error(), "unable to encode monitors result", ""))
			return fail.CodeAPI
		}
		return fail.CodeOK
	}

	for _, m := range monitors {
		tags := strings.Join(m.Tags, ",")
		fmt.Fprintf(stdout, "[%d] %-10s %-8s %s  tags:%s\n",
			m.ID, m.OverallState, m.Type, m.Name, tags)
	}
	return fail.CodeOK
}
