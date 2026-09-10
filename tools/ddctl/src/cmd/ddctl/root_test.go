package main

import (
	"bytes"
	"strings"
	"testing"

	"github.com/Alechan/ai-resources/tools/ddctl/src/internal/app"
	"github.com/Alechan/ai-resources/tools/ddctl/src/internal/fail"
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
	if !strings.Contains(stdout.String(), "logs") {
		t.Fatalf("expected usage output to include logs command, got:\n%s", stdout.String())
	}
	if !strings.Contains(stdout.String(), "dashboards") {
		t.Fatalf("expected usage output to include dashboards command, got:\n%s", stdout.String())
	}
	if !strings.Contains(stdout.String(), "monitors") {
		t.Fatalf("expected usage output to include monitors command, got:\n%s", stdout.String())
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
				"list",
				"search",
				"validate",
				"create",
				"update",
				"clone",
				"delete",
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
			name: "list help",
			args: []string{"dashboards", "list", "--help"},
			shouldContain: []string{
				"ddctl dashboards list",
				"--limit",
			},
		},
		{
			name: "search help",
			args: []string{"dashboards", "search", "--help"},
			shouldContain: []string{
				"ddctl dashboards search",
				"--title",
				"--tag",
			},
		},
		{
			name: "clone help",
			args: []string{"dashboards", "clone", "--help"},
			shouldContain: []string{
				"ddctl dashboards clone",
				"--title",
				"--dry-run",
			},
		},
		{
			name: "delete help",
			args: []string{"dashboards", "delete", "--help"},
			shouldContain: []string{
				"ddctl dashboards delete",
				"--confirm",
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

func TestExecute_MonitorsNestedHelp(t *testing.T) {
	tests := []struct {
		name          string
		args          []string
		shouldContain []string
	}{
		{
			name: "group help",
			args: []string{"monitors", "--help"},
			shouldContain: []string{
				"ddctl monitors",
				"list",
				"get",
				"validate",
				"create",
				"update",
				"mute",
				"unmute",
				"delete",
			},
		},
		{
			name: "list help",
			args: []string{"monitors", "list", "--help"},
			shouldContain: []string{"ddctl monitors list", "--tag"},
		},
		{
			name: "get help",
			args: []string{"monitors", "get", "--help"},
			shouldContain: []string{"ddctl monitors get <id>"},
		},
		{
			name: "validate help",
			args: []string{"monitors", "validate", "--help"},
			shouldContain: []string{"ddctl monitors validate", "--from-file"},
		},
		{
			name: "create help",
			args: []string{"monitors", "create", "--help"},
			shouldContain: []string{"ddctl monitors create", "--dry-run", "--muted"},
		},
		{
			name: "update help",
			args: []string{"monitors", "update", "--help"},
			shouldContain: []string{"ddctl monitors update", "--replace-all", "--if-unmodified-since"},
		},
		{
			name: "mute help",
			args: []string{"monitors", "mute", "--help"},
			shouldContain: []string{"ddctl monitors mute", "--until"},
		},
		{
			name: "unmute help",
			args: []string{"monitors", "unmute", "--help"},
			shouldContain: []string{"ddctl monitors unmute", "--confirm"},
		},
		{
			name: "delete help",
			args: []string{"monitors", "delete", "--help"},
			shouldContain: []string{"ddctl monitors delete", "--confirm"},
		},
		{
			name: "help subcommand form",
			args: []string{"help", "monitors", "mute"},
			shouldContain: []string{"ddctl monitors mute"},
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

func TestExecute_NotebooksValidateHelp(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := Execute([]string{"notebooks", "validate", "--help"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("code = %d stderr=%s", code, stderr.String())
	}
	out := stdout.String()
	for _, want := range []string{"--from-file", "warning", "exit 0"} {
		if !strings.Contains(out, want) {
			t.Fatalf("help missing %q:\n%s", want, out)
		}
	}
}

func TestExecute_NotebooksNestedHelp(t *testing.T) {
	tests := []struct {
		name          string
		args          []string
		shouldContain []string
	}{
		{
			name: "group help",
			args: []string{"notebooks", "--help"},
			shouldContain: []string{
				"ddctl notebooks",
				"get",
				"create",
				"update",
				"validate",
			},
		},
		{
			name: "create help",
			args: []string{"notebooks", "create", "--help"},
			shouldContain: []string{
				"ddctl notebooks create",
				"--dry-run",
				"--skip-validate",
			},
		},
		{
			name: "update help",
			args: []string{"notebooks", "update", "--help"},
			shouldContain: []string{
				"ddctl notebooks update",
				"--replace-all",
				"--if-unmodified-since",
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
			out := stdout.String()
			for _, want := range tc.shouldContain {
				if !strings.Contains(out, want) {
					t.Fatalf("help missing %q:\n%s", want, out)
				}
			}
		})
	}
}

func TestExecute_RemovedHyphenatedCommands(t *testing.T) {
	tests := []struct {
		oldCmd string
		newUse string
	}{
		{"logs-query", "ddctl logs query"},
		{"metrics-query", "ddctl metrics query"},
		{"events-list", "ddctl events list"},
	}
	for _, tc := range tests {
		t.Run(tc.oldCmd, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			code := Execute([]string{tc.oldCmd}, &stdout, &stderr)
			if code != fail.CodeValidation {
				t.Fatalf("code = %d, want %d", code, fail.CodeValidation)
			}
			if !strings.Contains(stderr.String(), tc.newUse) {
				t.Fatalf("stderr = %s", stderr.String())
			}
		})
	}
}

func TestExecute_QueryGroupsHelp(t *testing.T) {
	tests := []struct {
		name          string
		args          []string
		shouldContain []string
	}{
		{
			name: "logs group",
			args: []string{"logs", "--help"},
			shouldContain: []string{"ddctl logs", "query"},
		},
		{
			name: "logs query help",
			args: []string{"logs", "query", "--help"},
			shouldContain: []string{"ddctl logs query", "--count-only", "--all"},
		},
		{
			name: "metrics group",
			args: []string{"metrics", "--help"},
			shouldContain: []string{"ddctl metrics", "query"},
		},
		{
			name: "metrics query help",
			args: []string{"metrics", "query", "--help"},
			shouldContain: []string{"ddctl metrics query", "--query"},
		},
		{
			name: "events group",
			args: []string{"events", "--help"},
			shouldContain: []string{"ddctl events", "list"},
		},
		{
			name: "events list help",
			args: []string{"events", "list", "--help"},
			shouldContain: []string{"ddctl events list", "--count-only", "--all"},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			code := Execute(tc.args, &stdout, &stderr)
			if code != 0 {
				t.Fatalf("code = %d stderr=%s", code, stderr.String())
			}
			out := stdout.String()
			for _, want := range tc.shouldContain {
				if !strings.Contains(out, want) {
					t.Fatalf("help missing %q:\n%s", want, out)
				}
			}
		})
	}
}

func TestExecute_JSONErrorEnvelope(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := Execute([]string{"--json", "dashboards", "get"}, &stdout, &stderr)
	if code == 0 {
		t.Fatal("expected validation failure")
	}
	if !strings.Contains(stderr.String(), `"error"`) {
		t.Fatalf("stderr = %s", stderr.String())
	}
	if strings.Contains(stderr.String(), "Error [validation]:") {
		t.Fatalf("stderr should be JSON only: %s", stderr.String())
	}
}

func TestWriteError_IncludesRedactedResponseBody(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	writeError(&buf, &fail.Error{
		Category:   "api",
		Message:    "Bad Request",
		Action:     "inspect API response",
		Details:    `{"errors":["Bad Request"]}`,
		HTTPStatus: 400,
		Errors:     []string{"Bad Request"},
	}, app.Config{})

	out := buf.String()
	for _, want := range []string{
		"Error [api]: Bad Request",
		"Details: {\"errors\":[\"Bad Request\"]}",
		"Server error: Bad Request",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("writeError missing %q:\n%s", want, out)
		}
	}
}
