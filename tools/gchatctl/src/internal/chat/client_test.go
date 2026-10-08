package chat

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/Alechan/ai-resources/tools/gchatctl/src/internal/auth"
	"github.com/Alechan/ai-resources/tools/gchatctl/src/internal/topics"
)

func TestProbeStatusClassifiesErrors(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want string
	}{
		{name: "ok", err: nil, want: "ok"},
		{name: "401", err: &APIError{Kind: ErrorAuth, StatusCode: http.StatusUnauthorized}, want: "http_401"},
		{name: "403", err: &APIError{Kind: ErrorAuth, StatusCode: http.StatusForbidden}, want: "http_403"},
		{name: "400", err: &APIError{Kind: ErrorAPI, StatusCode: http.StatusBadRequest}, want: "http_other"},
		{name: "transport", err: &APIError{Kind: ErrorTransport}, want: "transport"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// When
			got := ProbeStatus(tc.err)

			// Then
			if got != tc.want {
				t.Fatalf("ProbeStatus() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestListTopicsParsesFixture(t *testing.T) {
	// Given
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path != "/u/0/api/list_topics" {
			t.Errorf("path = %s", request.URL.Path)
		}
		if request.Header.Get("Cookie") != "SID="+"generated-cookie" {
			t.Errorf("cookie missing")
		}
		if request.Header.Get("Origin") != "https://chat.google.com" {
			t.Errorf("origin = %s", request.Header.Get("Origin"))
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
    "hello from fixture"
  ]]
]]]]`))
	}))
	t.Cleanup(server.Close)
	client := NewClient(auth.Credential{
		AccountKey:  "0",
		RequestHost: strings.TrimPrefix(server.URL, "http://"),
		Method:      http.MethodPost,
		URLPath:     "/u/ACCOUNT/api/list_topics",
		Cookie:      auth.NewSecret("SID=" + "generated-cookie"),
		Body:        auth.NewSecret("[null,1]"),
	}, server.Client())

	// When
	page, raw, err := client.ListTopics(t.Context())

	// Then
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), "dfe.t.lt") {
		t.Fatalf("raw = %s", raw)
	}
	if len(page.Topics) != 1 || page.Topics[0].Messages[0].Text != "hello from fixture" {
		t.Fatalf("page = %#v", page)
	}
}

func TestListTopicsRequiresStoredRPC(t *testing.T) {
	client := NewClient(auth.Credential{
		AccountKey:  "0",
		RequestHost: "chat.google.com",
		URLPath:     "/u/ACCOUNT/api/get_spaces",
		Cookie:      auth.NewSecret("SID=" + "generated-cookie"),
		Body:        auth.NewSecret("[]"),
	}, nil)
	if _, _, err := client.ListTopics(t.Context()); err == nil || !strings.Contains(err.Error(), "list_topics") {
		t.Fatalf("err = %v", err)
	}
}

func TestListTopicsPageKeepsPermalinkReferer(t *testing.T) {
	// Given
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.Header.Get("Referer") != "https://chat.google.com/room/AAQAP4TTGjI" {
			t.Errorf("referer = %s", request.Header.Get("Referer"))
		}
		if request.Header.Get("x-goog-chat-space-id") != "AAQAP4TTGjI" {
			t.Errorf("space = %s", request.Header.Get("x-goog-chat-space-id"))
		}
		writer.Write([]byte(`)]}'

[["dfe.t.lt",[]]]`))
	}))
	t.Cleanup(server.Close)
	client := NewClient(auth.Credential{
		AccountKey:  "0",
		RequestHost: strings.TrimPrefix(server.URL, "http://"),
		Method:      http.MethodPost,
		URLPath:     "/u/ACCOUNT/api/list_topics",
		Cookie:      auth.NewSecret("SID=" + "generated-cookie"),
		Body:        auth.NewSecret(`[null,40,null,[null,111],[3,1,4],1000,20,[["oldspaceid1"]],[222],[333],2]`),
		ExtraHeaders: map[string]auth.Secret{
			"Referer":              auth.NewSecret("https://chat.google.com/room/otherspace1"),
			"x-goog-chat-space-id": auth.NewSecret("otherspace1"),
		},
	}, server.Client())

	// When
	_, err := client.ListTopicsPage(t.Context(), topics.RequestOptions{SpaceID: "AAQAP4TTGjI", Cursor: 1789000000000001})

	// Then
	if err != nil {
		t.Fatal(err)
	}
}

func TestListTopicsUnauthorized(t *testing.T) {
	// Given
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		writer.WriteHeader(http.StatusUnauthorized)
	}))
	t.Cleanup(server.Close)
	client := NewClient(auth.Credential{
		AccountKey:  "0",
		RequestHost: strings.TrimPrefix(server.URL, "http://"),
		Method:      http.MethodPost,
		URLPath:     "/u/ACCOUNT/api/list_topics",
		Cookie:      auth.NewSecret("SID=" + "generated-cookie"),
		Body:        auth.NewSecret("[]"),
	}, server.Client())

	// When
	_, _, err := client.ListTopics(t.Context())

	// Then
	if !IsAuth(err) {
		t.Fatalf("err = %v", err)
	}
}

func TestPaginatedWorldPostsStoredTemplateBody(t *testing.T) {
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
  [[null,"spaceidxxx1"],null,"1","2",null,null,null,null,null,"Space One Name",null,null,null,"spaceidxxx1"]
]]]]`))
	}))
	t.Cleanup(server.Close)
	client := NewClient(auth.Credential{
		AccountKey:  "0",
		RequestHost: strings.TrimPrefix(server.URL, "http://"),
		Method:      http.MethodPost,
		URLPath:     "/u/ACCOUNT/api/list_topics",
		Cookie:      auth.NewSecret("SID=" + "generated-cookie"),
		Body:        auth.NewSecret("[null,1]"),
		Templates: []auth.Template{{
			Operation: "paginated_world",
			Method:    "POST",
			URLPath:   "/u/ACCOUNT/api/paginated_world",
			HasBody:   true,
			Body:      `[["world-body"]]`,
		}},
	}, server.Client())

	// When
	world, err := client.PaginatedWorld(t.Context())

	// Then
	if err != nil {
		t.Fatal(err)
	}
	if len(world.Spaces) != 1 || world.Spaces[0].ID != "spaceidxxx1" || world.Spaces[0].Name != "Space One Name" {
		t.Fatalf("world = %#v", world)
	}
}

