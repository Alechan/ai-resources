package main

import (
	"context"
	"io"

	"github.com/Alechan/ai-resources/tools/ddctl/src/internal/app"
	"github.com/Alechan/ai-resources/tools/ddctl/src/internal/fail"
)

func runEventsCmd(ctx context.Context, svcs app.Services, cfg app.Config, args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		writeError(stderr, fail.NewValidation("missing events subcommand", "usage: ddctl events list [flags]"), cfg)
		return fail.CodeValidation
	}
	switch args[0] {
	case "list":
		return runEventsListCmd(ctx, svcs, cfg, args[1:], stdout, stderr)
	default:
		writeError(stderr, fail.NewValidation("unknown events subcommand", "usage: ddctl events list [flags]"), cfg)
		return fail.CodeValidation
	}
}
