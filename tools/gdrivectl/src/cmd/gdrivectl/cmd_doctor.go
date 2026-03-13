package main

import (
	"context"
	"flag"
	"fmt"
	"io"

	"github.com/Alechan/ai-resources/tools/gdrivectl/src/internal/app"
	"github.com/Alechan/ai-resources/tools/gdrivectl/src/internal/fail"
)

func runDoctorCmd(ctx context.Context, svcs app.Services, cfg app.Config, args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("doctor", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	if err := fs.Parse(args); err != nil {
		err = fail.NewValidation(err.Error(), "usage: gdrivectl doctor")
		writeError(stderr, err)
		return fail.ExitCode(err)
	}

	report, err := svcs.Doctor.Run(ctx)
	if err != nil {
		writeError(stderr, err)
		return fail.ExitCode(err)
	}

	if cfg.JSON {
		if err := svcs.Output.JSON(stdout, report); err != nil {
			writeError(stderr, fail.NewAPI(err.Error(), "unable to encode doctor report", ""))
			return fail.CodeAPI
		}
		return fail.CodeOK
	}

	fmt.Fprintf(stdout, "gcloud binary: %s\n", report.GcloudBin)
	fmt.Fprintf(stdout, "gcloud exists: %t\n", report.GcloudExists)
	fmt.Fprintf(stdout, "token ok: %t\n", report.TokenOK)
	fmt.Fprintf(stdout, "drive endpoint ok: %t\n", report.DriveOK)
	fmt.Fprintf(stdout, "docs endpoint ok: %t\n", report.DocsOK)
	if report.Note != "" {
		fmt.Fprintf(stdout, "note: %s\n", report.Note)
	}
	return fail.CodeOK
}