func TestPaginatedWorldRequiresLearnedBody(t *testing.T) {
	client := NewClient(auth.Credential{
		AccountKey:  "0",
		RequestHost: "chat.google.com",
		URLPath:     "/u/ACCOUNT/api/list_topics",
		Cookie:      auth.NewSecret("SID=" + "generated-cookie"),
		Body:        auth.NewSecret("[null,1]"),
	}, nil)

	// When
	_, err := client.PaginatedWorld(t.Context())

	// Then
	if err == nil || !strings.Contains(err.Error(), "learn paginated_world") {
		t.Fatalf("err = %v", err)
	}
}

func TestSearchMessagesRewritesQueryAndParsesHits(t *testing.T) {
	// Given
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path != "/_/DynamiteWebUi/data/batchexecute" {
			t.Errorf("path = %s", request.URL.Path)
		}
		if request.URL.Query().Get("rpcids") != "SBNmJb" {
			t.Errorf("query = %s", request.URL.RawQuery)
		}
		if request.Header.Get("Content-Type") != "application/x-www-form-urlencoded" {
			t.Errorf("content-type = %s", request.Header.Get("Content-Type"))
		}
		body, _ := io.ReadAll(request.Body)
		form := string(body)
		if strings.Contains(form, "nps") || !strings.Contains(form, "hello") {
			t.Errorf("body = %q", form)
		}
		if !strings.Contains(form, "synthetic-at-token-value") {
			t.Errorf("at missing from %q", form)
		}
		writer.Write(syntheticSearchResponse())
	}))
	t.Cleanup(server.Close)
	form := "f.req=" + url.QueryEscape(`[[["SBNmJb","[null,null,null,\"nps\",null,\"xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx\"]",null,"1"]]]`) + "&at=synthetic-at-token-value"
	client := NewClient(auth.Credential{
		AccountKey:  "0",
		RequestHost: strings.TrimPrefix(server.URL, "http://"),
		Method:      http.MethodPost,
		URLPath:     "/u/ACCOUNT/api/list_topics",
		RawQuery:    "c=1",
		Cookie:      auth.NewSecret("SID=" + "generated-cookie"),
		Body:        auth.NewSecret("[null,1]"),
		Templates: []auth.Template{{
			Operation:   "search.messages",
			Method:      "POST",
			URLPath:     "/_/DynamiteWebUi/data/batchexecute",
			RawQuery:    "rpcids=SBNmJb&source-path=/app/search",
			ContentType: "application/x-www-form-urlencoded",
			HasBody:     true,
			Body:        form,
		}},
	}, server.Client())

	// When
	result, err := client.SearchMessages(t.Context(), "hello")

	// Then
	if err != nil {
		t.Fatal(err)
	}
	if result.Query != "hello" || len(result.Hits) != 1 || result.Hits[0].Text != "hit-text-01" {
		t.Fatalf("result = %#v", result)
	}
}

func TestSearchMessagesRequiresLearnedBody(t *testing.T) {
	client := NewClient(auth.Credential{
		AccountKey:  "0",
		RequestHost: "chat.google.com",
		URLPath:     "/u/ACCOUNT/api/list_topics",
		Cookie:      auth.NewSecret("SID=" + "generated-cookie"),
		Body:        auth.NewSecret("[null,1]"),
	}, nil)
	if _, err := client.SearchMessages(t.Context(), "hello"); err == nil || !strings.Contains(err.Error(), "learn search.messages") {
		t.Fatalf("err = %v", err)
	}
}

