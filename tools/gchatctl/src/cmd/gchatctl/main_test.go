package main

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Alechan/ai-resources/tools/gchatctl/src/internal/auth"
)

func testApp() (*application, *bytes.Buffer, *bytes.Buffer) {
	out, errOut := &bytes.Buffer{}, &bytes.Buffer{}
	return &application{
		store:  auth.NewMemoryStore(),
		stdin:  strings.NewReader(""),
		stdout: out,
		stderr: errOut,
		getenv: func(string) string { return "" },
	}, out, errOut
}

func syntheticCURL(t *testing.T) string {
	t.Helper()
	cookie := "SID=" + "generated-cookie-" + t.Name()
	return "curl 'https://chat.google.com/u/0/api/get_spaces' -H 'cookie: " + cookie + "'"
}

func TestHelpListsCommandsAndFlags(t *testing.T) {
	app, out, _ := testApp()
	if code := app.run(t.Context(), []string{"--help"}); code != exitOK {
		t.Fatalf("exit=%d", code)
	}
	for _, expected := range []string{
		"init",
		"doctor",
		"learn",
		"fixture",
		"topics list",
		"spaces list",
		"spaces get",
		"members list",
		"search messages",
		"conversation export",
		"--account",
		"--har-file",
		"--curl-file",
		"Do not paste",
	} {
		if !strings.Contains(out.String(), expected) {
			t.Errorf("help missing %q:\n%s", expected, out.String())
		}
	}
}

func TestInitAndDoctorTextAndJSON(t *testing.T) {
	app, out, errOut := testApp()
	cookie := "SID=" + "generated-cookie-" + t.Name()
	app.stdin = strings.NewReader("curl 'https://chat.google.com/u/0/api/get_spaces' -H 'cookie: " + cookie + "'")

	// When
	if code := app.run(t.Context(), []string{"init"}); code != exitOK {
		t.Fatalf("init exit=%d stderr=%s", code, errOut.String())
	}
	out.Reset()
	if code := app.run(t.Context(), []string{"doctor"}); code != exitOK {
		t.Fatalf("doctor exit=%d stderr=%s", code, errOut.String())
	}

	// Then
	if !strings.Contains(out.String(), "credentials found: true") || strings.Contains(out.String(), cookie) {
		t.Fatalf("unsafe doctor output: %s", out.String())
	}
	if !strings.Contains(out.String(), "live check: not_run") || !strings.Contains(out.String(), "bootstrap") {
		t.Fatalf("doctor missing phase 0 fields: %s", out.String())
	}
	out.Reset()
	if code := app.run(t.Context(), []string{"doctor", "--json"}); code != exitOK || !strings.Contains(out.String(), `"cookie_present": true`) {
		t.Fatalf("JSON doctor exit=%d output=%s", code, out.String())
	}
}

func TestLearnPaginatedWorldKeepsListTopicsBody(t *testing.T) {
	app, out, errOut := testApp()
	app.stdin = strings.NewReader(listTopicsCURL(t))
	if code := app.run(t.Context(), []string{"init"}); code != exitOK {
		t.Fatalf("init exit=%d stderr=%s", code, errOut.String())
	}
	path := filepath.Join(t.TempDir(), "mixed.har")
	har := `{
  "log": {
    "entries": [
      {
        "request": {
          "method": "POST",
          "url": "https://chat.google.com/u/0/api/list_topics",
          "postData": {"text": "[null,99]"}
        },
        "response": {"status": 200}
      },
      {
        "request": {
          "method": "POST",
          "url": "https://chat.google.com/u/0/api/paginated_world",
          "postData": {"text": "[[\"world-body\"]]"}
        },
        "response": {"status": 200}
      }
    ]
  }
}`
	if err := os.WriteFile(path, []byte(har), 0o600); err != nil {
		t.Fatal(err)
	}

	// When
	out.Reset()
	if code := app.run(t.Context(), []string{"learn", "paginated_world", "--har-file", path}); code != exitOK {
		t.Fatalf("learn exit=%d stderr=%s", code, errOut.String())
	}

	// Then
	credential, err := app.store.Load(t.Context(), "0")
	if err != nil {
		t.Fatal(err)
	}
	if credential.Body.Reveal() != "[null,1]" {
		t.Fatalf("list_topics body replaced: %q", credential.Body.Reveal())
	}
	template, ok := credential.OperationTemplate("paginated_world")
	if !ok || template.Body != `[["world-body"]]` || !strings.Contains(template.URLPath, "paginated_world") {
		t.Fatalf("template = %#v ok=%t", template, ok)
	}
	out.Reset()
	if code := app.run(t.Context(), []string{"doctor"}); code != exitOK {
		t.Fatalf("doctor exit=%d stderr=%s", code, errOut.String())
	}
	text := out.String()
	if !strings.Contains(text, "list_topics ready: true") || !strings.Contains(text, "paginated_world ready: true") {
		t.Fatalf("doctor = %s", text)
	}
	if strings.Contains(text, "world-body") || strings.Contains(text, "generated-cookie") {
		t.Fatalf("doctor leaked capture: %s", text)
	}
}

