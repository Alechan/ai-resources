package service

import (
	_ "embed"
	"fmt"
	"strings"
	"testing"
)

//go:embed testdata/notebook_beaver_runbook.json
var notebookBeaverRunbookFixture []byte

//go:embed testdata/notebook_alias_after_ui.json
var notebookAliasAfterUIFixture []byte

//go:embed testdata/notebook_metadata_alias_invalid.json
var notebookMetadataAliasInvalidFixture []byte

func TestNotebookBeaverFixture_StructureValid(t *testing.T) {
	t.Parallel()

	env, err := NormalizeNotebookEnvelope(notebookBeaverRunbookFixture)
	if err != nil {
		t.Fatalf("NormalizeNotebookEnvelope() error = %v", err)
	}
	warnings, err := PrepareNotebookSchema(env)
	if err != nil {
		t.Fatalf("PrepareNotebookSchema() error = %v", err)
	}
	if len(warnings) != 0 {
		t.Fatalf("warnings = %v", warnings)
	}
}

func TestNotebookBeaverFixture_ExtractMetricQueries(t *testing.T) {
	t.Parallel()

	env, err := NormalizeNotebookEnvelope(notebookBeaverRunbookFixture)
	if err != nil {
		t.Fatalf("NormalizeNotebookEnvelope() error = %v", err)
	}
	queries, err := ExtractNotebookMetricQueries(env)
	if err != nil {
		t.Fatalf("ExtractNotebookMetricQueries() error = %v", err)
	}
	if len(queries) != 4 {
		t.Fatalf("len(queries) = %d, want 4", len(queries))
	}
	if !strings.Contains(queries[0].Resolved, "kube_namespace:production") {
		t.Fatalf("resolved[0] = %q", queries[0].Resolved)
	}
}

func TestNotebookBeaverFixture_CreatePayloadMatchesValidate(t *testing.T) {
	t.Parallel()

	env, err := NormalizeNotebookEnvelope(notebookBeaverRunbookFixture)
	if err != nil {
		t.Fatalf("NormalizeNotebookEnvelope() error = %v", err)
	}
	if _, err := PrepareNotebookSchema(env); err != nil {
		t.Fatalf("PrepareNotebookSchema() error = %v", err)
	}
	if _, err := PrepareNotebookCreatePayload(env, "", ""); err != nil {
		t.Fatalf("PrepareNotebookCreatePayload() error = %v", err)
	}
}

func TestNotebookAliasAfterUIFixture_StructureValid(t *testing.T) {
	t.Parallel()

	env, err := NormalizeNotebookEnvelope(notebookAliasAfterUIFixture)
	if err != nil {
		t.Fatalf("NormalizeNotebookEnvelope() error = %v", err)
	}
	if _, err := PrepareNotebookSchema(env); err != nil {
		t.Fatalf("PrepareNotebookSchema() error = %v", err)
	}
	payload, err := PrepareNotebookUpdatePayload(env, "15505341", true)
	if err != nil {
		t.Fatalf("PrepareNotebookUpdatePayload() error = %v", err)
	}
	req := mustMap(mustMap(mustMap(payload["data"])["attributes"])["cells"].([]any)[3].(map[string]any)["attributes"].(map[string]any)["definition"].(map[string]any)["requests"].([]any)[0].(map[string]any))
	formulas, ok := req["formulas"].([]any)
	if !ok || len(formulas) == 0 {
		t.Fatalf("formulas = %#v", req["formulas"])
	}
	alias, _ := mustMap(formulas[0])["alias"].(string)
	if alias != "alias 1" {
		t.Fatalf("alias = %q", alias)
	}
}

func TestNotebookMetadataAliasNameFailsValidation(t *testing.T) {
	t.Parallel()

	env, err := NormalizeNotebookEnvelope(notebookMetadataAliasInvalidFixture)
	if err != nil {
		t.Fatalf("NormalizeNotebookEnvelope() error = %v", err)
	}
	_, err = PrepareNotebookSchema(env)
	if err == nil || !strings.Contains(err.Error(), "requests[0] alias") {
		t.Fatalf("PrepareNotebookSchema() error = %v", err)
	}
}

func TestResolveNotebookQuery_UnresolvedVariable(t *testing.T) {
	t.Parallel()

	_, err := resolveNotebookQuery("avg:system.cpu.user{kube_namespace:$missing.value}", nil)
	if err == nil {
		t.Fatal("expected unresolved variable error")
	}
}

func TestNormalizeTemplateVariableDefaultToDefaults(t *testing.T) {
	t.Parallel()

	attrs := map[string]any{
		"name": "n",
		"time": map[string]any{"live_span": "1h"},
		"cells": []any{
			map[string]any{
				"type": "notebook_cells",
				"attributes": map[string]any{
					"definition": map[string]any{"type": "markdown", "text": "x"},
				},
			},
		},
		"template_variables": []any{
			map[string]any{
				"name":    "environment",
				"prefix":  "kube_namespace",
				"default": "production",
			},
		},
	}
	warnings, err := normalizeNotebookTemplateVariables(attrs)
	if err != nil {
		t.Fatalf("normalizeNotebookTemplateVariables() error = %v", err)
	}
	if len(warnings) != 1 {
		t.Fatalf("warnings = %v", warnings)
	}
	tv := mustMap(mustMap(attrs["template_variables"].([]any)[0]))
	defaults, ok := tv["defaults"].([]any)
	if !ok || len(defaults) != 1 || fmt.Sprint(defaults[0]) != "production" {
		t.Fatalf("defaults = %#v", tv["defaults"])
	}
}
