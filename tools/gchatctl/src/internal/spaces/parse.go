package spaces

import (
	"bytes"
	"encoding/json"
	"errors"
	"regexp"
	"unicode"

	"github.com/Alechan/ai-resources/tools/gchatctl/src/internal/topics"
)

var spaceIDPattern = regexp.MustCompile(`^[A-Za-z0-9_-]{11}$`)

type World struct {
	RPC    string  `json:"rpc,omitempty"`
	Spaces []Space `json:"spaces"`
}

type Space struct {
	ID   string `json:"id"`
	Name string `json:"name,omitempty"`
}

type Group struct {
	RPC string `json:"rpc,omitempty"`
	Space
}

func ParseGroup(body []byte) (Group, error) {
	trimmed := topics.StripXSSI(body)
	if len(trimmed) == 0 {
		return Group{}, errors.New("empty Google Chat response")
	}
	decoder := json.NewDecoder(bytes.NewReader(trimmed))
	decoder.UseNumber()
	var root any
	if err := decoder.Decode(&root); err != nil {
		return Group{}, errors.New("malformed Google Chat response")
	}
	envelope, ok := asList(root)
	if !ok || len(envelope) == 0 {
		return Group{}, errors.New("malformed Google Chat response")
	}
	inner, ok := asList(envelope[0])
	if !ok || len(inner) == 0 {
		return Group{}, errors.New("malformed Google Chat response")
	}
	group := Group{}
	group.RPC, _ = inner[0].(string)
	if len(inner) > 1 {
		if record, ok := asList(inner[1]); ok {
			if space, ok := parseGroupRecord(record); ok {
				group.Space = space
			}
		}
	}
	if group.ID == "" {
		return Group{}, errors.New("malformed Google Chat space record")
	}
	return group, nil
}

func parseGroupRecord(record []any) (Space, bool) {
	if len(record) == 0 {
		return Space{}, false
	}
	id := firstSpaceID(record[0])
	if id == "" {
		id = firstSpaceID(record)
	}
	if id == "" {
		return Space{}, false
	}
	space := Space{ID: id}
	if len(record) > 1 {
		if name, ok := record[1].(string); ok && looksDisplayName(name) {
			space.Name = name
		}
	}
	return space, true
}

func firstSpaceID(value any) string {
	if id, ok := value.(string); ok {
		if isSpaceID(id) {
			return id
		}
		return ""
	}
	list, ok := asList(value)
	if !ok {
		return ""
	}
	for _, item := range list {
		if id := firstSpaceID(item); id != "" {
			return id
		}
	}
	return ""
}

func looksDisplayName(value string) bool {
	if value == "" || isSpaceID(value) {
		return false
	}
	if len(value) < 2 || len(value) > 80 {
		return false
	}
	letters := 0
	hasSpace := false
	hasDot := false
	for _, r := range value {
		switch {
		case unicode.IsLetter(r):
			letters++
		case r == ' ':
			hasSpace = true
		case r == '.':
			hasDot = true
		case r == '@' || r == '<' || r == '>':
			return false
		}
	}
	if letters < 2 {
		return false
	}
	if hasDot && !hasSpace {
		return false
	}
	return true
}

func Parse(body []byte) (World, error) {
	trimmed := topics.StripXSSI(body)
	if len(trimmed) == 0 {
		return World{}, errors.New("empty Google Chat response")
	}
	decoder := json.NewDecoder(bytes.NewReader(trimmed))
	decoder.UseNumber()
	var root any
	if err := decoder.Decode(&root); err != nil {
		return World{}, errors.New("malformed Google Chat response")
	}
	world := World{Spaces: []Space{}}
	if envelope, ok := asList(root); ok && len(envelope) > 0 {
		if inner, ok := asList(envelope[0]); ok && len(inner) > 0 {
			world.RPC, _ = inner[0].(string)
		}
	}
	seen := map[string]int{}
	for _, item := range findSpaceItems(root) {
		space, ok := parseSpace(item)
		if !ok {
			continue
		}
		if index, exists := seen[space.ID]; exists {
			if space.Name != "" {
				world.Spaces[index] = space
			}
			continue
		}
		seen[space.ID] = len(world.Spaces)
		world.Spaces = append(world.Spaces, space)
	}
	return world, nil
}

func findSpaceItems(value any) []any {
	list, ok := asList(value)
	if !ok {
		return nil
	}
	if looksLikeSpaceList(list) {
		return list
	}
	var found []any
	for _, item := range list {
		found = append(found, findSpaceItems(item)...)
	}
	return found
}

func looksLikeSpaceList(list []any) bool {
	if len(list) == 0 {
		return false
	}
	matched := 0
	for _, item := range list {
		if _, ok := parseSpace(item); ok {
			matched++
		}
	}
	return matched > 0 && matched*2 >= len(list)
}

func parseSpace(value any) (Space, bool) {
	item, ok := asList(value)
	if !ok || len(item) < 1 {
		return Space{}, false
	}
	ident, ok := asList(item[0])
	if !ok || len(ident) < 2 {
		return Space{}, false
	}
	id, _ := ident[1].(string)
	if !isSpaceID(id) {
		return Space{}, false
	}
	space := Space{ID: id}
	if len(item) > 9 {
		if name, ok := item[9].(string); ok {
			space.Name = name
		}
	}
	return space, true
}

func isSpaceID(value string) bool {
	if !spaceIDPattern.MatchString(value) {
		return false
	}
	for _, r := range value {
		if unicode.IsLetter(r) {
			return true
		}
	}
	return false
}

func asList(value any) ([]any, bool) {
	list, ok := value.([]any)
	return list, ok
}
