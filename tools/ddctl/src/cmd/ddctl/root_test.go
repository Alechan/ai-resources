package main

import (
	"bytes"
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
