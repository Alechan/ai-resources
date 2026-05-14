package service

import (
	"strings"
	"testing"
)

func TestNormalizeNotebookEnvelope_FromAttributesOnly(t *testing.T) {
	raw := []byte(`{
  "attributes": {
    "name": "Notebook A",
    "time": {"live_span":"1w"},
    "cells": [{"id":"abc12345","type":"notebook_cells","attributes":{"definition":{"type":"rich_text"}}}],
    "template_variables": [],
    "schema_version": 26
  }
}`)

	env, err := NormalizeNotebookEnvelope(raw)
	if err != nil {
		t.Fatalf("NormalizeNotebookEnvelope() error = %v", err)
	}

	data, ok := env["data"].(map[string]any)
	if !ok {
		t.Fatalf("expected data map, got %T", env["data"])
	}

	if got, _ := data["type"].(string); got != "notebooks" {
		t.Fatalf("data.type = %q, want %q", got, "notebooks")
	}

	attrs, ok := data["attributes"].(map[string]any)
	if !ok {
		t.Fatalf("expected attributes map, got %T", data["attributes"])
	}

	if got, _ := attrs["name"].(string); got != "Notebook A" {
		t.Fatalf("attributes.name = %q, want %q", got, "Notebook A")
	}
}

func TestPrepareNotebookUpdatePayload_MissingCellsFails(t *testing.T) {
	raw := []byte(`{
  "data": {
    "type": "notebooks",
    "attributes": {
      "name": "Notebook A",
      "time": {"live_span":"1w"}
    }
  }
}`)

	env, err := NormalizeNotebookEnvelope(raw)
	if err != nil {
		t.Fatalf("NormalizeNotebookEnvelope() error = %v", err)
	}

	_, err = PrepareNotebookUpdatePayload(env, "14515133", true)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "attributes.cells") {
		t.Fatalf("error %q does not mention attributes.cells", err)
	}
}

func TestExtractTimeseriesQueries(t *testing.T) {
	raw := []byte(`{
  "data": {
    "type": "notebooks",
    "attributes": {
      "name": "Notebook A",
      "time": {"live_span":"1w"},
      "cells": [
        {
          "id": "ts1",
          "type": "notebook_cells",
          "attributes": {
            "definition": {
              "type": "timeseries",
              "requests": [
                {
                  "queries": [
                    {"data_source":"metrics","name":"query1","query":"sum:aws.sqs.number_of_messages_deleted{queuename:albatross*}"},
                    {"data_source":"metrics","name":"query2","query":"max:aws.sqs.approximate_number_of_messages_visible{queuename:albatross*}"}
                  ]
                }
              ]
            }
          }
        }
      ]
    }
  }
}`)

	env, err := NormalizeNotebookEnvelope(raw)
	if err != nil {
		t.Fatalf("NormalizeNotebookEnvelope() error = %v", err)
	}

	queries, err := ExtractTimeseriesQueries(env)
	if err != nil {
		t.Fatalf("ExtractTimeseriesQueries() error = %v", err)
	}
	if len(queries) != 2 {
		t.Fatalf("len(queries) = %d, want 2", len(queries))
	}

	if queries[0] != "sum:aws.sqs.number_of_messages_deleted{queuename:albatross*}" {
		t.Fatalf("queries[0] = %q", queries[0])
	}
	if queries[1] != "max:aws.sqs.approximate_number_of_messages_visible{queuename:albatross*}" {
		t.Fatalf("queries[1] = %q", queries[1])
	}
}
