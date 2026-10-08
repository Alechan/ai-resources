package main

import (
	"context"
	"flag"
	"io"

	"github.com/Alechan/ai-resources/tools/gdrivectl/src/internal/app"
	"github.com/Alechan/ai-resources/tools/gdrivectl/src/internal/fail"
	"github.com/Alechan/ai-resources/tools/gdrivectl/src/internal/service"
)

func runUploadCmd(ctx context.Context, svcs app.Services, cfg app.Config, args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("upload", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	var req service.UploadRequest
	fs.StringVar(&req.Path, "path", "", "Local file path to upload")
	fs.StringVar(&req.Name, "name", "", "Drive file name (default: base name from --path)")
	fs.StringVar(&req.ParentID, "parent-id", "", "Destination Drive folder ID")
	fs.StringVar(&req.MIME, "mime", "", "MIME type (default: auto-detected)")
	fs.StringVar(&req.Fields, "fields", "id,name,mimeType,webViewLink,parents", "Fields projection")

	if err := fs.Parse(args); err != nil {
		err = fail.NewValidation(err.Error(), "usage: gdrivectl upload --path <local_file>")
		writeError(stderr, err)
		return fail.ExitCode(err)
	}
	if req.Path == "" {
		err := fail.NewValidation("missing --path", "provide a readable local file path")
		writeError(stderr, err)
		return fail.ExitCode(err)
	}

	resp, err := svcs.Upload.Run(ctx, req)
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
