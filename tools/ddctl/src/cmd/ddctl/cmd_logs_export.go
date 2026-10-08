package main

import (
	"context"
	"fmt"
	"io"

	"github.com/Alechan/ai-resources/tools/ddctl/src/internal/app"
	"github.com/Alechan/ai-resources/tools/ddctl/src/internal/fail"
	"github.com/Alechan/ai-resources/tools/ddctl/src/internal/service"
)

func runLogsExportCmd(ctx context.Context, svcs app.Services, cfg app.Config, args []string, stdout, stderr io.Writer) int {
	flags, err := parseLogsExportFlags(args)
	if err != nil {
		writeError(stderr, fail.NewValidation(err.Error(), "usage: ddctl logs export [flags] --output <path>"), cfg)
		return fail.CodeValidation
	}
	if flags.limit < 1 || flags.limit > 10000 {
		writeError(stderr, fail.NewValidation("--limit must be between 1 and 10000", "provide a value between 1 and 10000"), cfg)
		return fail.CodeValidation
	}

	exportInput := service.LogsExportInput{
		Query:  flags.query,
		From:   flags.from,
		To:     flags.to,
		Limit:  flags.limit,
		Format: flags.format,
		Fields: service.ParseLogFieldList(flags.fields),
	}

	var result service.LogsQueryResult
	if flags.output == "-" {
		result, err = svcs.LogsQuery.RunAll(ctx, service.LogsQueryInput{
			Query: flags.query,
			From:  flags.from,
			To:    flags.to,
		}, flags.limit)
		if err != nil {
			writeError(stderr, err, cfg)
			return fail.ExitCode(err)
		}
		if err := writeLogsExportOutput(result.Data, exportInput, stdout); err != nil {
			writeError(stderr, fail.AsError(err), cfg)
			return fail.ExitCode(err)
		}
	} else {
		result, err = svcs.LogsQuery.ExportLogsToFile(ctx, exportInput, flags.output)
		if err != nil {
			writeError(stderr, err, cfg)
			return fail.ExitCode(err)
		}
	}

	destination := flags.output
	if destination == "-" {
		destination = "stdout"
	}
	fmt.Fprintf(stderr, "exported %d events (hit_count=%d) to %s\n", result.ReturnedCount, result.HitCount, destination)
	return fail.CodeOK
}

func writeLogsExportOutput(events []service.LogEvent, input service.LogsExportInput, w io.Writer) error {
	switch input.Format {
	case "", "ndjson":
		return service.WriteLogsNDJSON(events, input.Fields, w)
	case "csv":
		return service.WriteLogsCSV(events, input.Fields, w)
	default:
		return fail.NewValidation("unsupported export format", fmt.Sprintf("use ndjson or csv, got %q", input.Format))
	}
}
