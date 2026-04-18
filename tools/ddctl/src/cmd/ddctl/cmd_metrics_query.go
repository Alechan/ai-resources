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

func runMetricsQueryCmd(ctx context.Context, svcs app.Services, cfg app.Config, args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("metrics-query", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	query := fs.String("query", "", "DataDog metrics query (e.g. avg:system.cpu.user{service:my-svc})")
	fs.String("q", "", "metrics query shorthand")
	from := fs.String("from", "now-1h", "start time (relative or ISO-8601)")
	to := fs.String("to", "now", "end time (relative or ISO-8601)")
	raw := fs.Bool("raw", false, "include full pointlist in JSON output (default: summary stats only)")

	if err := fs.Parse(args); err != nil {
		writeError(stderr, fail.NewValidation(err.Error(), "usage: ddctl metrics-query --query <query> [flags]"))
		return fail.CodeValidation
	}

	// -q shorthand
	if q := fs.Lookup("q"); q != nil && q.Value.String() != "" {
		query = &[]string{q.Value.String()}[0]
	}

	if *query == "" {
		writeError(stderr, fail.NewValidation("--query is required", `example: ddctl metrics-query --query "avg:system.cpu.user{service:tapir}"`))
		return fail.CodeValidation
	}

	input := service.MetricsQueryInput{
		Query: *query,
		From:  *from,
		To:    *to,
	}

	result, err := svcs.MetricsQuery.Run(ctx, input)
	if err != nil {
		writeError(stderr, err)
		return fail.ExitCode(err)
	}

	if cfg.JSON {
		out := result
		if !*raw {
			// Strip pointlist from JSON output for brevity
			stripped := make([]service.MetricsSeries, len(result.Series))
			for i, s := range result.Series {
				stripped[i] = s
				stripped[i].Pointlist = nil
			}
			out.Series = stripped
		}
		if err := svcs.Output.JSON(stdout, out); err != nil {
			writeError(stderr, fail.NewAPI(err.Error(), "unable to encode metrics result", ""))
			return fail.CodeAPI
		}
		return fail.CodeOK
	}

	if len(result.Series) == 0 {
		fmt.Fprintln(stdout, "no data")
		return fail.CodeOK
	}

	for _, s := range result.Series {
		fmt.Fprintf(stdout, "%s {%s}\n", s.Metric, s.Scope)
		fmt.Fprintf(stdout, "  points: %d  interval: %ds\n", len(s.Pointlist), s.Interval)
		fmt.Fprintf(stdout, "  min=%.4g  avg=%.4g  max=%.4g  last=%.4g\n",
			s.Min, s.Avg, s.Max, s.Last)
	}
	return fail.CodeOK
}
