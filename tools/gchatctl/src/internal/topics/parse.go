package topics

import (
	"bytes"
	"encoding/json"
	"errors"
	"strconv"
	"strings"
	"time"
)

type Page struct {
	RPC      string  `json:"rpc,omitempty"`
	Topics   []Topic `json:"topics"`
	Complete bool    `json:"complete"`
}

type Topic struct {
	ID        string    `json:"id"`
	SpaceID   string    `json:"space_id,omitempty"`
	CreatedAt time.Time `json:"created_at,omitempty"`
	Messages  []Message `json:"messages"`
}

type Message struct {
	ID        string    `json:"id,omitempty"`
	WebID     string    `json:"web_id,omitempty"`
	TopicID   string    `json:"topic_id,omitempty"`
	AuthorID  string    `json:"author_id,omitempty"`
	CreatedAt time.Time `json:"created_at,omitempty"`
	Text      string    `json:"text"`
}

func StripXSSI(body []byte) []byte {
	trimmed := bytes.TrimSpace(body)
	if bytes.HasPrefix(trimmed, []byte(")]}'")) {
		trimmed = bytes.TrimSpace(trimmed[4:])
	}
	return trimmed
}

func Parse(body []byte) (Page, error) {
	trimmed := StripXSSI(body)
	if len(trimmed) == 0 {
		return Page{}, errors.New("empty Google Chat response")
	}
	decoder := json.NewDecoder(bytes.NewReader(trimmed))
	decoder.UseNumber()
	var root any
	if err := decoder.Decode(&root); err != nil {
		return Page{}, errors.New("malformed Google Chat response")
	}
	envelope, ok := asList(root)
	if !ok || len(envelope) == 0 {
		return Page{}, errors.New("malformed Google Chat response")
	}
	inner, ok := asList(envelope[0])
	if !ok || len(inner) < 2 {
		return Page{}, errors.New("malformed Google Chat response")
	}
	page := Page{Complete: true, Topics: []Topic{}}
	page.RPC, _ = inner[0].(string)
	if len(inner) > 4 {
		if complete, ok := inner[4].(bool); ok {
			page.Complete = complete
		}
	}
	for _, item := range findTopicItems(inner) {
		topic, ok := parseTopic(item)
		if ok {
			page.Topics = append(page.Topics, topic)
		}
	}
	return page, nil
}

func findTopicItems(value any) []any {
	list, ok := asList(value)
	if !ok {
		return nil
	}
	if looksLikeTopicList(list) {
		return list
	}
	for _, item := range list {
		if found := findTopicItems(item); found != nil {
			return found
		}
	}
	return nil
}

func looksLikeTopicList(list []any) bool {
	if len(list) == 0 {
		return false
	}
	matched := 0
	for _, item := range list {
		if _, ok := parseTopic(item); ok {
			matched++
		}
	}
	return matched > 0 && matched*2 >= len(list)
}

func Filter(page Page, query string) Page {
	query = strings.TrimSpace(query)
	if query == "" {
		return page
	}
	needle := strings.ToLower(query)
	filtered := Page{RPC: page.RPC, Complete: page.Complete, Topics: []Topic{}}
	for _, topic := range page.Topics {
		kept := Topic{ID: topic.ID, SpaceID: topic.SpaceID, CreatedAt: topic.CreatedAt, Messages: []Message{}}
		for _, message := range topic.Messages {
			if strings.Contains(strings.ToLower(message.Text), needle) {
				kept.Messages = append(kept.Messages, message)
			}
		}
		if len(kept.Messages) > 0 {
			filtered.Topics = append(filtered.Topics, kept)
		}
	}
	return filtered
}

func FilterSpace(page Page, spaceID string) Page {
	spaceID = strings.TrimSpace(spaceID)
	if spaceID == "" {
		return page
	}
	filtered := Page{RPC: page.RPC, Complete: page.Complete, Topics: []Topic{}}
	for _, topic := range page.Topics {
		if topic.SpaceID == spaceID {
			filtered.Topics = append(filtered.Topics, topic)
		}
	}
	return filtered
}

func Apply(page Page, spaceID, query string) Page {
	return Filter(FilterSpace(page, spaceID), query)
}

func Counts(page Page) (topicCount, messageCount int) {
	topicCount = len(page.Topics)
	for _, topic := range page.Topics {
		messageCount += len(topic.Messages)
	}
	return topicCount, messageCount
}

func parseTopic(value any) (Topic, bool) {
	item, ok := asList(value)
	if !ok || len(item) < 7 {
		return Topic{}, false
	}
	id := asString(item[1])
	if id == "" {
		return Topic{}, false
	}
	topic := Topic{ID: id, CreatedAt: parseMicroTime(id), Messages: []Message{}}
	if ident, ok := asList(item[0]); ok && len(ident) > 1 {
		if space := asString(ident[1]); space != "" {
			topic.SpaceID = space
		}
	}
	messages, ok := asList(item[6])
	if !ok {
		return topic, true
	}
	for _, raw := range messages {
		message, ok := parseMessage(raw, id, topic.CreatedAt)
		if ok {
			topic.Messages = append(topic.Messages, message)
		}
	}
	return topic, true
}

func parseMessage(value any, topicID string, fallback time.Time) (Message, bool) {
	item, ok := asList(value)
	if !ok || len(item) < 10 {
		return Message{}, false
	}
	text := asString(item[9])
	if text == "" && len(item) > 10 && item[10] != nil {
		text = "(attachment)"
	}
	message := Message{
		TopicID:   topicID,
		CreatedAt: fallback,
		Text:      text,
	}
	if id := asString(item[2]); id != "" {
		message.ID = id
	}
	if len(item) > 13 {
		if webID := asString(item[13]); webID != "" {
			message.WebID = webID
		}
	}
	if author, ok := asList(item[0]); ok && len(author) > 1 {
		if id := asString(author[1]); id != "" {
			message.AuthorID = id
		}
	}
	return message, true
}

func asList(value any) ([]any, bool) {
	list, ok := value.([]any)
	return list, ok
}

func asString(value any) string {
	switch typed := value.(type) {
	case string:
		return typed
	case json.Number:
		return typed.String()
	case float64:
		return strconv.FormatInt(int64(typed), 10)
	default:
		return ""
	}
}

func parseMicroTime(id string) time.Time {
	n, err := strconv.ParseInt(id, 10, 64)
	if err != nil || n <= 0 {
		return time.Time{}
	}
	return time.UnixMicro(n).UTC()
}
