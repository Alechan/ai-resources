package main

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Alechan/ai-resources/tools/ddctl/src/internal/app"
	"github.com/Alechan/ai-resources/tools/ddctl/src/internal/service"
)

func TestSplitLeadingPositional(t *testing.T) {
	id, rest := splitLeadingPositional([]string{"14515133", "--from-file", "/tmp/a.json"})
	if id != "14515133" {
		t.Fatalf("id = %q, want 14515133", id)
	}
	if len(rest) != 2 || rest[0] != "--from-file" {
		t.Fatalf("rest = %#v", rest)
	}
}

func TestNormalizeNotebookID(t *testing.T) {
	if got := normalizeNotebookID(float64(14515133)); got != "14515133" {
		t.Fatalf("normalizeNotebookID(float64) = %q", got)
	}
	if got := normalizeNotebookID("14515133"); got != "14515133" {
		t.Fatalf("normalizeNotebookID(string) = %q", got)
	}
}

func TestRunNotebooksUpdate_RequiresReplaceAll(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "payload.json")
	err := os.WriteFile(file, []byte(`{
  "attributes": {
    "name": "Notebook A",
    "time": {"live_span":"1w"},
    "cells": [{"id":"abc12345","type":"notebook_cells","attributes":{"definition":{"type":"rich_text"}}}],
    "template_variables": [],
    "schema_version": 26
  }
}`), 0o600)
	if err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	svcs := app.Services{
		Notebooks: service.NewNotebooksService(nil, nil),
	}
	cfg := app.NewConfig("datadoghq.com", 10*time.Second, false, false)

	var out, stderr bytes.Buffer
	code := runNotebooksUpdateCmd(
		context.Background(),
		svcs,
		cfg,
		[]string{"14515133", "--from-file", file},
		&out,
		&stderr,
	)
	if code == 0 {
		t.Fatalf("expected non-zero code, got 0")
	}
	if !strings.Contains(stderr.String(), "--replace-all is required") {
		t.Fatalf("stderr does not contain replace-all validation: %s", stderr.String())
	}
}
