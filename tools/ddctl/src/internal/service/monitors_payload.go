package service

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/Alechan/ai-resources/tools/ddctl/src/internal/fail"
)

var monitorIdentityFields = []string{
	"id",
	"url",
	"created",
	"created_at",
	"modified",
	"modified_at",
	"creator",
	"matching_downtimes",
	"overall_state",
	"state",
}

// NormalizeMonitorPayload accepts a raw monitor object or {"monitor": {...}}.
func NormalizeMonitorPayload(raw []byte) (map[string]any, error) {
	var env map[string]any
	if err := json.Unmarshal(raw, &env); err != nil {
		return nil, fail.NewValidation("invalid JSON monitor file", "ensure the file contains a monitor object")
	}
	if inner, ok := env["monitor"].(map[string]any); ok {
		return cloneMap(inner), nil
	}
	return env, nil
}

func PrepareMonitorCreatePayload(env map[string]any) (map[string]any, error) {
	payload := cloneMap(env)
	if err := assertMonitorRequired(payload); err != nil {
		return nil, err
	}
	for _, k := range monitorIdentityFields {
		delete(payload, k)
	}
	return payload, nil
}

func PrepareMonitorUpdatePayload(env map[string]any, monitorID int64, replaceAll bool) (map[string]any, error) {
	if !replaceAll {
		return nil, fail.NewValidation("--replace-all is required", "update is full replacement; pass --replace-all to confirm")
	}
	if monitorID <= 0 {
		return nil, fail.NewValidation("missing monitor ID", "usage: ddctl monitors update <id> --from-file <path> --replace-all")
	}
	payload := cloneMap(env)
	if err := assertMonitorRequired(payload); err != nil {
		return nil, err
	}
	payload["id"] = monitorID
	return payload, nil
}

func assertMonitorRequired(payload map[string]any) error {
	name, _ := payload["name"].(string)
	if strings.TrimSpace(name) == "" {
		return fail.NewValidation(`missing "name"`, "set name in the file")
	}
	query, _ := payload["query"].(string)
	if strings.TrimSpace(query) == "" {
		return fail.NewValidation(`missing "query"`, "set query in the file")
	}
	monitorType, _ := payload["type"].(string)
	if strings.TrimSpace(monitorType) == "" {
		return fail.NewValidation(`missing "type"`, "set type in the file")
	}
	return nil
}

func monitorQueryFromPayload(payload map[string]any) string {
	q, _ := payload["query"].(string)
	return strings.TrimSpace(q)
}

func metricQueryForPreflight(monitorQuery string) string {
	q := strings.TrimSpace(monitorQuery)
	if idx := strings.Index(q, "):"); idx >= 0 {
		q = q[idx+2:]
	}
	for _, sep := range []string{" >= ", " <= ", " > ", " < ", " == ", " != "} {
		if cut, _, ok := strings.Cut(q, sep); ok {
			return strings.TrimSpace(cut)
		}
	}
	return strings.TrimSpace(q)
}

func monitorModifiedAt(payload map[string]any) string {
	if v, ok := payload["modified"]; ok && fmt.Sprint(v) != "" {
		return fmt.Sprint(v)
	}
	return fmt.Sprint(payload["modified_at"])
}

func monitorIDFrom(payload map[string]any) int64 {
	switch v := payload["id"].(type) {
	case float64:
		return int64(v)
	case int64:
		return v
	case int:
		return int64(v)
	case json.Number:
		n, _ := v.Int64()
		return n
	case string:
		n, _ := strconv.ParseInt(v, 10, 64)
		return n
	default:
		return 0
	}
}

func monitorTags(payload map[string]any) []string {
	raw, _ := payload["tags"].([]any)
	var tags []string
	for _, item := range raw {
		if s, ok := item.(string); ok {
			tags = append(tags, s)
		}
	}
	return tags
}

func isProductionMonitor(tags []string) bool {
	for _, tag := range tags {
		switch tag {
		case "env:prod", "env:production":
			return true
		}
	}
	return false
}

func monitorCanonicalURL(site string, id int64) string {
	if site == "" {
		site = "datadoghq.com"
	}
	return fmt.Sprintf("https://app.%s/monitors/%d", site, id)
}

func attachMonitorURL(payload map[string]any, site string, id int64) {
	if payload == nil || id <= 0 {
		return
	}
	payload["url"] = monitorCanonicalURL(site, id)
}

func DiffMonitorPayloads(current, next map[string]any) string {
	left := cloneMap(current)
	right := cloneMap(next)
	for _, k := range monitorIdentityFields {
		delete(left, k)
		delete(right, k)
	}
	return SemanticDiffDashboardPayloads(left, right)
}
