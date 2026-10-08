package service

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Alechan/ai-resources/tools/gchatctl/src/internal/auth"
)

func TestDoctorLiveClassifiesStatuses(t *testing.T) {
	tests := []struct {
		name   string
		status int
		want   string
	}{
		{name: "ok", status: http.StatusOK, want: "ok"},
		{name: "401", status: http.StatusUnauthorized, want: "http_401"},
		{name: "403", status: http.StatusForbidden, want: "http_403"},
		{name: "500", status: http.StatusInternalServerError, want: "http_other"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
				if tc.status == http.StatusOK {
					writer.Write([]byte(`)]}'

[["dfe.t.lt",[]]]`))
					return
				}
				writer.WriteHeader(tc.status)
			}))
			t.Cleanup(server.Close)
			store := auth.NewMemoryStore()
			credential := auth.Credential{
				AccountKey:  "0",
				RequestHost: strings.TrimPrefix(server.URL, "http://"),
				Method:      http.MethodPost,
				URLPath:     "/u/ACCOUNT/api/list_topics",
				Cookie:      auth.NewSecret("generated-cookie"),
				Body:        auth.NewSecret("[null,1]"),
				ExtraHeaders: map[string]auth.Secret{
					"x-framework-xsrf-token": auth.NewSecret("xsrf-token-fixture"),
				},
			}
			if err := store.Save(t.Context(), credential); err != nil {
				t.Fatal(err)
			}

			// When
			report, err := DoctorLive(t.Context(), store, "0", server.Client())

			// Then
			if err != nil {
				t.Fatal(err)
			}
			if report.LiveCheck != tc.want {
				t.Fatalf("live check = %q, want %q", report.LiveCheck, tc.want)
			}
			if !report.LiveCookie || !report.LiveXSRF {
				t.Fatalf("header class = cookie=%t xsrf=%t", report.LiveCookie, report.LiveXSRF)
			}
		})
	}
}

func TestDoctorSuccess(t *testing.T) {
	store := auth.NewMemoryStore()
	credential := auth.Credential{
		AccountKey:  "0",
		RequestHost: "chat.google.com",
		Cookie:      auth.NewSecret("generated-cookie"),
		Templates:   []auth.Template{{Operation: "bootstrap", Method: "GET", URLPath: "/u/ACCOUNT/api/get_spaces"}},
	}

	// Given
	if err := store.Save(t.Context(), credential); err != nil {
		t.Fatal(err)
	}

	// When
	report, err := Doctor(t.Context(), store, "0")

	// Then
	if err != nil {
		t.Fatal(err)
	}
	if !report.CredentialsFound || !report.HostAllowed || !report.CookiePresent {
		t.Fatalf("report = %#v", report)
	}
	if report.ListTopicsReady {
		t.Fatal("bootstrap get_spaces capture should not be list_topics ready")
	}
	if report.PaginatedWorldReady {
		t.Fatal("bootstrap capture should not be paginated_world ready")
	}
	if report.LiveCheck != "not_run" {
		t.Fatalf("phase 0 doctor should not call Google Chat: %#v", report)
	}
	if len(report.LearnedOperations) != 1 || report.LearnedOperations[0] != "bootstrap" {
		t.Fatalf("operations = %v", report.LearnedOperations)
	}
}

func TestDoctorListTopicsReady(t *testing.T) {
	store := auth.NewMemoryStore()
	credential := auth.Credential{
		AccountKey:  "0",
		RequestHost: "chat.google.com",
		URLPath:     "/u/ACCOUNT/api/list_topics",
		Cookie:      auth.NewSecret("generated-cookie"),
		Body:        auth.NewSecret("[null,1]"),
	}

	// Given
	if err := store.Save(t.Context(), credential); err != nil {
		t.Fatal(err)
	}

	// When
	report, err := Doctor(t.Context(), store, "0")

	// Then
	if err != nil {
		t.Fatal(err)
	}
	if !report.ListTopicsReady || report.ReplayPath != credential.URLPath || report.BodyBytes != 8 || report.BodySlots != 2 {
		t.Fatalf("report = %#v", report)
	}
	if report.BodyShape == "" || !strings.Contains(report.BodyShape, "len=2") || !strings.Contains(report.BodyShape, "1:num") {
		t.Fatalf("body shape = %q", report.BodyShape)
	}
}

