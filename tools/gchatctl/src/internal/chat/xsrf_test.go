package chat

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Alechan/ai-resources/tools/gchatctl/src/internal/auth"
)

func TestParseXSRFFromWIZGlobalData(t *testing.T) {
	// Given
	html := []byte(`<html><script>window.WIZ_global_data = {"SNlM0e":"xsrf-token-fixture-ok","n":1};</script></html>`)

	// When
	got, ok := ParseXSRF(html)

	// Then
	if !ok || got != "xsrf-token-fixture-ok" {
		t.Fatalf("ParseXSRF() = %q %t", got, ok)
	}
}

func TestParseXSRFMissing(t *testing.T) {
	if _, ok := ParseXSRF([]byte(`<html><body>no token</body></html>`)); ok {
		t.Fatal("expected no token")
	}
}

func TestListTopicsRefreshesXSRFOnce(t *testing.T) {
	posts := 0
	gets := 0
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		switch request.Method {
		case http.MethodGet:
			gets++
			if request.Header.Get("Cookie") != "SID="+"generated-cookie" {
				t.Errorf("refresh cookie missing")
			}
			writer.Write([]byte(`<script>window.WIZ_global_data={"SNlM0e":"xsrf-token-refreshed"};</script>`))
		default:
			posts++
			if posts == 1 {
				if request.Header.Get("x-framework-xsrf-token") != "xsrf-token-stale" {
					t.Errorf("first xsrf = %s", request.Header.Get("x-framework-xsrf-token"))
				}
				writer.WriteHeader(http.StatusForbidden)
				return
			}
			if request.Header.Get("x-framework-xsrf-token") != "xsrf-token-refreshed" {
				t.Errorf("retry xsrf = %s", request.Header.Get("x-framework-xsrf-token"))
			}
			writer.Write([]byte(`)]}'

[["dfe.t.lt",[]]]`))
		}
	}))
	t.Cleanup(server.Close)
	saved := 0
	client := NewClient(auth.Credential{
		AccountKey:  "0",
		RequestHost: strings.TrimPrefix(server.URL, "http://"),
		Method:      http.MethodPost,
		URLPath:     "/u/ACCOUNT/api/list_topics",
		Cookie:      auth.NewSecret("SID=" + "generated-cookie"),
		Body:        auth.NewSecret("[null,1]"),
		ExtraHeaders: map[string]auth.Secret{
			"x-framework-xsrf-token": auth.NewSecret("xsrf-token-stale"),
		},
	}, server.Client()).WithPersist(func(_ context.Context, credential auth.Credential) error {
		saved++
		if credential.ExtraHeaders["x-framework-xsrf-token"].Reveal() != "xsrf-token-refreshed" {
			t.Errorf("persisted token missing")
		}
		return nil
	})

	// When
	_, _, err := client.ListTopics(t.Context())

	// Then
	if err != nil {
		t.Fatal(err)
	}
	if posts != 2 || gets != 1 || saved != 1 {
		t.Fatalf("posts=%d gets=%d saved=%d", posts, gets, saved)
	}
}