func TestSpacesListWritesCountsAndFiles(t *testing.T) {
	// Given
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path != "/u/0/api/paginated_world" {
			t.Errorf("path = %s", request.URL.Path)
		}
		body, _ := io.ReadAll(request.Body)
		if string(body) != `[["world-body"]]` {
			t.Errorf("body = %q", body)
		}
		writer.Write([]byte(`)]}'

[["dfe.pw.pw",[[
  [[null,"spaceidxxx1"],null,"1","2",null,null,null,null,null,"Space One Name",null,null,null,"spaceidxxx1"],
  [[null,"spaceidxxx2"],null,"1","2",null,null,null,null,null,"Space Two Name",null,null,null,"spaceidxxx2"]
]]]]`))
	}))
	t.Cleanup(server.Close)
	app, out, errOut := testApp()
	app.httpClient = server.Client()
	app.stdin = strings.NewReader(listTopicsCURL(t))
	if code := app.run(t.Context(), []string{"init"}); code != exitOK {
		t.Fatalf("init exit=%d stderr=%s", code, errOut.String())
	}
	credential, err := app.store.Load(t.Context(), "0")
	if err != nil {
		t.Fatal(err)
	}
	credential.RequestHost = strings.TrimPrefix(server.URL, "http://")
	credential.UpsertTemplate(auth.Template{
		Operation: "paginated_world",
		Method:    "POST",
		URLPath:   "/u/ACCOUNT/api/paginated_world",
		HasBody:   true,
		Body:      `[["world-body"]]`,
	})
	if err := app.store.Save(t.Context(), credential); err != nil {
		t.Fatal(err)
	}
	directory := filepath.Join(t.TempDir(), "spaces-export")

	// When
	out.Reset()
	if code := app.run(t.Context(), []string{"spaces", "list", "--output", directory}); code != exitOK {
		t.Fatalf("spaces list exit=%d stderr=%s", code, errOut.String())
	}

	// Then
	text := out.String()
	if !strings.Contains(text, "listed spaces: 2") || !strings.Contains(text, "space: spaceidxxx1") || !strings.Contains(text, "space: spaceidxxx2") {
		t.Fatalf("summary = %s", text)
	}
	if strings.Contains(text, "Space One Name") || strings.Contains(text, "Space Two Name") {
		t.Fatalf("summary quoted space names: %s", text)
	}
	markdown, err := os.ReadFile(filepath.Join(directory, "spaces.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(markdown), "Space One Name") {
		t.Fatalf("markdown = %s", markdown)
	}
}

func TestSpacesListRequiresLearnedWorld(t *testing.T) {
	app, _, errOut := testApp()
	app.stdin = strings.NewReader(listTopicsCURL(t))
	if code := app.run(t.Context(), []string{"init"}); code != exitOK {
		t.Fatalf("init exit=%d stderr=%s", code, errOut.String())
	}
	if code := app.run(t.Context(), []string{"spaces", "list", "--stdout", "--format", "json"}); code == exitOK || !strings.Contains(errOut.String(), "learn paginated_world") {
		t.Fatalf("exit=%d stderr=%s", code, errOut.String())
	}
}

