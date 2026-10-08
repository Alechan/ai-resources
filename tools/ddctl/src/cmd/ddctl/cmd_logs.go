package main

import (
	"context"
	"io"

	"github.com/Alechan/ai-resources/tools/ddctl/src/internal/app"
	"github.com/Alechan/ai-resources/tools/ddctl/src/internal/fail"
)

func runLogsCmd(ctx context.Context, svcs app.Services, cfg app.Config, args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		writeError(stderr, fail.NewValidation("missing logs subcommand", "usage: ddctl logs <query|export|get> [flags]"), cfg)
		return fail.CodeValidation
	}
	switch args[0] {
	case "query":
		return runLogsQueryCmd(ctx, svcs, cfg, args[1:], stdout, stderr)
	case "export":
		return runLogsExportCmd(ctx, svcs, cfg, args[1:], stdout, stderr)
	case "get":
		return runLogsGetCmd(ctx, svcs, cfg, args[1:], stdout, stderr)
	default:
		writeError(stderr, fail.NewValidation("unknown logs subcommand", "usage: ddctl logs <query|export|get> [flags]"), cfg)
		return fail.CodeValidation
	}
}
