package topics

import (
	"bytes"
	"encoding/json"
	"errors"
	"strconv"
	"strings"
	"time"
)

const defaultPageSize = 32
const defaultBodySlots = 100

type RequestOptions struct {
	SpaceID  string
	Cursor   int64
	PageSize int
}

func DefaultBody(spaceID string, cursor int64, pageSize int) []any {
	if pageSize <= 0 {
		pageSize = defaultPageSize
	}
	if cursor <= 0 {
		cursor = time.Now().UnixMicro()
	}
	body := make([]any, defaultBodySlots)
	body[1] = pageSize
	body[3] = []any{nil, cursor}
	body[4] = []any{3, 1, 4}
	body[5] = 1000
	body[6] = 20
	body[7] = []any{[]any{spaceID}}
	body[8] = []any{cursor}
	body[9] = []any{cursor}
	body[10] = 2
	return body
}

func RewriteBody(raw string, opts RequestOptions) (string, error) {
	var body []any
	if strings.TrimSpace(raw) == "" {
		encoded, err := marshalBody(DefaultBody(opts.SpaceID, opts.Cursor, opts.PageSize))
		if err != nil {
			return "", err
		}
		return encoded, nil
	}
	decoder := json.NewDecoder(bytes.NewReader([]byte(raw)))
	decoder.UseNumber()
	if err := decoder.Decode(&body); err != nil {
		return "", errors.New("stored list_topics body is not JSON")
	}
	cursor := opts.Cursor
	ensureLen(&body, 11)
	if opts.PageSize > 0 {
		body[1] = opts.PageSize
	}
	if cursor > 0 {
		if isTopicSelector(body[3]) {
			body[3] = []any{nil, nil, nil, nil, []any{cursor}}
			if body[1] == nil {
				body[1] = defaultPageSize
			}
			body[8] = []any{cursor}
			body[9] = []any{cursor}
		} else {
			if body[3] == nil {
				body[3] = []any{nil, cursor}
			} else {
				body[3] = replaceNumbers(body[3], cursor)
			}
			if body[8] == nil {
				body[8] = []any{cursor}
			} else {
				body[8] = replaceNumbers(body[8], cursor)
			}
			if body[9] == nil {
				body[9] = []any{cursor}
			} else {
				body[9] = replaceNumbers(body[9], cursor)
			}
		}
	}
	if opts.SpaceID != "" {
		body[7] = replaceSpace(body[7], opts.SpaceID)
	}
	if body[4] == nil {
		body[4] = []any{3, 1, 4}
	}
	if body[5] == nil {
		body[5] = 1000
	}
	if body[6] == nil {
		body[6] = 20
	}
	if body[10] == nil {
		body[10] = 2
	}
	return marshalBody(body)
}

func HeartbeatBody(raw string) (string, bool) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", false
	}
	decoder := json.NewDecoder(bytes.NewReader([]byte(raw)))
	decoder.UseNumber()
	var body []any
	if err := decoder.Decode(&body); err != nil || len(body) < 100 || body[99] == nil {
		return "", false
	}
	encoded, err := marshalBody([]any{body[99], []any{1, 1}})
	if err != nil {
		return "", false
	}
	return encoded, true
}

func OldestCursor(page Page) int64 {
	var oldest int64
	for _, topic := range page.Topics {
		n, err := strconv.ParseInt(topic.ID, 10, 64)
		if err != nil {
			continue
		}
		if oldest == 0 || n < oldest {
			oldest = n
		}
	}
	return oldest
}

func replaceNumbers(value any, cursor int64) any {
	switch typed := value.(type) {
	case json.Number, float64, int, int32, int64:
		return cursor
	case []any:
		replaced := make([]any, len(typed))
		for i, item := range typed {
			replaced[i] = replaceNumbers(item, cursor)
		}
		return replaced
	default:
		return typed
	}
}

func isTopicSelector(value any) bool {
	switch typed := value.(type) {
	case string:
		if !chatIDPattern.MatchString(typed) {
			return false
		}
		for _, r := range typed {
			if r >= 'A' && r <= 'Z' || r >= 'a' && r <= 'z' {
				return true
			}
		}
		return false
	case []any:
		for _, item := range typed {
			if isTopicSelector(item) {
				return true
			}
		}
		return false
	default:
		return false
	}
}

func marshalBody(body []any) (string, error) {
	var encoded bytes.Buffer
	encoder := json.NewEncoder(&encoded)
	encoder.SetEscapeHTML(false)
	if err := encoder.Encode(body); err != nil {
		return "", errors.New("could not encode list_topics body")
	}
	return strings.TrimSpace(encoded.String()), nil
}

func ensureLen(body *[]any, n int) {
	if len(*body) >= n {
		return
	}
	grown := make([]any, n)
	copy(grown, *body)
	*body = grown
}

func replaceSpace(value any, spaceID string) any {
	switch typed := value.(type) {
	case string:
		if chatIDPattern.MatchString(typed) {
			return spaceID
		}
		return typed
	case []any:
		replaced := make([]any, len(typed))
		for i, item := range typed {
			replaced[i] = replaceSpace(item, spaceID)
		}
		return replaced
	default:
		if typed == nil {
			return []any{[]any{spaceID}}
		}
		return typed
	}
}
