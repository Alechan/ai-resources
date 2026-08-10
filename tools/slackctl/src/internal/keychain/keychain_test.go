package keychain

import (
	"context"
	"strings"
	"testing"

	"github.com/Alechan/ai-resources/tools/slackctl/src/internal/auth"
)

type fakeRunner struct {
	calls [][]string
	in    []byte
	out   []byte
	err   error
}

func (f *fakeRunner) Run(_ context.Context, input []byte, args ...string) ([]byte, error) {
	f.calls = append(f.calls, append([]string(nil), args...))
	f.in = append([]byte(nil), input...)
	return f.out, f.err
}

func TestStoreUsesServiceAndWorkspaceAccount(t *testing.T) {
	ctx := t.Context()
	runner := &fakeRunner{}
	store := NewStore(runner)
	cred := auth.Credential{WorkspaceHost: "alpha.slack.com", WorkspaceID: "T11111111", Token: auth.NewSecret("generated-token"), Cookie: auth.NewSecret("generated-cookie")}
	if err := store.Save(ctx, cred); err != nil {
		t.Fatal(err)
	}
	call := strings.Join(runner.calls[0], " ")
	if !strings.Contains(call, "add-generic-password") || !strings.Contains(call, "-s slackctl") || !strings.Contains(call, "-a alpha.slack.com") || !strings.Contains(call, "-U") {
		t.Fatalf("unexpected save command: %s", call)
	}
	if strings.Contains(call, cred.Token.Reveal()) || strings.Contains(call, cred.Cookie.Reveal()) {
		t.Fatal("credential appeared in process arguments")
	}
	runner.out = runner.in
	got, err := store.Load(ctx, cred.WorkspaceHost)
	if err != nil || got.Token.Reveal() != cred.Token.Reveal() {
		t.Fatalf("load = %#v, %v", got, err)
	}
	if err := store.Clear(ctx, cred.WorkspaceHost); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(strings.Join(runner.calls[2], " "), "delete-generic-password") {
		t.Fatal("clear did not call delete")
	}
}