func TestMembersListWritesCountsAndFiles(t *testing.T) {
	// Given
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path != "/u/0/api/list_members" {
			t.Errorf("path = %s", request.URL.Path)
		}
		if request.Header.Get("x-goog-chat-space-id") != "spaceidxxx1" {
			t.Errorf("space header = %s", request.Header.Get("x-goog-chat-space-id"))
		}
		body, _ := io.ReadAll(request.Body)
		if strings.Contains(string(body), "oldspaceid1") || !strings.Contains(string(body), "spaceidxxx1") {
			t.Errorf("body = %s", body)
		}
		writer.Write(cliMembersResponse())
	}))
	t.Cleanup(server.Close)
	app, out, errOut := testApp()
	app.httpClient = server.Client()
	app.stdin = strings.NewReader(listTopicsCURL(t))
	if code := app.run(t.Context(), []string{"init"}); code != exitOK {
		t.Fatalf("init exit=%d stderr=%s", code, errOut.String())
	}
	credential, err := app.store.Load(t.Context(), "0")
	if err != nil {
		t.Fatal(err)
	}
	credential.RequestHost = strings.TrimPrefix(server.URL, "http://")
	credential.UpsertTemplate(auth.Template{
		Operation: "list_members",
		Method:    "POST",
		URLPath:   "/u/ACCOUNT/api/list_members",
		HasBody:   true,
		Body:      cliHundredSlotJSON(4, []any{[]any{"oldspaceid1"}}),
	})
	if err := app.store.Save(t.Context(), credential); err != nil {
		t.Fatal(err)
	}
	directory := filepath.Join(t.TempDir(), "members-export")

	// When
	out.Reset()
	if code := app.run(t.Context(), []string{"members", "list", "--space", "spaceidxxx1", "--output", directory}); code != exitOK {
		t.Fatalf("members list exit=%d stderr=%s", code, errOut.String())
	}

	// Then
	text := out.String()
	if !strings.Contains(text, "listed members: 1") || !strings.Contains(text, "member: 123456789012345678901") {
		t.Fatalf("summary = %s", text)
	}
	if strings.Contains(text, "Member One Name") {
		t.Fatalf("summary quoted member name: %s", text)
	}
	markdown, err := os.ReadFile(filepath.Join(directory, "members.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(markdown), "Member One Name") {
		t.Fatalf("markdown = %s", markdown)
	}
}

func TestMembersListRequiresLearnedCapture(t *testing.T) {
	app, _, errOut := testApp()
	app.stdin = strings.NewReader(listTopicsCURL(t))
	if code := app.run(t.Context(), []string{"init"}); code != exitOK {
		t.Fatalf("init exit=%d stderr=%s", code, errOut.String())
	}
	if code := app.run(t.Context(), []string{"members", "list", "--space", "spaceidxxx1", "--stdout", "--format", "json"}); code == exitOK || !strings.Contains(errOut.String(), "learn list_members") {
		t.Fatalf("exit=%d stderr=%s", code, errOut.String())
	}
}

func TestSpacesGetWritesIDAndFile(t *testing.T) {
	// Given
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path != "/u/0/api/get_group" {
			t.Errorf("path = %s", request.URL.Path)
		}
		if request.Header.Get("x-goog-chat-space-id") != "spaceidxxx1" {
			t.Errorf("space header = %s", request.Header.Get("x-goog-chat-space-id"))
		}
		writer.Write(cliGroupResponse())
	}))
	t.Cleanup(server.Close)
	app, out, errOut := testApp()
	app.httpClient = server.Client()
	app.stdin = strings.NewReader(listTopicsCURL(t))
	if code := app.run(t.Context(), []string{"init"}); code != exitOK {
		t.Fatalf("init exit=%d stderr=%s", code, errOut.String())
	}
	credential, err := app.store.Load(t.Context(), "0")
	if err != nil {
		t.Fatal(err)
	}
	credential.RequestHost = strings.TrimPrefix(server.URL, "http://")
	credential.UpsertTemplate(auth.Template{
		Operation: "get_group",
		Method:    "POST",
		URLPath:   "/u/ACCOUNT/api/get_group",
		HasBody:   true,
		Body:      cliHundredSlotJSON(0, []any{[]any{[]any{"oldspaceid1"}}}),
	})
	if err := app.store.Save(t.Context(), credential); err != nil {
		t.Fatal(err)
	}
	directory := filepath.Join(t.TempDir(), "space-get")

	// When
	out.Reset()
	if code := app.run(t.Context(), []string{"spaces", "get", "spaceidxxx1", "--output", directory}); code != exitOK {
		t.Fatalf("spaces get exit=%d stderr=%s", code, errOut.String())
	}

	// Then
	text := out.String()
	if !strings.Contains(text, "listed spaces: 1") || !strings.Contains(text, "space: spaceidxxx1") {
		t.Fatalf("summary = %s", text)
	}
	if strings.Contains(text, "Space One Name") {
		t.Fatalf("summary quoted space name: %s", text)
	}
	markdown, err := os.ReadFile(filepath.Join(directory, "spaces.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(markdown), "Space One Name") {
		t.Fatalf("markdown = %s", markdown)
	}
}

