package keychain

import (
	"testing"

	"github.com/Alechan/ai-resources/tools/slackctl/src/internal/auth"
)

type fakeBackend struct {
	service string
	account string
	data    []byte
}

func (f *fakeBackend) Set(service, account string, data []byte) error {
	f.service = service
	f.account = account
	f.data = append([]byte(nil), data...)
	return nil
}

func (f *fakeBackend) Get(service, account string) ([]byte, error) {
	if service != f.service || account != f.account {
		return nil, ErrItemNotFound
	}
	return append([]byte(nil), f.data...), nil
}

func (f *fakeBackend) Delete(service, account string) error {
	if service != f.service || account != f.account {
		return ErrItemNotFound
	}
	f.data = nil
	return nil
}

func TestStoreRoundTripsCredentialThroughBackend(t *testing.T) {
	ctx := t.Context()
	backend := &fakeBackend{}
	store := NewStore(backend)
	cred := auth.Credential{WorkspaceHost: "alpha.slack.com", WorkspaceID: "T11111111", Token: auth.NewSecret("generated-token"), Cookie: auth.NewSecret("generated-cookie")}
	if err := store.Save(ctx, cred); err != nil {
		t.Fatal(err)
	}
	if backend.service != "slackctl" || backend.account != cred.WorkspaceHost {
		t.Fatalf("backend key = %s/%s", backend.service, backend.account)
	}
	got, err := store.Load(ctx, cred.WorkspaceHost)
	if err != nil || got.Token.Reveal() != cred.Token.Reveal() {
		t.Fatalf("load = %#v, %v", got, err)
	}
	if err := store.Clear(ctx, cred.WorkspaceHost); err != nil {
		t.Fatal(err)
	}
	if backend.data != nil {
		t.Fatal("clear did not delete backend data")
	}
}
