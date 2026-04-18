package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/Alechan/ai-resources/tools/ddctl/src/internal/app"
	"github.com/Alechan/ai-resources/tools/ddctl/src/internal/fail"
)

func runMonitorsGetCmd(ctx context.Context, svcs app.Services, cfg app.Config, args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("monitors-get", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	if err := fs.Parse(args); err != nil {
		writeError(stderr, fail.NewValidation(err.Error(), "usage: ddctl monitors-get <id>"))
		return fail.CodeValidation
	}

	if fs.NArg() < 1 {
		writeError(stderr, fail.NewValidation("missing monitor ID", "usage: ddctl monitors-get <id>"))
		return fail.CodeValidation
	}

	id, err := strconv.ParseInt(fs.Arg(0), 10, 64)
	if err != nil {
		writeError(stderr, fail.NewValidation("monitor ID must be a number", "usage: ddctl monitors-get <id>"))
		return fail.CodeValidation
	}

	result, svcErr := svcs.MonitorsGet.Run(ctx, id)
	if svcErr != nil {
		writeError(stderr, svcErr)
		return fail.ExitCode(svcErr)
	}

	if cfg.JSON {
		if err := svcs.Output.JSON(stdout, result); err != nil {
			writeError(stderr, fail.NewAPI(err.Error(), "unable to encode monitor result", ""))
			return fail.CodeAPI
		}
		return fail.CodeOK
	}

	m := result.Monitor
	fmt.Fprintf(stdout, "ID:     %d\n", m.ID)
	fmt.Fprintf(stdout, "Name:   %s\n", m.Name)
	fmt.Fprintf(stdout, "Type:   %s\n", m.Type)
	fmt.Fprintf(stdout, "State:  %s\n", m.OverallState)
	fmt.Fprintf(stdout, "URL:    %s\n", m.URL)
	fmt.Fprintf(stdout, "Tags:   %s\n", strings.Join(m.Tags, ", "))
	fmt.Fprintf(stdout, "Query:  %s\n", m.Query)
	if m.Message != "" {
		fmt.Fprintf(stdout, "Msg:    %s\n", m.Message)
	}
	return fail.CodeOK
}