func TestDoctorPaginatedWorldReady(t *testing.T) {
	store := auth.NewMemoryStore()
	credential := auth.Credential{
		AccountKey:  "0",
		RequestHost: "chat.google.com",
		URLPath:     "/u/ACCOUNT/api/list_topics",
		Cookie:      auth.NewSecret("generated-cookie"),
		Body:        auth.NewSecret("[null,1]"),
		Templates: []auth.Template{{
			Operation: "paginated_world",
			Method:    "POST",
			URLPath:   "/u/ACCOUNT/api/paginated_world",
			HasBody:   true,
			Body:      `[["world-body"]]`,
		}},
	}

	// Given
	if err := store.Save(t.Context(), credential); err != nil {
		t.Fatal(err)
	}

	// When
	report, err := Doctor(t.Context(), store, "0")

	// Then
	if err != nil {
		t.Fatal(err)
	}
	if !report.ListTopicsReady || !report.PaginatedWorldReady {
		t.Fatalf("report = %#v", report)
	}
}

func TestDoctorSearchReady(t *testing.T) {
	store := auth.NewMemoryStore()
	credential := auth.Credential{
		AccountKey:  "0",
		RequestHost: "chat.google.com",
		URLPath:     "/u/ACCOUNT/api/list_topics",
		Cookie:      auth.NewSecret("generated-cookie"),
		Body:        auth.NewSecret("[null,1]"),
		Templates: []auth.Template{{
			Operation: "search.messages",
			Method:    "POST",
			URLPath:   "/_/DynamiteWebUi/data/batchexecute",
			RawQuery:  "rpcids=SBNmJb",
			HasBody:   true,
			Body:      "f.req=%5B%5D&at=x",
		}},
	}

	// Given
	if err := store.Save(t.Context(), credential); err != nil {
		t.Fatal(err)
	}

	// When
	report, err := Doctor(t.Context(), store, "0")

	// Then
	if err != nil {
		t.Fatal(err)
	}
	if !report.ListTopicsReady || !report.SearchReady {
		t.Fatalf("report = %#v", report)
	}
}

func TestDoctorListMembersAndGetGroupReady(t *testing.T) {
	store := auth.NewMemoryStore()
	credential := auth.Credential{
		AccountKey:  "0",
		RequestHost: "chat.google.com",
		URLPath:     "/u/ACCOUNT/api/list_topics",
		Cookie:      auth.NewSecret("generated-cookie"),
		Body:        auth.NewSecret("[null,1]"),
		Templates: []auth.Template{
			{Operation: "list_members", Method: "POST", URLPath: "/u/ACCOUNT/api/list_members", HasBody: true, Body: "[null,1]"},
			{Operation: "get_group", Method: "POST", URLPath: "/u/ACCOUNT/api/get_group", HasBody: true, Body: "[null,1]"},
		},
	}

	// Given
	if err := store.Save(t.Context(), credential); err != nil {
		t.Fatal(err)
	}

	// When
	report, err := Doctor(t.Context(), store, "0")

	// Then
	if err != nil {
		t.Fatal(err)
	}
	if !report.ListMembersReady || !report.GetGroupReady {
		t.Fatalf("report = %#v", report)
	}
}

func TestDoctorMissingAndInvalidCredentials(t *testing.T) {
	if report, err := Doctor(t.Context(), auth.NewMemoryStore(), "0"); err == nil || report.CredentialsFound {
		t.Fatalf("missing report=%#v error=%v", report, err)
	}

	store := auth.NewMemoryStore()
	_ = store.Save(t.Context(), auth.Credential{AccountKey: "0", RequestHost: "example.com", Cookie: auth.NewSecret("generated-cookie")})
	report, err := Doctor(t.Context(), store, "0")
	if err == nil || report.HostAllowed || err.Error() != "stored request host is not a Google Chat host" {
		t.Fatalf("invalid host report=%#v error=%v", report, err)
	}

	_ = store.Save(t.Context(), auth.Credential{AccountKey: "0", RequestHost: "chat.google.com"})
	report, err = Doctor(t.Context(), store, "0")
	if err == nil || report.CookiePresent || err.Error() != "stored credentials do not include a cookie" {
		t.Fatalf("missing cookie report=%#v error=%v", report, err)
	}
}
