package capture

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Alechan/ai-resources/tools/gchatctl/src/internal/auth"
)

func TestSelectMatchingPrefersNeedlePath(t *testing.T) {
	// Given
	requests := []Request{
		{URLPath: "/u/ACCOUNT/api/list_topics", RequestHost: "chat.google.com"},
		{URLPath: "/u/ACCOUNT/api/paginated_world", RequestHost: "chat.google.com", Body: auth.NewSecret("[1]")},
	}

	// When
	got, err := SelectMatching(requests, "paginated_world")

	// Then
	if err != nil {
		t.Fatal(err)
	}
	if got.URLPath != "/u/ACCOUNT/api/paginated_world" || got.Body.Reveal() != "[1]" {
		t.Fatalf("got = %#v", got)
	}
}

func TestSelectMatchingSearchMessagesUsesBatchexecute(t *testing.T) {
	// Given
	requests := []Request{
		{URLPath: "/u/ACCOUNT/api/list_topics", RequestHost: "chat.google.com"},
		{URLPath: "/_/DynamiteWebUi/data/batchexecute", RawQuery: "rpcids=SBNmJb&source-path=/app/search", RequestHost: "chat.google.com", Body: auth.NewSecret("f.req=1")},
	}

	// When
	got, err := SelectMatching(requests, "search.messages")

	// Then
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(got.URLPath, "batchexecute") || got.Body.Reveal() != "f.req=1" {
		t.Fatalf("got = %#v", got)
	}
}

func TestAllowedHost(t *testing.T) {
	tests := []struct {
		host string
		want bool
	}{
		{"chat.google.com", true},
		{"clients6.google.com", true},
		{"chat.googleapis.com", true},
		{"cloudsearch.clients6.google.com", true},
		{"peoplestack-pa.clients6.google.com", true},
		{"mail.google.com", false},
		{"fonts.googleapis.com", false},
		{"slack.com", false},
		{"example.com", false},
	}
	for _, tc := range tests {
		t.Run(tc.host, func(t *testing.T) {
			if got := AllowedHost(tc.host); got != tc.want {
				t.Fatalf("AllowedHost(%q) = %t, want %t", tc.host, got, tc.want)
			}
		})
	}
}