func TestSpacesGetRequiresLearnedCapture(t *testing.T) {
	app, _, errOut := testApp()
	app.stdin = strings.NewReader(listTopicsCURL(t))
	if code := app.run(t.Context(), []string{"init"}); code != exitOK {
		t.Fatalf("init exit=%d stderr=%s", code, errOut.String())
	}
	if code := app.run(t.Context(), []string{"spaces", "get", "spaceidxxx1", "--stdout", "--format", "json"}); code == exitOK || !strings.Contains(errOut.String(), "learn get_group") {
		t.Fatalf("exit=%d stderr=%s", code, errOut.String())
	}
}

func TestSearchMessagesWritesCountsAndFiles(t *testing.T) {
	// Given
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path != "/_/DynamiteWebUi/data/batchexecute" {
			t.Errorf("path = %s", request.URL.Path)
		}
		body, _ := io.ReadAll(request.Body)
		if strings.Contains(string(body), "nps") || !strings.Contains(string(body), "hello") {
			t.Errorf("body = %q", body)
		}
		writer.Write(cliSearchResponse())
	}))
	t.Cleanup(server.Close)
	app, out, errOut := testApp()
	app.httpClient = server.Client()
	app.stdin = strings.NewReader(listTopicsCURL(t))
	if code := app.run(t.Context(), []string{"init"}); code != exitOK {
		t.Fatalf("init exit=%d stderr=%s", code, errOut.String())
	}
	credential, err := app.store.Load(t.Context(), "0")
	if err != nil {
		t.Fatal(err)
	}
	credential.RequestHost = strings.TrimPrefix(server.URL, "http://")
	credential.UpsertTemplate(auth.Template{
		Operation:   "search.messages",
		Method:      "POST",
		URLPath:     "/_/DynamiteWebUi/data/batchexecute",
		RawQuery:    "rpcids=SBNmJb&source-path=/app/search",
		ContentType: "application/x-www-form-urlencoded",
		HasBody:     true,
		Body:        "f.req=" + url.QueryEscape(`[[["SBNmJb","[null,null,null,\"nps\",null,\"xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx\"]",null,"1"]]]`) + "&at=synthetic-at-token-value",
	})
	if err := app.store.Save(t.Context(), credential); err != nil {
		t.Fatal(err)
	}
	directory := filepath.Join(t.TempDir(), "search-export")

	// When
	out.Reset()
	if code := app.run(t.Context(), []string{"search", "messages", "--query", "hello", "--output", directory}); code != exitOK {
		t.Fatalf("search exit=%d stderr=%s", code, errOut.String())
	}

	// Then
	text := out.String()
	if !strings.Contains(text, "listed hits: 1") || !strings.Contains(text, "query: hello") {
		t.Fatalf("summary = %s", text)
	}
	if strings.Contains(text, "hit-text-01") {
		t.Fatalf("summary quoted hit text: %s", text)
	}
	markdown, err := os.ReadFile(filepath.Join(directory, "search.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(markdown), "hit-text-01") {
		t.Fatalf("markdown = %s", markdown)
	}
}

