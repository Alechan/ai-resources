package auth

import (
	"fmt"
	"testing"
)

func TestSecretAlwaysRedacts(t *testing.T) {
	// Given
	s := NewSecret("synthetic-secret-" + t.Name())

	// When
	got := fmt.Sprintf("%s %v %#v", s, s, s)

	// Then
	if got != "[REDACTED] [REDACTED] [REDACTED]" {
		t.Fatalf("secret formatting = %q", got)
	}
	if s.Reveal() == "" {
		t.Fatal("secret should retain its value")
	}
}

func TestMemoryStoreAccounts(t *testing.T) {
	ctx := t.Context()
	store := NewMemoryStore()
	first := Credential{
		AccountKey:    "0",
		RequestHost:   "chat.google.com",
		Cookie:        NewSecret("generated-cookie-one"),
		Authorization: NewSecret("generated-auth-one"),
	}
	second := Credential{
		AccountKey:  "1",
		RequestHost: "clients6.google.com",
		Cookie:      NewSecret("generated-cookie-two"),
	}

	// When
	if err := store.Save(ctx, first); err != nil {
		t.Fatal(err)
	}
	if err := store.Save(ctx, second); err != nil {
		t.Fatal(err)
	}

	// Then
	got, err := store.Load(ctx, first.AccountKey)
	if err != nil || got.RequestHost != first.RequestHost {
		t.Fatalf("load = %#v, %v", got, err)
	}
	first.RequestHost = "chat.googleapis.com"
	if err := store.Save(ctx, first); err != nil {
		t.Fatal(err)
	}
	got, _ = store.Load(ctx, first.AccountKey)
	if got.RequestHost != first.RequestHost {
		t.Fatal("save did not replace credential")
	}
	if err := store.Clear(ctx, first.AccountKey); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Load(ctx, first.AccountKey); !IsNotFound(err) {
		t.Fatalf("missing load error = %v", err)
	}
}

func TestCredentialJSONRoundTrip(t *testing.T) {
	// Given
	in := Credential{
		AccountKey:    "0",
		RequestHost:   "chat.google.com",
		Cookie:        NewSecret("generated-cookie"),
		Authorization: NewSecret("generated-auth"),
		ExtraHeaders:  map[string]Secret{"x-goog-authuser": NewSecret("0")},
		Templates: []Template{{
			Operation:   "bootstrap",
			Method:      "POST",
			URLPath:     "/u/ACCOUNT/api/example",
			QueryNames:  []string{"rt"},
			HeaderNames: []string{"authorization", "cookie"},
			HasBody:     true,
		}},
	}

	// When
	data, err := in.MarshalJSON()
	if err != nil {
		t.Fatal(err)
	}
	var out Credential
	if err := out.UnmarshalJSON(data); err != nil {
		t.Fatal(err)
	}

	// Then
	if out.Cookie.Reveal() != in.Cookie.Reveal() || out.Authorization.Reveal() != in.Authorization.Reveal() {
		t.Fatal("credential did not round trip")
	}
	if out.ExtraHeaders["x-goog-authuser"].Reveal() != "0" {
		t.Fatal("extra headers did not round trip")
	}
	if len(out.Templates) != 1 || out.Templates[0].Operation != "bootstrap" {
		t.Fatalf("templates = %#v", out.Templates)
	}
	printed := fmt.Sprintf("%v %v", out.Cookie, out.Authorization)
	if printed != "[REDACTED] [REDACTED]" {
		t.Fatalf("credential formatting leaked or changed: %s", printed)
	}
}

func TestUpsertTemplateReplacesSameOperation(t *testing.T) {
	// Given
	credential := Credential{}
	credential.UpsertTemplate(Template{Operation: "search", Method: "GET", URLPath: "/old"})

	// When
	credential.UpsertTemplate(Template{Operation: "search", Method: "POST", URLPath: "/new"})
	credential.UpsertTemplate(Template{Operation: "spaces.list", Method: "GET", URLPath: "/spaces"})

	// Then
	if len(credential.Templates) != 2 {
		t.Fatalf("templates = %#v", credential.Templates)
	}
	if credential.Templates[0].Operation != "search" || credential.Templates[0].Method != "POST" || credential.Templates[0].URLPath != "/new" {
		t.Fatalf("search template = %#v", credential.Templates[0])
	}
	if got := credential.LearnedOperations(); fmt.Sprint(got) != "[search spaces.list]" {
		t.Fatalf("operations = %v", got)
	}
}
