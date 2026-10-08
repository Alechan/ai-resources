package main

import (
	"context"
	"io"

	"github.com/Alechan/ai-resources/tools/ddctl/src/internal/app"
	"github.com/Alechan/ai-resources/tools/ddctl/src/internal/fail"
)

func runMetricsCmd(ctx context.Context, svcs app.Services, cfg app.Config, args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		writeError(stderr, fail.NewValidation("missing metrics subcommand", "usage: ddctl metrics query --query <query> [flags]"), cfg)
		return fail.CodeValidation
	}
	switch args[0] {
	case "query":
		return runMetricsQueryCmd(ctx, svcs, cfg, args[1:], stdout, stderr)
	default:
		writeError(stderr, fail.NewValidation("unknown metrics subcommand", "usage: ddctl metrics query --query <query> [flags]"), cfg)
		return fail.CodeValidation
	}
}