func TestSearchMessagesRequiresLearnedCapture(t *testing.T) {
	app, _, errOut := testApp()
	app.stdin = strings.NewReader(listTopicsCURL(t))
	if code := app.run(t.Context(), []string{"init"}); code != exitOK {
		t.Fatalf("init exit=%d stderr=%s", code, errOut.String())
	}
	if code := app.run(t.Context(), []string{"search", "messages", "--query", "hello", "--stdout", "--format", "json"}); code == exitOK || !strings.Contains(errOut.String(), "learn search.messages") {
		t.Fatalf("exit=%d stderr=%s", code, errOut.String())
	}
}

func TestLearnAddsOperation(t *testing.T) {
	app, out, errOut := testApp()
	cookie := "SID=" + "generated-cookie-" + t.Name()
	app.stdin = strings.NewReader("curl 'https://chat.google.com/u/0/api/get_spaces' -H 'cookie: " + cookie + "'")
	if code := app.run(t.Context(), []string{"init"}); code != exitOK {
		t.Fatalf("init exit=%d stderr=%s", code, errOut.String())
	}
	app.stdin = strings.NewReader("curl 'https://clients6.google.com/chat/v1/messages:search?q=incident' -H 'cookie: " + cookie + "' --data-raw '{}'")

	// When
	if code := app.run(t.Context(), []string{"learn", "messages.search"}); code != exitOK {
		t.Fatalf("learn exit=%d stderr=%s", code, errOut.String())
	}

	// Then
	if !strings.Contains(out.String(), "messages.search") {
		t.Fatalf("learn output = %s", out.String())
	}
	out.Reset()
	if code := app.run(t.Context(), []string{"doctor", "--json"}); code != exitOK {
		t.Fatalf("doctor exit=%d stderr=%s", code, errOut.String())
	}
	if !strings.Contains(out.String(), `"messages.search"`) || strings.Contains(out.String(), cookie) {
		t.Fatalf("doctor after learn = %s", out.String())
	}
}

func TestFixtureWritesRedactedFile(t *testing.T) {
	app, out, errOut := testApp()
	cookie := "SID=" + "generated-cookie-" + t.Name()
	path := filepath.Join(t.TempDir(), "capture.har")
	har := `{
  "log": {
    "entries": [
      {
        "request": {
          "method": "GET",
          "url": "https://chat.google.com/u/0/api/get_spaces",
          "headers": [{"name": "Cookie", "value": "` + cookie + `"}]
        },
        "response": {"status": 200}
      }
    ]
  }
}`
	if err := os.WriteFile(path, []byte(har), 0o600); err != nil {
		t.Fatal(err)
	}
	fixturePath := filepath.Join(t.TempDir(), "redacted.json")

	// When
	if code := app.run(t.Context(), []string{"fixture", "--har-file", path, "--out", fixturePath, "--operation", "bootstrap"}); code != exitOK {
		t.Fatalf("fixture exit=%d stderr=%s", code, errOut.String())
	}

	// Then
	data, err := os.ReadFile(fixturePath)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), cookie) {
		t.Fatalf("fixture leaked cookie: %s", data)
	}
	var parsed map[string]any
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), fixturePath) {
		t.Fatalf("stdout = %s", out.String())
	}
}

func TestInitRejectsHARWithoutCookies(t *testing.T) {
	app, _, errOut := testApp()
	path := filepath.Join(t.TempDir(), "nocookie.har")
	har := `{
  "log": {
    "entries": [
      {
        "request": {
          "method": "POST",
          "url": "https://chat.google.com/u/0/api/list_topics",
          "headers": [{"name": "origin", "value": "https://chat.google.com"}]
        },
        "response": {"status": 200}
      }
    ]
  }
}`
	if err := os.WriteFile(path, []byte(har), 0o600); err != nil {
		t.Fatal(err)
	}
	code := app.run(t.Context(), []string{"init", "--har-file", path})
	if code == exitOK || !strings.Contains(errOut.String(), "Allow to generate HAR with sensitive data") {
		t.Fatalf("exit=%d stderr=%s", code, errOut.String())
	}
}

