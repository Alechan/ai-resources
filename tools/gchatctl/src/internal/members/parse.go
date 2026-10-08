package members

import (
	"bytes"
	"encoding/json"
	"errors"
	"unicode"

	"github.com/Alechan/ai-resources/tools/gchatctl/src/internal/topics"
)

type Roster struct {
	RPC     string   `json:"rpc,omitempty"`
	Members []Member `json:"members"`
}

type Member struct {
	ID      string `json:"id"`
	SpaceID string `json:"space_id,omitempty"`
	Name    string `json:"name,omitempty"`
	Email   string `json:"email,omitempty"`
}

func Parse(body []byte) (Roster, error) {
	trimmed := topics.StripXSSI(body)
	if len(trimmed) == 0 {
		return Roster{}, errors.New("empty Google Chat response")
	}
	decoder := json.NewDecoder(bytes.NewReader(trimmed))
	decoder.UseNumber()
	var root any
	if err := decoder.Decode(&root); err != nil {
		return Roster{}, errors.New("malformed Google Chat response")
	}
	inner := rpcInner(root)
	if inner == nil {
		return Roster{}, errors.New("malformed Google Chat response")
	}
	roster := Roster{Members: []Member{}}
	if rpc, ok := inner[0].(string); ok {
		roster.RPC = rpc
	}
	names := map[string]string{}
	collectNames(inner, names)
	seen := map[string]int{}
	for _, item := range memberRows(inner) {
		member, ok := parseMember(item)
		if !ok {
			continue
		}
		if name := names[member.ID]; name != "" {
			member.Name = name
		}
		if index, exists := seen[member.ID]; exists {
			if member.Name != "" {
				roster.Members[index] = member
			}
			continue
		}
		seen[member.ID] = len(roster.Members)
		roster.Members = append(roster.Members, member)
	}
	return roster, nil
}

func rpcInner(root any) []any {
	envelope, ok := asList(root)
	if !ok || len(envelope) == 0 {
		return nil
	}
	inner, ok := asList(envelope[0])
	if !ok || len(inner) == 0 {
		return nil
	}
	return inner
}

func memberRows(inner []any) []any {
	if len(inner) < 2 {
		return nil
	}
	rows, ok := asList(inner[1])
	if !ok {
		return nil
	}
	if looksLikeMemberList(rows) {
		return rows
	}
	for _, item := range rows {
		if nested, ok := asList(item); ok && looksLikeMemberList(nested) {
			return nested
		}
	}
	return rows
}

func looksLikeMemberList(list []any) bool {
	if len(list) == 0 {
		return false
	}
	matched := 0
	for _, item := range list {
		if _, ok := parseMember(item); ok {
			matched++
		}
	}
	return matched > 0 && matched*2 >= len(list)
}

func parseMember(value any) (Member, bool) {
	userID := firstUserID(value)
	if userID == "" {
		return Member{}, false
	}
	return Member{ID: userID, SpaceID: firstSpaceID(value)}, true
}

func collectNames(value any, names map[string]string) {
	list, ok := asList(value)
	if !ok {
		return
	}
	if len(list) >= 2 {
		if name, ok := list[0].(string); ok && looksDisplayName(name) {
			if id := firstUserID(list[1]); id != "" && names[id] == "" {
				names[id] = name
			}
		}
	}
	for _, item := range list {
		collectNames(item, names)
	}
}

func firstUserID(value any) string {
	switch typed := value.(type) {
	case string:
		if isUserID(typed) {
			return typed
		}
	case []any:
		for _, item := range typed {
			if id := firstUserID(item); id != "" {
				return id
			}
		}
	}
	return ""
}

func firstSpaceID(value any) string {
	switch typed := value.(type) {
	case string:
		if isSpaceID(typed) {
			return typed
		}
	case []any:
		for _, item := range typed {
			if id := firstSpaceID(item); id != "" {
				return id
			}
		}
	}
	return ""
}

func looksDisplayName(value string) bool {
	if len(value) < 2 || len(value) > 80 || isUserID(value) || isSpaceID(value) {
		return false
	}
	if stringsContainsDotNoSpace(value) {
		return false
	}
	letters := 0
	for _, r := range value {
		if unicode.IsLetter(r) {
			letters++
		}
		if r == '@' || r == '<' || r == '>' {
			return false
		}
	}
	return letters >= 2
}

func stringsContainsDotNoSpace(value string) bool {
	hasDot := false
	for _, r := range value {
		if r == '.' {
			hasDot = true
		}
		if r == ' ' {
			return false
		}
	}
	return hasDot
}

func isUserID(value string) bool {
	if len(value) < 18 || len(value) > 24 {
		return false
	}
	for _, r := range value {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
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
