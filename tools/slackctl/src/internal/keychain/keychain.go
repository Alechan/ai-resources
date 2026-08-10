package keychain

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/Alechan/ai-resources/tools/slackctl/src/internal/auth"
	nativekeychain "github.com/keybase/go-keychain"
)

const service = "slackctl"

var ErrItemNotFound = errors.New("keychain item not found")

type Backend interface {
	Set(service, account string, data []byte) error
	Get(service, account string) ([]byte, error)
	Delete(service, account string) error
}

type nativeBackend struct{}

func (nativeBackend) Set(service, account string, data []byte) error {
	if err := nativekeychain.DeleteGenericPasswordItem(service, account); err != nil && err != nativekeychain.ErrorItemNotFound {
		return err
	}
	item := nativekeychain.NewGenericPassword(service, account, service, data, "")
	return nativekeychain.AddItem(item)
}

func (nativeBackend) Get(service, account string) ([]byte, error) {
	data, err := nativekeychain.GetGenericPassword(service, account, "", "")
	if err == nativekeychain.ErrorItemNotFound {
		return nil, ErrItemNotFound
	}
	return data, err
}

func (nativeBackend) Delete(service, account string) error {
	err := nativekeychain.DeleteGenericPasswordItem(service, account)
	if err == nativekeychain.ErrorItemNotFound {
		return ErrItemNotFound
	}
	return err
}

type Store struct {
	backend Backend
}

func New() *Store                     { return NewStore(nativeBackend{}) }
func NewStore(backend Backend) *Store { return &Store{backend: backend} }

func (s *Store) Save(_ context.Context, credential auth.Credential) error {
	data, err := json.Marshal(credential)
	if err != nil {
		return errors.New("could not serialize credentials")
	}
	err = s.backend.Set(service, credential.WorkspaceHost, data)
	clear(data)
	if err != nil {
		return errors.New("could not save credentials in macOS Keychain")
	}
	return nil
}

func (s *Store) Load(_ context.Context, host string) (auth.Credential, error) {
	data, err := s.backend.Get(service, host)
	if err != nil {
		if errors.Is(err, ErrItemNotFound) {
			return auth.Credential{}, fmt.Errorf("%w for workspace %s", auth.ErrNotFound, host)
		}
		return auth.Credential{}, errors.New("could not read credentials from macOS Keychain")
	}
	defer clear(data)
	var credential auth.Credential
	if err := json.Unmarshal(bytes.TrimSpace(data), &credential); err != nil {
		return auth.Credential{}, errors.New("invalid credential record in macOS Keychain")
	}
	return credential, nil
}

func (s *Store) Clear(_ context.Context, host string) error {
	err := s.backend.Delete(service, host)
	if err != nil && !errors.Is(err, ErrItemNotFound) {
		return errors.New("could not clear credentials from macOS Keychain")
	}

	return nil
}