func TestListMembersRewritesSpaceAndParsesRoster(t *testing.T) {
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
		writer.Write(syntheticMembersResponse())
	}))
	t.Cleanup(server.Close)
	client := NewClient(auth.Credential{
		AccountKey:  "0",
		RequestHost: strings.TrimPrefix(server.URL, "http://"),
		Method:      http.MethodPost,
		URLPath:     "/u/ACCOUNT/api/list_topics",
		Cookie:      auth.NewSecret("SID=" + "generated-cookie"),
		Body:        auth.NewSecret("[null,1]"),
		Templates: []auth.Template{{
			Operation: "list_members",
			Method:    "POST",
			URLPath:   "/u/ACCOUNT/api/list_members",
			HasBody:   true,
			Body:      hundredSlotJSON(4, []any{[]any{"oldspaceid1"}}),
		}},
	}, server.Client())

	// When
	roster, err := client.ListMembers(t.Context(), "spaceidxxx1")

	// Then
	if err != nil {
		t.Fatal(err)
	}
	if len(roster.Members) != 1 || roster.Members[0].ID != "123456789012345678901" || roster.Members[0].Name != "Member One Name" {
		t.Fatalf("roster = %#v", roster)
	}
}

func TestGetGroupRewritesSpaceAndParsesRecord(t *testing.T) {
	// Given
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path != "/u/0/api/get_group" {
			t.Errorf("path = %s", request.URL.Path)
		}
		if request.Header.Get("x-goog-chat-space-id") != "spaceidxxx1" {
			t.Errorf("space header = %s", request.Header.Get("x-goog-chat-space-id"))
		}
		writer.Write(syntheticGroupResponse())
	}))
	t.Cleanup(server.Close)
	client := NewClient(auth.Credential{
		AccountKey:  "0",
		RequestHost: strings.TrimPrefix(server.URL, "http://"),
		Method:      http.MethodPost,
		URLPath:     "/u/ACCOUNT/api/list_topics",
		Cookie:      auth.NewSecret("SID=" + "generated-cookie"),
		Body:        auth.NewSecret("[null,1]"),
		Templates: []auth.Template{{
			Operation: "get_group",
			Method:    "POST",
			URLPath:   "/u/ACCOUNT/api/get_group",
			HasBody:   true,
			Body:      hundredSlotJSON(0, []any{[]any{[]any{"123456789012345678901"}}, nil, []any{[]any{"oldspaceid1"}}}),
		}},
	}, server.Client())

	// When
	group, err := client.GetGroup(t.Context(), "spaceidxxx1")

	// Then
	if err != nil {
		t.Fatal(err)
	}
	if group.ID != "spaceidxxx1" || group.Name != "Space One Name" {
		t.Fatalf("group = %#v", group)
	}
}

func TestListMembersRequiresLearnedBody(t *testing.T) {
	client := NewClient(auth.Credential{
		AccountKey:  "0",
		RequestHost: "chat.google.com",
		URLPath:     "/u/ACCOUNT/api/list_topics",
		Cookie:      auth.NewSecret("SID=" + "generated-cookie"),
		Body:        auth.NewSecret("[null,1]"),
	}, nil)
	if _, err := client.ListMembers(t.Context(), "spaceidxxx1"); err == nil || !strings.Contains(err.Error(), "learn list_members") {
		t.Fatalf("err = %v", err)
	}
}

func TestGetGroupRequiresLearnedBody(t *testing.T) {
	client := NewClient(auth.Credential{
		AccountKey:  "0",
		RequestHost: "chat.google.com",
		URLPath:     "/u/ACCOUNT/api/list_topics",
		Cookie:      auth.NewSecret("SID=" + "generated-cookie"),
		Body:        auth.NewSecret("[null,1]"),
	}, nil)
	if _, err := client.GetGroup(t.Context(), "spaceidxxx1"); err == nil || !strings.Contains(err.Error(), "learn get_group") {
		t.Fatalf("err = %v", err)
	}
}

func syntheticSearchResponse() []byte {
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

func syntheticMembersResponse() []byte {
	row := []any{[]any{[]any{[]any{"123456789012345678901"}}, nil, []any{[]any{"spaceidxxx1"}}}, "1789000000001000", 1, nil, 1}
	names := []any{[]any{"Member One Name", []any{"123456789012345678901"}}}
	payload, err := json.Marshal([]any{[]any{"dfe.lm.lm", []any{row}, nil, "", nil, nil, nil, names}})
	if err != nil {
		panic(err)
	}
	return append([]byte(")]}'\n\n"), payload...)
}

func syntheticGroupResponse() []byte {
	record := []any{[]any{[]any{[]any{[]any{"spaceidxxx1"}}}}, "Space One Name"}
	payload, err := json.Marshal([]any{[]any{"dfe.g.gg", record}})
	if err != nil {
		panic(err)
	}
	return append([]byte(")]}'\n\n"), payload...)
}

func hundredSlotJSON(slot int, value any) string {
	body := make([]any, 100)
	body[slot] = value
	body[99] = []any{"keep-blob"}
	encoded, err := json.Marshal(body)
	if err != nil {
		panic(err)
	}
	return string(encoded)
}
