package main

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Alechan/ai-resources/tools/gdrivectl/src/internal/fail"
)

func TestParseRootArgsDefaults(t *testing.T) {
	t.Setenv("GDRIVECTL_GCLOUD_BIN", "")
	opts, cmd, cmdArgs, err := parseRootArgs(nil)
	if err != nil {
		t.Fatalf("parseRootArgs() error = %v", err)
	}
	if cmd != "" {
		t.Fatalf("cmd = %q, want empty", cmd)
	}
	if len(cmdArgs) != 0 {
		t.Fatalf("cmdArgs len = %d, want 0", len(cmdArgs))
	}
	if opts.gcloudBin == "" {
		t.Fatalf("default gcloud bin should not be empty")
	}
	if opts.gcloudBin != "gcloud" {
		t.Fatalf("default gcloud bin = %q, want %q", opts.gcloudBin, "gcloud")
	}
	if opts.timeout != 20*time.Second {
		t.Fatalf("timeout = %v, want %v", opts.timeout, 20*time.Second)
	}
}

func TestParseRootArgsReadsEnvDefault(t *testing.T) {
	t.Setenv("GDRIVECTL_GCLOUD_BIN", "/tmp/from-env-gcloud")
	opts, _, _, err := parseRootArgs(nil)
	if err != nil {
		t.Fatalf("parseRootArgs() error = %v", err)
	}
	if opts.gcloudBin != "/tmp/from-env-gcloud" {
		t.Fatalf("gcloudBin = %q, want env value", opts.gcloudBin)
	}
}

func TestParseRootArgsFlagOverridesEnv(t *testing.T) {
	t.Setenv("GDRIVECTL_GCLOUD_BIN", "/tmp/from-env-gcloud")
	opts, cmd, _, err := parseRootArgs([]string{"--gcloud-bin", "/tmp/from-flag-gcloud", "doctor"})
	if err != nil {
		t.Fatalf("parseRootArgs() error = %v", err)
	}
	if cmd != "doctor" {
		t.Fatalf("cmd = %q, want doctor", cmd)
	}
	if opts.gcloudBin != "/tmp/from-flag-gcloud" {
		t.Fatalf("gcloudBin = %q, want flag value", opts.gcloudBin)
	}
}

func TestResolveGcloudBin(t *testing.T) {
	t.Run("resolves absolute executable path", func(t *testing.T) {
		p := filepath.Join(t.TempDir(), "fake-gcloud.sh")
		if err := os.WriteFile(p, []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
			t.Fatalf("write script: %v", err)
		}
		resolved, ok := resolveGcloudBin(p)
		if !ok {
			t.Fatalf("resolveGcloudBin() ok = false, want true")
		}
		if resolved != p {
			t.Fatalf("resolved = %q, want %q", resolved, p)
		}
	})

	t.Run("missing executable", func(t *testing.T) {
		p := filepath.Join(t.TempDir(), "missing-gcloud")
		resolved, ok := resolveGcloudBin(p)
		if ok {
			t.Fatalf("resolveGcloudBin() ok = true, want false")
		}
		if resolved != p {
			t.Fatalf("resolved = %q, want %q", resolved, p)
		}
	})
}

func TestParseRootArgsParsesGlobalFlags(t *testing.T) {
	args := []string{
		"--json",
		"--debug",
		"--timeout", "45s",
		"--gcloud-bin", "/tmp/gcloud",
		"search",
		"--query", "name contains 'x'",
	}

	opts, cmd, cmdArgs, err := parseRootArgs(args)
	if err != nil {
		t.Fatalf("parseRootArgs() error = %v", err)
	}
	if !opts.json || !opts.debug {
		t.Fatalf("json/debug parsing failed: %+v", opts)
	}
	if opts.timeout != 45*time.Second {
		t.Fatalf("timeout = %v, want 45s", opts.timeout)
	}
	if opts.gcloudBin != "/tmp/gcloud" {
		t.Fatalf("gcloudBin = %q, want /tmp/gcloud", opts.gcloudBin)
	}
	if cmd != "search" {
		t.Fatalf("cmd = %q, want search", cmd)
	}
	if len(cmdArgs) != 2 || cmdArgs[0] != "--query" {
		t.Fatalf("cmdArgs = %#v, want [--query ...]", cmdArgs)
	}
}

func TestParseRootArgsParsesGlobalFlagsAfterCommand(t *testing.T) {
	opts, cmd, cmdArgs, err := parseRootArgs([]string{
		"search", "--query", "abc", "--json", "--timeout=3s",
	})
	if err != nil {
		t.Fatalf("parseRootArgs() error = %v", err)
	}
	if cmd != "search" {
		t.Fatalf("cmd = %q, want search", cmd)
	}
	if !opts.json {
		t.Fatalf("opts.json = false, want true")
	}
	if opts.timeout != 3*time.Second {
		t.Fatalf("timeout = %v, want 3s", opts.timeout)
	}
	if len(cmdArgs) != 2 || cmdArgs[0] != "--query" || cmdArgs[1] != "abc" {
		t.Fatalf("cmdArgs = %#v, want [--query abc]", cmdArgs)
	}
}

func TestParseRootArgsValidationFailures(t *testing.T) {
	tests := []struct {
		name string
		args []string
	}{
		{name: "missing gcloud bin", args: []string{"--gcloud-bin"}},
		{name: "missing timeout", args: []string{"--timeout"}},
		{name: "invalid timeout", args: []string{"--timeout=notaduration"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, _, err := parseRootArgs(tt.args)
			var fe *fail.Error
			if !errors.As(err, &fe) {
				t.Fatalf("err type = %T, want *fail.Error", err)
			}
			if fe.Category != "validation" {
				t.Fatalf("category = %q, want validation", fe.Category)
			}
		})
	}
}

func TestExecuteValidationErrorsForRequiredFlags(t *testing.T) {
	tests := []struct {
		name         string
		args         []string
		wantInStderr string
	}{
		{name: "search missing query", args: []string{"search"}, wantInStderr: "missing --query"},
		{name: "file-meta missing id", args: []string{"file-meta"}, wantInStderr: "missing --id"},
		{name: "doc-tabs missing id", args: []string{"doc-tabs"}, wantInStderr: "missing --id"},
		{name: "doc-export missing required flags", args: []string{"doc-export"}, wantInStderr: "missing required flags"},
		{name: "upload missing path", args: []string{"upload"}, wantInStderr: "missing --path"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			code := Execute(tt.args, &stdout, &stderr)
			if code != fail.CodeValidation {
				t.Fatalf("exit code = %d, want %d", code, fail.CodeValidation)
			}
			if !strings.Contains(stderr.String(), tt.wantInStderr) {
				t.Fatalf("stderr = %q, want contains %q", stderr.String(), tt.wantInStderr)
			}
		})
	}
}

func TestExecuteUnknownCommand(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := Execute([]string{"unknown-cmd"}, &stdout, &stderr)
	if code != fail.CodeValidation {
		t.Fatalf("exit code = %d, want %d", code, fail.CodeValidation)
	}
	if !strings.Contains(stderr.String(), "unknown command") {
		t.Fatalf("stderr = %q, want contains unknown command", stderr.String())
	}
}

func TestExecuteHelp(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := Execute([]string{"--help"}, &stdout, &stderr)
	if code != fail.CodeOK {
		t.Fatalf("exit code = %d, want %d", code, fail.CodeOK)
	}
	if !strings.Contains(stdout.String(), "Usage: gdrivectl") {
		t.Fatalf("stdout = %q, want usage text", stdout.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q, want empty", stderr.String())
	}
}
