package main

import (
	"context"
	"flag"
	"io"

	"github.com/Alechan/ai-resources/tools/gdrivectl/src/internal/app"
	"github.com/Alechan/ai-resources/tools/gdrivectl/src/internal/fail"
	"github.com/Alechan/ai-resources/tools/gdrivectl/src/internal/service"
)

func runFileMetaCmd(ctx context.Context, svcs app.Services, cfg app.Config, args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("file-meta", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	var req service.FileMetaRequest
	fs.StringVar(&req.ID, "id", "", "File ID")
	fs.StringVar(&req.Fields, "fields", "id,name,mimeType,webViewLink", "Fields projection")
	if err := fs.Parse(args); err != nil {
		err = fail.NewValidation(err.Error(), "usage: gdrivectl file-meta --id <file_id>")
		writeError(stderr, err)
		return fail.ExitCode(err)
	}
	if req.ID == "" {
		err := fail.NewValidation("missing --id", "provide a Google Drive file id")
		writeError(stderr, err)
		return fail.ExitCode(err)
	}
	resp, err := svcs.FileMeta.Run(ctx, req)
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
