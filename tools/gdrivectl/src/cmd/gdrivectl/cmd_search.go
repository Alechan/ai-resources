package main

import (
	"context"
	"flag"
	"io"

	"github.com/Alechan/ai-resources/tools/gdrivectl/src/internal/app"
	"github.com/Alechan/ai-resources/tools/gdrivectl/src/internal/fail"
	"github.com/Alechan/ai-resources/tools/gdrivectl/src/internal/service"
)

func runSearchCmd(ctx context.Context, svcs app.Services, cfg app.Config, args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("search", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	var req service.SearchRequest
	fs.StringVar(&req.Query, "query", "", "Drive query")
	fs.StringVar(&req.Corpora, "corpora", "allDrives", "allDrives|user|drive")
	fs.StringVar(&req.DriveID, "drive-id", "", "Drive ID when corpora=drive")
	fs.IntVar(&req.PageSize, "page-size", 100, "Page size")
	fs.StringVar(&req.Fields, "fields", "files(id,name,mimeType,webViewLink),nextPageToken", "Fields projection")
	if err := fs.Parse(args); err != nil {
		err = fail.NewValidation(err.Error(), "usage: gdrivectl search --query <q>")
		writeError(stderr, err)
		return fail.ExitCode(err)
	}
	if req.Query == "" {
		err := fail.NewValidation("missing --query", "provide a Drive query expression")
		writeError(stderr, err)
		return fail.ExitCode(err)
	}
	resp, err := svcs.Search.Run(ctx, req)
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
