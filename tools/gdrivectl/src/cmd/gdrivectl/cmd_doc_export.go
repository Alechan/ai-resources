package main

import (
	"context"
	"flag"
	"io"
	"os"

	"github.com/Alechan/ai-resources/tools/gdrivectl/src/internal/app"
	"github.com/Alechan/ai-resources/tools/gdrivectl/src/internal/fail"
	"github.com/Alechan/ai-resources/tools/gdrivectl/src/internal/service"
)

func runDocExportCmd(ctx context.Context, svcs app.Services, cfg app.Config, args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("doc-export", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	var req service.DocExportRequest
	fs.StringVar(&req.ID, "id", "", "Document ID")
	fs.StringVar(&req.MIME, "mime", "", "Export MIME type")
	fs.StringVar(&req.OutPath, "out", "", "Output path (default stdout)")
	if err := fs.Parse(args); err != nil {
		err = fail.NewValidation(err.Error(), "usage: gdrivectl doc-export --id <doc_id> --mime <mime>")
		writeError(stderr, err)
		return fail.ExitCode(err)
	}
	if req.ID == "" || req.MIME == "" {
		err := fail.NewValidation("missing required flags", "provide --id and --mime")
		writeError(stderr, err)
		return fail.ExitCode(err)
	}
	data, err := svcs.DocExport.Run(ctx, req)
	if err != nil {
		writeError(stderr, err)
		return fail.ExitCode(err)
	}

	if req.OutPath == "" {
		_, err = stdout.Write(data)
	} else {
		err = os.WriteFile(req.OutPath, data, 0o644)
	}
	if err != nil {
		writeError(stderr, fail.NewAPI(err.Error(), "unable to write export output", "check file path/permissions"))
		return fail.CodeAPI
	}
	return fail.CodeOK
}