func TestAccountKey(t *testing.T) {
	tests := []struct {
		name     string
		rawURL   string
		header   string
		explicit string
		want     string
	}{
		{"explicit wins", "https://chat.google.com/u/0/", "1", "2", "2"},
		{"path authuser", "https://chat.google.com/u/1/api/foo", "", "", "1"},
		{"header fallback", "https://clients6.google.com/chat/v1/foo", "3", "", "3"},
		{"default", "https://chat.googleapis.com/v1/spaces", "", "", "0"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := AccountKey(tc.rawURL, tc.header, tc.explicit); got != tc.want {
				t.Fatalf("AccountKey = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestNormalizePathKeepsAPINames(t *testing.T) {
	tests := []struct {
		rawURL string
		want   string
	}{
		{"https://chat.google.com/u/0/api/get_group_scoped_capabilities", "/u/ACCOUNT/api/get_group_scoped_capabilities"},
		{"https://chat.google.com/u/0/api/list_topics", "/u/ACCOUNT/api/list_topics"},
		{"https://clients6.google.com/chat/v1/spaces/AAAAlongIdentifierValue12345/messages", "/chat/v1/spaces/REDACTED_ID/messages"},
	}
	for _, tc := range tests {
		if got := NormalizePath(tc.rawURL); got != tc.want {
			t.Fatalf("NormalizePath(%q) = %q, want %q", tc.rawURL, got, tc.want)
		}
	}
}

func TestParseCURLSession(t *testing.T) {
	cookie := "SID=" + "generated-cookie-" + t.Name()
	authz := "SAPISIDHASH generated-auth-" + t.Name()
	input := "curl 'https://chat.google.com/u/0/api/get_user_presence?rt=c' -H 'cookie: " + cookie + "' -H 'authorization: " + authz + "' -H 'x-goog-authuser: 0' --data-raw '{}'"

	// When
	got, err := Parse(input)

	// Then
	if err != nil {
		t.Fatal(err)
	}
	if got.AccountKey != "0" || got.RequestHost != "chat.google.com" || got.Method != "POST" {
		t.Fatalf("request = %#v", got)
	}
	if got.Cookie.Reveal() != cookie || got.Authorization.Reveal() != authz {
		t.Fatal("secrets did not parse")
	}
	if got.URLPath != "/u/ACCOUNT/api/get_user_presence" {
		t.Fatalf("path = %q", got.URLPath)
	}
	if fmt.Sprint(got.QueryNames) != "[rt]" {
		t.Fatalf("query = %v", got.QueryNames)
	}
	if !got.HasBody || got.Body.Reveal() != "{}" {
		t.Fatalf("body = %q", got.Body.Reveal())
	}
}

func TestParseHARPrefersChatHostAndRedactsFixture(t *testing.T) {
	cookie := "SID=" + "generated-cookie-" + t.Name()
	authz := "SAPISIDHASH generated-auth-" + t.Name()
	har := fmt.Sprintf(`{
  "log": {
    "entries": [
      {
        "request": {
          "method": "GET",
          "url": "https://mail.google.com/mail/",
          "headers": [{"name": "Cookie", "value": %q}]
        },
        "response": {"status": 200}
      },
      {
        "request": {
          "method": "POST",
          "url": "https://clients6.google.com/chat/v1/spaces/AAAAlongIdentifierValue12345/messages?pageSize=50",
          "headers": [
            {"name": "Cookie", "value": %q},
            {"name": "Authorization", "value": %q}
          ],
          "queryString": [{"name": "pageSize", "value": "50"}],
          "postData": {"text": "{\"text\":\"secret-message-body\"}"}
        },
        "response": {"status": 200}
      },
      {
        "request": {
          "method": "GET",
          "url": "https://chat.google.com/u/0/api/get_spaces",
          "headers": [
            {"name": "Cookie", "value": %q},
            {"name": "X-Goog-AuthUser", "value": "0"}
          ]
        },
        "response": {"status": 200}
      }
    ]
  }
}`, cookie, cookie, authz, cookie)

	// When
	selected, err := Parse(har)
	if err != nil {
		t.Fatal(err)
	}
	requests, err := ParseAll(har)
	if err != nil {
		t.Fatal(err)
	}
	fixture := NewFixture(requests, selected, "bootstrap")

	// Then
	if selected.RequestHost != "chat.google.com" || selected.URLPath != "/u/ACCOUNT/api/get_spaces" {
		t.Fatalf("selected = %#v", selected)
	}
	if len(requests) != 2 {
		t.Fatalf("requests = %d", len(requests))
	}
	data, err := json.Marshal(fixture)
	if err != nil {
		t.Fatal(err)
	}
	text := string(data)
	for _, secret := range []string{cookie, authz, "secret-message-body", "AAAAlongIdentifierValue12345"} {
		if strings.Contains(text, secret) {
			t.Fatalf("fixture leaked %q: %s", secret, text)
		}
	}
	if !strings.Contains(text, "REDACTED_ID") || !strings.Contains(text, `"has_cookie":true`) {
		t.Fatalf("fixture missing expected fields: %s", text)
	}
}

func TestIsStaticPath(t *testing.T) {
	tests := []struct {
		path string
		want bool
	}{
		{"/u/0/api/get_spaces", false},
		{"/chat/v1/spaces/foo/messages", false},
		{"/u/0/boq-chat/_/js/k=script.main.js", true},
		{"/static/foo.png", true},
		{"/_/scs/boq-chat/_/js/k=chat.main.esd.O/am=AAA/d=1/rs=AAA/m=base", true},
		{"/foo.css", true},
	}
	for _, tc := range tests {
		if got := IsStaticPath(tc.path); got != tc.want {
			t.Fatalf("IsStaticPath(%q) = %t, want %t", tc.path, got, tc.want)
		}
	}
}

func TestParseHARSkipsStaticAndDedupesFixture(t *testing.T) {
	cookie := "SID=" + "generated-cookie-" + t.Name()
	har := fmt.Sprintf(`{
  "log": {
    "version": "1.2",
    "creator": {"name": "Chrome", "version": "1"},
    "entries": [
      {
        "request": {
          "method": "GET",
          "url": "https://chat.google.com/u/0/static/app.js",
          "headers": [{"name": "Cookie", "value": %q}]
        },
        "response": {"status": 200}
      },
      {
        "request": {
          "method": "GET",
          "url": "https://chat.google.com/u/0/api/get_spaces",
          "headers": [{"name": "Cookie", "value": %q}]
        },
        "response": {"status": 200}
      },
      {
        "request": {
          "method": "GET",
          "url": "https://chat.google.com/u/0/api/get_spaces",
          "headers": [{"name": "Cookie", "value": %q}, {"name": "sec-ch-ua", "value": "Chrome"}]
        },
        "response": {"status": 200}
      }
    ]
  }
}`, cookie, cookie, cookie)

	// When
	requests, err := ParseAll(har)
	if err != nil {
		t.Fatal(err)
	}
	selected, err := Select(requests)
	if err != nil {
		t.Fatal(err)
	}
	fixture := NewFixture(requests, selected, "bootstrap")

	// Then
	if len(requests) != 2 {
		t.Fatalf("requests = %d", len(requests))
	}
	if len(fixture.Requests) != 1 || fixture.Requests[0].Count != 2 {
		t.Fatalf("fixture requests = %#v", fixture.Requests)
	}
	if strings.Contains(fmt.Sprint(fixture.Requests[0].HeaderNames), "sec-ch-ua") {
		t.Fatalf("fixture kept noisy headers: %#v", fixture.Requests[0].HeaderNames)
	}
}

func TestParseHARWithoutCookiesStillInventoriesRequests(t *testing.T) {
	har := `{
  "log": {
    "entries": [
      {
        "request": {
          "method": "POST",
          "url": "https://chat.google.com/u/0/api/list_topics",
          "headers": [{"name": "origin", "value": "https://chat.google.com"}],
          "postData": {"text": "{}"}
        },
        "response": {"status": 200}
      }
    ]
  }
}`

	// When
	requests, err := ParseAll(har)

	// Then
	if err != nil {
		t.Fatal(err)
	}
	if len(requests) != 1 || requests[0].URLPath != "/u/ACCOUNT/api/list_topics" || requests[0].Cookie.Reveal() != "" {
		t.Fatalf("requests = %#v", requests)
	}
}

func TestParseRejectsInvalidInputWithoutLeaks(t *testing.T) {
	cookie := "SID=" + "generated-sensitive-cookie-" + t.Name()
	tests := []string{
		"curl https://chat.google.com/u/0/api/foo",
		"curl https://mail.google.com/mail -H 'cookie: " + cookie + "'",
		"curl https://chat.google.com/u/0/api/foo https://clients6.google.com/chat -H 'cookie: " + cookie + "'",
		"curl 'https://chat.google.com/u/0/api/foo -H x",
		`{"log":{"entries":[{"request":{"method":"GET","url":"https://chat.google.com/u/0/api/foo"},"response":{"status":401}}]}}`,
		"{not json",
		"",
	}
	for _, input := range tests {
		_, err := Parse(input)
		if err == nil {
			t.Fatalf("Parse(%q) unexpectedly succeeded", input)
		}
		if strings.Contains(err.Error(), cookie) {
			t.Fatalf("error leaked a secret: %v", err)
		}
	}
}

func TestWriteFixtureOmitsSecrets(t *testing.T) {
	cookie := "SID=" + "generated-cookie-" + t.Name()
	request := Request{
		AccountKey:    "0",
		RequestHost:   "chat.google.com",
		Method:        "GET",
		URLPath:       "/u/ACCOUNT/api/get_spaces",
		Cookie:        auth.NewSecret(cookie),
		Authorization: auth.NewSecret("generated-auth"),
		HasBody:       false,
		Status:        200,
	}
	path := filepath.Join(t.TempDir(), "fixture.json")

	// When
	if err := WriteFixture(path, NewFixture([]Request{request}, request, "bootstrap")); err != nil {
		t.Fatal(err)
	}

	// Then
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), cookie) || strings.Contains(string(data), "generated-auth") {
		t.Fatalf("wrote secrets: %s", data)
	}
}
