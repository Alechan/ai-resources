package search

import (
	"bytes"
	"encoding/json"
	"errors"
	"strings"
	"unicode"

	"github.com/Alechan/ai-resources/tools/gchatctl/src/internal/topics"
)

type Result struct {
	RPC   string `json:"rpc,omitempty"`
	Query string `json:"query,omitempty"`
	Hits  []Hit  `json:"hits"`
}

type Hit struct {
	SpaceID      string `json:"space_id,omitempty"`
	TopicID      string `json:"topic_id,omitempty"`
	ThreadWebID  string `json:"thread_web_id,omitempty"`
	MessageWebID string `json:"message_web_id,omitempty"`
	Permalink    string `json:"permalink,omitempty"`
	Text         string `json:"text,omitempty"`
}

func Parse(body []byte) (Result, error) {
	trimmed := stripEnvelope(body)
	if len(trimmed) == 0 {
		return Result{}, errors.New("empty Google Chat search response")
	}
	decoder := json.NewDecoder(bytes.NewReader(trimmed))
	decoder.UseNumber()
	var root any
	if err := decoder.Decode(&root); err != nil {
		return Result{}, errors.New("malformed Google Chat search response")
	}
	row := unwrapRow(root)
	result := Result{Hits: []Hit{}}
	if len(row) > 1 {
		result.RPC, _ = row[1].(string)
	}
	payload := payloadJSON(row)
	if payload == "" {
		return result, nil
	}
	var inner any
	if err := json.Unmarshal([]byte(payload), &inner); err != nil {
		return Result{}, errors.New("malformed Google Chat search payload")
	}
	for _, item := range findHits(inner) {
		if hit, ok := parseHit(item); ok {
			result.Hits = append(result.Hits, hit)
		}
	}
	return result, nil
}

func stripEnvelope(body []byte) []byte {
	trimmed := string(topics.StripXSSI(body))
	trimmed = strings.TrimSpace(trimmed)
	for {
		i := 0
		for i < len(trimmed) && trimmed[i] >= '0' && trimmed[i] <= '9' {
			i++
		}
		if i == 0 || i >= len(trimmed) || trimmed[i] != '\n' {
			return []byte(trimmed)
		}
		trimmed = trimmed[i+1:]
	}
}

func unwrapRow(root any) []any {
	list, ok := root.([]any)
	if !ok {
		return nil
	}
	if len(list) == 1 {
		if row, ok := list[0].([]any); ok {
			return row
		}
	}
	if len(list) > 2 {
		if _, ok := list[0].(string); ok {
			return list
		}
	}
	return list
}

func payloadJSON(row []any) string {
	if len(row) > 2 {
		if payload, ok := row[2].(string); ok {
			return payload
		}
	}
	return ""
}

func findHits(value any) []any {
	list, ok := value.([]any)
	if !ok {
		return nil
	}
	if looksLikeHitList(list) {
		return list
	}
	for _, item := range list {
		if found := findHits(item); len(found) > 0 {
			return found
		}
	}
	return nil
}

func looksLikeHitList(list []any) bool {
	if len(list) == 0 {
		return false
	}
	matched := 0
	for _, item := range list {
		if _, ok := parseHit(item); ok {
			matched++
		}
	}
	return matched > 0 && matched*2 >= len(list)
}

func parseHit(value any) (Hit, bool) {
	item, ok := asList(value)
	if !ok || len(item) < 3 {
		return Hit{}, false
	}
	spaceID := spaceIDFrom(item[0])
	if spaceID == "" {
		return Hit{}, false
	}
	hit := Hit{SpaceID: spaceID, TopicID: topicIDFrom(item)}
	hit.ThreadWebID, hit.MessageWebID = webIDsFrom(item, spaceID)
	hit.Permalink = permalinkFor(hit)
	if len(item) > 48 {
		if text, ok := item[48].(string); ok {
			hit.Text = text
		}
	}
	if hit.Text == "" && len(item) > 47 {
		if text, ok := item[47].(string); ok {
			hit.Text = text
		}
	}
	return hit, true
}

func topicIDFrom(item []any) string {
	for _, index := range []int{2, 8, 9, 27} {
		if index >= len(item) {
			continue
		}
		topic, ok := item[index].(string)
		if ok && isTopicID(topic) {
			return topic
		}
	}
	return ""
}

func isTopicID(value string) bool {
	if len(value) < 15 || len(value) > 16 {
		return false
	}
	for _, r := range value {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

func webIDsFrom(item []any, spaceID string) (string, string) {
	var ids []string
	for i, slot := range item {
		if i == 47 || i == 48 {
			continue
		}
		collectWebIDs(slot, spaceID, &ids)
	}
	switch len(ids) {
	case 0:
		return "", ""
	case 1:
		return ids[0], ids[0]
	default:
		return ids[0], ids[1]
	}
}

func collectWebIDs(value any, spaceID string, ids *[]string) {
	switch typed := value.(type) {
	case string:
		if !isSpaceID(typed) || typed == spaceID {
			return
		}
		for _, existing := range *ids {
			if existing == typed {
				return
			}
		}
		*ids = append(*ids, typed)
	case []any:
		for _, item := range typed {
			collectWebIDs(item, spaceID, ids)
		}
	}
}

func permalinkFor(hit Hit) string {
	thread := hit.ThreadWebID
	message := hit.MessageWebID
	if thread == "" {
		thread = message
	}
	if message == "" {
		message = thread
	}
	if hit.SpaceID == "" || thread == "" {
		return ""
	}
	return "https://chat.google.com/room/" + hit.SpaceID + "/" + thread + "/" + message
}

func spaceIDFrom(value any) string {
	if id, ok := value.(string); ok {
		return trailingSpaceID(id)
	}
	ident, ok := asList(value)
	if !ok {
		return ""
	}
	for _, item := range ident {
		if id, ok := item.(string); ok {
			if parsed := trailingSpaceID(id); parsed != "" {
				return parsed
			}
		}
	}
	return ""
}

func trailingSpaceID(value string) string {
	if i := strings.LastIndex(value, "/"); i >= 0 {
		value = value[i+1:]
	}
	if !isSpaceID(value) {
		return ""
	}
	return value
}

func isSpaceID(value string) bool {
	if len(value) != 11 {
		return false
	}
	hasLetter := false
	for _, r := range value {
		if unicode.IsLetter(r) {
			hasLetter = true
		} else if !unicode.IsDigit(r) && r != '_' && r != '-' {
			return false
		}
	}
	return hasLetter
}

func asList(value any) ([]any, bool) {
	list, ok := value.([]any)
	return list, ok
}
