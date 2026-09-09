package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestExecuteHelp(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := Execute([]string{"--help"}, &stdout, &stderr)
	if code != 0 {
		t.Errorf("expected exit 0 for --help, got %d; stderr: %s", code, stderr.String())
	}
	if stdout.Len() == 0 {
		t.Error("expected usage output on --help, got nothing")
	}
	if !strings.Contains(stdout.String(), "notebooks") {
		t.Fatalf("expected usage output to include notebooks command, got:\n%s", stdout.String())
	}
	if !strings.Contains(stdout.String(), "dashboards") {
		t.Fatalf("expected usage output to include dashboards command, got:\n%s", stdout.String())
	}
}

func TestExecuteNoArgs(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := Execute([]string{}, &stdout, &stderr)
	if code == 0 {
		t.Errorf("expected non-zero exit for no args, got 0")
	}
}

func TestExecuteUnknownCommand(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := Execute([]string{"not-a-command"}, &stdout, &stderr)
	if code == 0 {
		t.Errorf("expected non-zero exit for unknown command, got 0")
	}
}

func TestExecute_DashboardsNestedHelp(t *testing.T) {
	tests := []struct {
		name          string
		args          []string
		shouldContain []string
	}{
		{
			name: "group help",
			args: []string{"dashboards", "--help"},
			shouldContain: []string{
				"ddctl dashboards",
				"get",
				"validate",
				"create",
				"update",
			},
		},
		{
			name: "get help",
			args: []string{"dashboards", "get", "--help"},
			shouldContain: []string{
				"ddctl dashboards get <id>",
				"Positional arguments",
			},
		},
		{
			name: "validate help",
			args: []string{"dashboards", "validate", "--help"},
			shouldContain: []string{
				"ddctl dashboards validate",
				"--from-file",
				"--template-variable",
			},
		},
		{
			name: "create help",
			args: []string{"dashboards", "create", "--help"},
			shouldContain: []string{
				"ddctl dashboards create",
				"--dry-run",
				"POST",
			},
		},
		{
			name: "update help",
			args: []string{"dashboards", "update", "--help"},
			shouldContain: []string{
				"ddctl dashboards update <id>",
				"--replace-all",
				"--if-unmodified-since",
				"--diff",
			},
		},
		{
			name: "help subcommand form",
			args: []string{"help", "dashboards", "update"},
			shouldContain: []string{
				"ddctl dashboards update <id>",
				"--replace-all",
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			code := Execute(tc.args, &stdout, &stderr)
			if code != 0 {
				t.Fatalf("code = %d stderr=%s stdout=%s", code, stderr.String(), stdout.String())
			}
			if stderr.Len() != 0 {
				t.Fatalf("help should not write stderr, got %q", stderr.String())
			}
			out := stdout.String()
			for _, want := range tc.shouldContain {
				if !strings.Contains(out, want) {
					t.Fatalf("help missing %q:\n%s", want, out)
				}
			}
			if strings.Contains(out, "Usage: ddctl [global flags]") {
				t.Fatalf("nested help printed global usage:\n%s", out)
			}
		})
	}
}

func TestExecute_DashboardsHelpReturnsZero(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := Execute([]string{"dashboards", "get", "-h"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("code = %d stderr=%s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "ddctl dashboards get") {
		t.Fatalf("stdout = %s", stdout.String())
	}
}
