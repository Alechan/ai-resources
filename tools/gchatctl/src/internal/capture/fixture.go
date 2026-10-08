package capture

import (
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
)

type Fixture struct {
	SchemaVersion int              `json:"schema_version"`
	Operation     string           `json:"operation,omitempty"`
	AccountKey    string           `json:"account_key"`
	Selected      FixtureRequest   `json:"selected"`
	Requests      []FixtureRequest `json:"requests"`
}

type FixtureRequest struct {
	RequestHost      string   `json:"request_host"`
	Method           string   `json:"method"`
	URLPath          string   `json:"url_path"`
	QueryNames       []string `json:"query_names,omitempty"`
	HeaderNames      []string `json:"header_names,omitempty"`
	HasCookie        bool     `json:"has_cookie"`
	HasAuthorization bool     `json:"has_authorization"`
	HasBody          bool     `json:"has_body"`
	Status           int      `json:"status,omitempty"`
	Count            int      `json:"count"`
}

func NewFixture(requests []Request, selected Request, operation string) Fixture {
	counts := map[string]FixtureRequest{}
	order := make([]string, 0, len(requests))
	for _, request := range requests {
		item := toFixtureRequest(request)
		key := fixtureKey(item)
		if existing, ok := counts[key]; ok {
			existing.Count++
			counts[key] = existing
			continue
		}
		item.Count = 1
		counts[key] = item
		order = append(order, key)
	}
	items := make([]FixtureRequest, 0, len(order))
	for _, key := range order {
		items = append(items, counts[key])
	}
	slices.SortFunc(items, func(a, b FixtureRequest) int {
		if a.Count != b.Count {
			return b.Count - a.Count
		}
		if a.RequestHost != b.RequestHost {
			return strings.Compare(a.RequestHost, b.RequestHost)
		}
		if a.Method != b.Method {
			return strings.Compare(a.Method, b.Method)
		}
		return strings.Compare(a.URLPath, b.URLPath)
	})
	selectedItem := toFixtureRequest(selected)
	selectedItem.Count = 1
	if existing, ok := counts[fixtureKey(selectedItem)]; ok {
		selectedItem.Count = existing.Count
	}
	return Fixture{
		SchemaVersion: 1,
		Operation:     operation,
		AccountKey:    selected.AccountKey,
		Selected:      selectedItem,
		Requests:      items,
	}
}

func WriteFixture(path string, fixture Fixture) error {
	data, err := json.MarshalIndent(fixture, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o600)
}

func toFixtureRequest(request Request) FixtureRequest {
	return FixtureRequest{
		RequestHost:      request.RequestHost,
		Method:           request.Method,
		URLPath:          request.URLPath,
		QueryNames:       slices.Clone(request.QueryNames),
		HeaderNames:      filterHeaderNames(request.HeaderNames),
		HasCookie:        request.Cookie.Reveal() != "",
		HasAuthorization: request.Authorization.Reveal() != "",
		HasBody:          request.HasBody,
		Status:           request.Status,
		Count:            1,
	}
}

func filterHeaderNames(names []string) []string {
	filtered := make([]string, 0, len(names))
	for _, name := range names {
		if name == "cookie" || extraHeaderAllowlist[name] {
			filtered = append(filtered, name)
		}
	}
	slices.Sort(filtered)
	return slices.Compact(filtered)
}

func fixtureKey(request FixtureRequest) string {
	return strings.Join([]string{
		request.RequestHost,
		request.Method,
		request.URLPath,
		strings.Join(request.QueryNames, ","),
		strconv.FormatBool(request.HasBody),
	}, "|")
}