func TestInitRejectsConflictingCaptureFlags(t *testing.T) {
	app, _, errOut := testApp()
	code := app.run(t.Context(), []string{"init", "--har-file", "a.har", "--curl-file", "b.txt"})
	if code == exitOK || !strings.Contains(errOut.String(), "mutually exclusive") {
		t.Fatalf("exit=%d stderr=%s", code, errOut.String())
	}
}

func TestClearRemovesCredentials(t *testing.T) {
	app, _, errOut := testApp()
	app.stdin = strings.NewReader(syntheticCURL(t))
	if code := app.run(t.Context(), []string{"init"}); code != exitOK {
		t.Fatalf("init exit=%d stderr=%s", code, errOut.String())
	}
	if code := app.run(t.Context(), []string{"init", "--clear"}); code != exitOK {
		t.Fatalf("clear exit=%d stderr=%s", code, errOut.String())
	}
	if code := app.run(t.Context(), []string{"doctor"}); code == exitOK {
		t.Fatal("doctor succeeded after clear")
	}
}

func TestDoctorLiveReportsHTTPStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		writer.WriteHeader(http.StatusUnauthorized)
	}))
	t.Cleanup(server.Close)
	app, out, errOut := testApp()
	app.httpClient = server.Client()
	cookie := "SID=" + "generated-cookie-" + t.Name()
	app.stdin = strings.NewReader("curl 'https://chat.google.com/u/0/api/list_topics' -H 'cookie: " + cookie + "' -H 'x-framework-xsrf-token: xsrf-token-fixture' --data-raw '[null,1]'")
	if code := app.run(t.Context(), []string{"init"}); code != exitOK {
		t.Fatalf("init exit=%d stderr=%s", code, errOut.String())
	}
	credential, err := app.store.Load(t.Context(), "0")
	if err != nil {
		t.Fatal(err)
	}
	credential.RequestHost = strings.TrimPrefix(server.URL, "http://")
	if err := app.store.Save(t.Context(), credential); err != nil {
		t.Fatal(err)
	}

	// When
	out.Reset()
	if code := app.run(t.Context(), []string{"doctor", "--live"}); code != exitOK {
		t.Fatalf("doctor exit=%d stderr=%s", code, errOut.String())
	}

	// Then
	text := out.String()
	if strings.Contains(text, cookie) || strings.Contains(text, "xsrf-token-fixture") {
		t.Fatalf("leaked secret: %s", text)
	}
	for _, expected := range []string{"live check: http_401", "live cookie: true", "live xsrf: true"} {
		if !strings.Contains(text, expected) {
			t.Fatalf("missing %q in %s", expected, text)
		}
	}
}

func listTopicsCURL(t *testing.T) string {
	t.Helper()
	cookie := "SID=" + "generated-cookie-" + t.Name()
	return "curl 'https://chat.google.com/u/0/api/list_topics?c=1' -H 'cookie: " + cookie + "' -H 'origin: https://chat.google.com' --data-raw '[null,1]'"
}

func TestTopicsListWritesCountsAndFiles(t *testing.T) {
	// Given
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path != "/u/0/api/list_topics" {
			t.Errorf("path = %s", request.URL.Path)
		}
		body, _ := io.ReadAll(request.Body)
		if string(body) != "[null,1]" {
			t.Errorf("body = %q", body)
		}
		writer.Write([]byte(`)]}'

[["dfe.t.lt",[[
  [null,"spaceidxxxx1"],
  "1788269183261582",
  null,null,null,null,
  [[
    [[null,null,null,null],"authoridxxx"],
    "ignored",
    "messageidxxxxxx",
    "messageidxxxxxx",
    null,null,null,null,null,
    "Deployment finished with NPS notes"
  ]]
]]]]`))
	}))
	t.Cleanup(server.Close)
	app, out, errOut := testApp()
	app.httpClient = server.Client()
	app.stdin = strings.NewReader(listTopicsCURL(t))
	if code := app.run(t.Context(), []string{"init"}); code != exitOK {
		t.Fatalf("init exit=%d stderr=%s", code, errOut.String())
	}
	credential, err := app.store.Load(t.Context(), "0")
	if err != nil {
		t.Fatal(err)
	}
	credential.RequestHost = strings.TrimPrefix(server.URL, "http://")
	if err := app.store.Save(t.Context(), credential); err != nil {
		t.Fatal(err)
	}
	directory := filepath.Join(t.TempDir(), "topics-export")

	// When
	out.Reset()
	if code := app.run(t.Context(), []string{"topics", "list", "--contains", "nps", "--output", directory}); code != exitOK {
		t.Fatalf("topics list exit=%d stderr=%s", code, errOut.String())
	}

	// Then
	text := out.String()
	if !strings.Contains(text, "listed topics: 1") || !strings.Contains(text, "listed messages: 1") {
		t.Fatalf("summary = %s", text)
	}
	if strings.Contains(text, "Deployment finished") {
		t.Fatalf("summary quoted private text: %s", text)
	}
	markdown, err := os.ReadFile(filepath.Join(directory, "topics.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(markdown), "Deployment finished with NPS notes") {
		t.Fatalf("markdown = %s", markdown)
	}
}

