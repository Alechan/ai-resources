package keychain

import (
	"testing"

	"github.com/Alechan/ai-resources/tools/gchatctl/src/internal/auth"
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
	cred := auth.Credential{
		AccountKey:    "0",
		RequestHost:   "chat.google.com",
		Cookie:        auth.NewSecret("generated-cookie"),
		Authorization: auth.NewSecret("generated-auth"),
		Templates:     []auth.Template{{Operation: "bootstrap", Method: "GET", URLPath: "/u/ACCOUNT/api/get_spaces"}},
	}

	// When
	if err := store.Save(ctx, cred); err != nil {
		t.Fatal(err)
	}

	// Then
	if backend.service != "gchatctl" || backend.account != cred.AccountKey {
		t.Fatalf("backend key = %s/%s", backend.service, backend.account)
	}
	got, err := store.Load(ctx, cred.AccountKey)
	if err != nil || got.Cookie.Reveal() != cred.Cookie.Reveal() || got.Templates[0].Operation != "bootstrap" {
		t.Fatalf("load = %#v, %v", got, err)
	}
	if err := store.Clear(ctx, cred.AccountKey); err != nil {
		t.Fatal(err)
	}
	if backend.data != nil {
		t.Fatal("clear did not delete backend data")
	}
}
