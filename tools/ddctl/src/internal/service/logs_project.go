package service

import (
	"encoding/json"
	"sort"
	"strings"
)

var logEventTopLevelFields = map[string]struct{}{
	"id":        {},
	"timestamp": {},
	"status":    {},
	"service":   {},
	"host":      {},
	"message":   {},
}

// ParseLogFieldList parses a comma-separated field list. Nil means full event ("*").
func ParseLogFieldList(raw string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" || raw == "*" {
		return nil
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		out = append(out, part)
	}
	return out
}

// ProjectLogEventFields returns a map with only requested fields.
func ProjectLogEventFields(event LogEvent, fields []string) map[string]any {
	if len(fields) == 0 {
		return LogEventToMap(event)
	}
	out := make(map[string]any, len(fields))
	for _, field := range fields {
		if v, ok := logEventTopLevelValue(event, field); ok {
			out[field] = v
			continue
		}
		if event.Custom != nil {
			if v, ok := event.Custom[field]; ok {
				out[field] = v
			}
		}
	}
	return out
}

func LogEventToMap(event LogEvent) map[string]any {
	out := map[string]any{
		"id":        event.ID,
		"timestamp": event.Timestamp,
		"status":    event.Status,
		"service":   event.Service,
		"host":      event.Host,
		"message":   event.Message,
	}
	if len(event.Custom) > 0 {
		out["custom"] = event.Custom
	}
	if len(event.Tags) > 0 {
		out["tags"] = event.Tags
	}
	return out
}

func logEventTopLevelValue(event LogEvent, field string) (any, bool) {
	if _, ok := logEventTopLevelFields[field]; !ok {
		return nil, false
	}
	switch field {
	case "id":
		return event.ID, event.ID != ""
	case "timestamp":
		return event.Timestamp, event.Timestamp != ""
	case "status":
		return event.Status, event.Status != ""
	case "service":
		return event.Service, event.Service != ""
	case "host":
		return event.Host, event.Host != ""
	case "message":
		return event.Message, event.Message != ""
	default:
		return nil, false
	}
}

var verboseSkipCustomKeys = map[string]struct{}{
	"file":   {},
	"func":   {},
	"log":    {},
	"hash":   {},
	"fields": {},
	"time":   {},
	"level":  {},
}

// VerboseCustomLines returns formatted custom key lines for text output.
func VerboseCustomLines(event LogEvent, keys []string) []string {
	selected := verboseCustomKeys(event, keys)
	lines := make([]string, 0, len(selected))
	for _, item := range selected {
		lines = append(lines, formatVerboseCustomLine(item.key, item.value))
	}
	return lines
}

type verboseCustomItem struct {
	key   string
	value any
}

func verboseCustomKeys(event LogEvent, keys []string) []verboseCustomItem {
	if len(event.Custom) == 0 {
		return nil
	}
	if len(keys) > 0 {
		out := make([]verboseCustomItem, 0, len(keys))
		for _, key := range keys {
			value, ok := event.Custom[key]
			if !ok || !verboseCustomValueAllowed(key, value) {
				continue
			}
			out = append(out, verboseCustomItem{key: key, value: value})
		}
		return out
	}

	var names []string
	for key, value := range event.Custom {
		if !verboseCustomValueAllowed(key, value) {
			continue
		}
		if key == "msg" && event.Message != "" {
			continue
		}
		names = append(names, key)
	}
	sort.Strings(names)
	if len(names) > 10 {
		names = names[:10]
	}
	out := make([]verboseCustomItem, 0, len(names))
	for _, key := range names {
		out = append(out, verboseCustomItem{key: key, value: event.Custom[key]})
	}
	return out
}

func verboseCustomValueAllowed(key string, value any) bool {
	if _, skip := verboseSkipCustomKeys[key]; skip {
		return false
	}
	switch value.(type) {
	case string, float64, json.Number, int, int64, bool:
		return true
	default:
		return false
	}
}

func formatVerboseCustomLine(key string, value any) string {
	switch v := value.(type) {
	case string:
		return key + "=" + v
	default:
		b, _ := json.Marshal(v)
		return key + "=" + string(b)
	}
}

// ProjectLogsQueryResult applies field projection to query JSON output.
func ProjectLogsQueryResult(result LogsQueryResult, fields []string) map[string]any {
	out := map[string]any{
		"hit_count":      result.HitCount,
		"returned_count": result.ReturnedCount,
	}
	if result.NextCursor != "" {
		out["next_cursor"] = result.NextCursor
	}
	if len(result.Warnings) > 0 {
		out["warnings"] = result.Warnings
	}
	if result.Truncated {
		out["truncated"] = true
		out["limit"] = result.Limit
	}
	if len(result.Data) > 0 {
		data := make([]map[string]any, len(result.Data))
		for i, event := range result.Data {
			data[i] = ProjectLogEventFields(event, fields)
		}
		out["data"] = data
	}
	return out
}