func TestTopicsListSpaceFetchesRewrittenSpace(t *testing.T) {
	// Given
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.Header.Get("x-goog-chat-space-id") != "spaceidxxxx1" {
			t.Errorf("space header = %s", request.Header.Get("x-goog-chat-space-id"))
		}
		writer.Write([]byte(`)]}'

[["dfe.t.lt",[[
  [null,"spaceidxxxx1"],
  "1788269183261582",
  null,null,null,null,
  [[
    [[null,null,null,null],"authoridxxx"],
    "ignored",
    "messageidxxxxxx",
    "messageidxxxxxx",
    null,null,null,null,null,
    "hello from other space"
  ]]
]]]]`))
	}))
	t.Cleanup(server.Close)
	app, out, errOut := testApp()
	app.httpClient = server.Client()
	app.stdin = strings.NewReader(listTopicsCURL(t))
	if code := app.run(t.Context(), []string{"init"}); code != exitOK {
		t.Fatalf("init exit=%d stderr=%s", code, errOut.String())
	}
	credential, err := app.store.Load(t.Context(), "0")
	if err != nil {
		t.Fatal(err)
	}
	credential.RequestHost = strings.TrimPrefix(server.URL, "http://")
	if err := app.store.Save(t.Context(), credential); err != nil {
		t.Fatal(err)
	}
	directory := filepath.Join(t.TempDir(), "space-topics")

	// When
	out.Reset()
	if code := app.run(t.Context(), []string{"topics", "list", "--space", "spaceidxxxx1", "--output", directory}); code != exitOK {
		t.Fatalf("topics list exit=%d stderr=%s", code, errOut.String())
	}

	// Then
	if !strings.Contains(out.String(), "listed topics: 1") {
		t.Fatalf("summary = %s", out.String())
	}
	if strings.Contains(out.String(), "hello from other space") {
		t.Fatalf("summary quoted private text: %s", out.String())
	}
}

func TestTopicsListRequiresOutputOrStdout(t *testing.T) {
	app, _, errOut := testApp()
	app.stdin = strings.NewReader(listTopicsCURL(t))
	if code := app.run(t.Context(), []string{"init"}); code != exitOK {
		t.Fatalf("init exit=%d stderr=%s", code, errOut.String())
	}
	if code := app.run(t.Context(), []string{"topics", "list"}); code == exitOK || !strings.Contains(errOut.String(), "--output is required") {
		t.Fatalf("exit=%d stderr=%s", code, errOut.String())
	}
}

