package main

import (
	"strings"
	"testing"
	"time"
)

func TestRunSchedulePrintIsReadOnly(t *testing.T) {
	var stdout, stderr strings.Builder
	code := runWithLookPath([]string{"schedule", "print", "--csv", "/tmp/usage.csv", "--interval", "6h"}, fakeClient{}, &stdout, &stderr, time.Now, func(string) (string, error) {
		return "/opt/homebrew/bin/gh", nil
	})
	if code != exitOK {
		t.Fatalf("run() code = %d, stderr = %s", code, stderr.String())
	}
	for _, want := range []string{"current", "append", "--csv", "/tmp/usage.csv", "StartCalendarInterval"} {
		if !strings.Contains(stdout.String(), want) {
			t.Errorf("plist does not contain %q: %s", want, stdout.String())
		}
	}
}

func TestRunScheduleUninstallRequiresConfirmation(t *testing.T) {
	var stdout, stderr strings.Builder
	code := runWithLookPath([]string{"schedule", "uninstall"}, fakeClient{}, &stdout, &stderr, time.Now, func(string) (string, error) {
		return "/opt/homebrew/bin/gh", nil
	})
	if code != exitUsage {
		t.Fatalf("run() code = %d, want %d", code, exitUsage)
	}
	if !strings.Contains(stderr.String(), "--yes") {
		t.Fatalf("stderr = %s", stderr.String())
	}
}
