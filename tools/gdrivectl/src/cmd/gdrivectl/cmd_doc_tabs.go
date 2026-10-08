package main

import (
	"context"
	"flag"
	"io"

	"github.com/Alechan/ai-resources/tools/gdrivectl/src/internal/app"
	"github.com/Alechan/ai-resources/tools/gdrivectl/src/internal/fail"
	"github.com/Alechan/ai-resources/tools/gdrivectl/src/internal/service"
)

func runDocTabsCmd(ctx context.Context, svcs app.Services, cfg app.Config, args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("doc-tabs", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	var req service.DocTabsRequest
	fs.StringVar(&req.ID, "id", "", "Document ID")
	if err := fs.Parse(args); err != nil {
		err = fail.NewValidation(err.Error(), "usage: gdrivectl doc-tabs --id <doc_id>")
		writeError(stderr, err)
		return fail.ExitCode(err)
	}
	if req.ID == "" {
		err := fail.NewValidation("missing --id", "provide a Google Docs document id")
		writeError(stderr, err)
		return fail.ExitCode(err)
	}
	resp, err := svcs.DocTabs.Run(ctx, req)
	if err != nil {
		writeError(stderr, err)
		return fail.ExitCode(err)
	}
	if err := svcs.Output.JSON(stdout, resp); err != nil {
		writeError(stderr, fail.NewAPI(err.Error(), "unable to encode response", ""))
		return fail.CodeAPI
	}
	return fail.CodeOK
}