func TestConversationExportFromPermalink(t *testing.T) {
	// Given
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.Header.Get("x-goog-chat-space-id") != "AAQAP4TTGjI" {
			t.Errorf("space header = %s", request.Header.Get("x-goog-chat-space-id"))
		}
		writer.Write([]byte(`)]}'

[["dfe.t.lt",[
  [
    [null,"AAQAP4TTGjI"],
    "1789000000001000",
    null,null,null,null,
    [[[[null,null,null,null],"authoridxxx"],"ignored","idolderxxxxxxxxx","idolderxxxxxxxxx",null,null,null,null,null,"older",null,null,null,"olderwebid1"]]
  ],
  [
    [null,"AAQAP4TTGjI"],
    "1789000000002000",
    null,null,null,null,
    [[[[null,null,null,null],"authoridxxx"],"ignored","idstartxxxxxxxxx","idstartxxxxxxxxx",null,null,null,null,null,"start",null,null,null,"qMAc4oTc2i8"]]
  ],
  [
    [null,"AAQAP4TTGjI"],
    "1789000000003000",
    null,null,null,null,
    [[[[null,null,null,null],"authoridxxx"],"ignored","idnewerxxxxxxxxx","idnewerxxxxxxxxx",null,null,null,null,null,"newer",null,null,null,"newerwebid01"]]
  ]
],[],[],true,true]]`))
	}))
	t.Cleanup(server.Close)
	app, out, errOut := testApp()
	app.httpClient = server.Client()
	app.stdin = strings.NewReader(listTopicsCURL(t))
	if code := app.run(t.Context(), []string{"init"}); code != exitOK {
		t.Fatalf("init exit=%d stderr=%s", code, errOut.String())
	}
	credential, err := app.store.Load(t.Context(), "0")
	if err != nil {
		t.Fatal(err)
	}
	credential.RequestHost = strings.TrimPrefix(server.URL, "http://")
	if err := app.store.Save(t.Context(), credential); err != nil {
		t.Fatal(err)
	}
	directory := filepath.Join(t.TempDir(), "conversation-export")

	// When
	out.Reset()
	code := app.run(t.Context(), []string{
		"conversation", "export",
		"https://chat.google.com/room/AAQAP4TTGjI/qMAc4oTc2i8/qMAc4oTc2i8?cls=10",
		"--output", directory,
	})

	// Then
	if code != exitOK {
		t.Fatalf("export exit=%d stderr=%s", code, errOut.String())
	}
	text := out.String()
	if !strings.Contains(text, "listed topics: 2") || !strings.Contains(text, "listed messages: 2") {
		t.Fatalf("summary = %s", text)
	}
	markdown, err := os.ReadFile(filepath.Join(directory, "topics.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(markdown), "start") || !strings.Contains(string(markdown), "newer") || strings.Contains(string(markdown), "older") {
		t.Fatalf("markdown = %s", markdown)
	}
}

func cliSearchResponse() []byte {
	hit := make([]any, 49)
	hit[0] = []any{"space/spaceidxxx1", "spaceidxxx1", 1}
	hit[2] = "1789000000001000"
	hit[48] = "hit-text-01"
	payload, err := json.Marshal([]any{"", 1, nil, "", []any{hit}})
	if err != nil {
		panic(err)
	}
	outer, err := json.Marshal([]any{[]any{"wrb.fr", "SBNmJb", string(payload), nil, nil, nil, "1"}})
	if err != nil {
		panic(err)
	}
	return append([]byte(")]}'\n\n"), outer...)
}

func cliMembersResponse() []byte {
	row := []any{[]any{[]any{[]any{"123456789012345678901"}}, nil, []any{[]any{"spaceidxxx1"}}}, "1789000000001000", 1, nil, 1}
	names := []any{[]any{"Member One Name", []any{"123456789012345678901"}}}
	payload, err := json.Marshal([]any{[]any{"dfe.lm.lm", []any{row}, nil, "", nil, nil, nil, names}})
	if err != nil {
		panic(err)
	}
	return append([]byte(")]}'\n\n"), payload...)
}

func cliGroupResponse() []byte {
	record := []any{[]any{[]any{[]any{[]any{"spaceidxxx1"}}}}, "Space One Name"}
	payload, err := json.Marshal([]any{[]any{"dfe.g.gg", record}})
	if err != nil {
		panic(err)
	}
	return append([]byte(")]}'\n\n"), payload...)
}

func cliHundredSlotJSON(slot int, value any) string {
	body := make([]any, 100)
	body[slot] = value
	body[99] = []any{"keep-blob"}
	encoded, err := json.Marshal(body)
	if err != nil {
		panic(err)
	}
	return string(encoded)
}
