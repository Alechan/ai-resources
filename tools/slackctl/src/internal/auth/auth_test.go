package auth

import (
	"fmt"
	"testing"
)

func TestSecretAlwaysRedacts(t *testing.T) {
	s := NewSecret("synthetic-secret-" + t.Name())
	if got := fmt.Sprintf("%s %v %#v", s, s, s); got != "[REDACTED] [REDACTED] [REDACTED]" {
		t.Fatalf("secret formatting = %q", got)
	}
	if s.Reveal() == "" {
		t.Fatal("secret should retain its value")
	}
}

func TestMemoryStoreWorkspaces(t *testing.T) {
	ctx := t.Context()
	store := NewMemoryStore()
	first := Credential{WorkspaceHost: "alpha.slack.com", WorkspaceID: "T11111111", Token: NewSecret("generated-token-one"), Cookie: NewSecret("generated-cookie-one")}
	second := Credential{WorkspaceHost: "beta.slack.com", WorkspaceID: "T22222222", Token: NewSecret("generated-token-two"), Cookie: NewSecret("generated-cookie-two")}
	if err := store.Save(ctx, first); err != nil {
		t.Fatal(err)
	}
	if err := store.Save(ctx, second); err != nil {
		t.Fatal(err)
	}
	got, err := store.Load(ctx, first.WorkspaceHost)
	if err != nil || got.WorkspaceID != first.WorkspaceID {
		t.Fatalf("load = %#v, %v", got, err)
	}
	first.WorkspaceID = "T33333333"
	if err := store.Save(ctx, first); err != nil {
		t.Fatal(err)
	}
	got, _ = store.Load(ctx, first.WorkspaceHost)
	if got.WorkspaceID != first.WorkspaceID {
		t.Fatal("save did not replace credential")
	}
	if err := store.Clear(ctx, first.WorkspaceHost); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Load(ctx, first.WorkspaceHost); !IsNotFound(err) {
		t.Fatalf("missing load error = %v", err)
	}
}

func TestCredentialJSONRoundTrip(t *testing.T) {
	in := Credential{WorkspaceHost: "alpha.slack.com", WorkspaceID: "T11111111", Token: NewSecret("generated-token"), Cookie: NewSecret("generated-cookie")}
	data, err := in.MarshalJSON()
	if err != nil {
		t.Fatal(err)
	}
	var out Credential
	if err := out.UnmarshalJSON(data); err != nil {
		t.Fatal(err)
	}
	if out.Token.Reveal() != in.Token.Reveal() || out.Cookie.Reveal() != in.Cookie.Reveal() {
		t.Fatal("credential did not round trip")
	}
	if fmt.Sprint(out) != "{alpha.slack.com T11111111 [REDACTED] [REDACTED]}" {
		t.Fatalf("credential formatting leaked or changed: %s", fmt.Sprint(out))
	}
}
