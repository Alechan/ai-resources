package auth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
)

type Secret struct {
	value string
}

func NewSecret(value string) Secret { return Secret{value: value} }
func (s Secret) Reveal() string     { return s.value }
func (Secret) String() string       { return "[REDACTED]" }
func (Secret) GoString() string     { return "[REDACTED]" }
func (Secret) Format(f fmt.State, _ rune) {
	_, _ = f.Write([]byte("[REDACTED]"))
}

type Credential struct {
	WorkspaceHost string `json:"workspace_host"`
	WorkspaceID   string `json:"workspace_id"`
	Token         Secret `json:"-"`
	Cookie        Secret `json:"-"`
}

type credentialJSON struct {
	WorkspaceHost string `json:"workspace_host"`
	WorkspaceID   string `json:"workspace_id"`
	Token         string `json:"token"`
	Cookie        string `json:"cookie"`
}

func (c Credential) MarshalJSON() ([]byte, error) {
	return json.Marshal(credentialJSON{c.WorkspaceHost, c.WorkspaceID, c.Token.Reveal(), c.Cookie.Reveal()})
}

func (c *Credential) UnmarshalJSON(data []byte) error {
	var raw credentialJSON
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	*c = Credential{raw.WorkspaceHost, raw.WorkspaceID, NewSecret(raw.Token), NewSecret(raw.Cookie)}
	return nil
}

type Store interface {
	Save(context.Context, Credential) error
	Load(context.Context, string) (Credential, error)
	Clear(context.Context, string) error
}

var ErrNotFound = errors.New("credentials not found")

func IsNotFound(err error) bool { return errors.Is(err, ErrNotFound) }

type MemoryStore struct {
	mu    sync.RWMutex
	items map[string]Credential
}

func NewMemoryStore() *MemoryStore { return &MemoryStore{items: make(map[string]Credential)} }

func (s *MemoryStore) Save(_ context.Context, c Credential) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.items[c.WorkspaceHost] = c
	return nil
}

func (s *MemoryStore) Load(_ context.Context, host string) (Credential, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	c, ok := s.items[host]
	if !ok {
		return Credential{}, ErrNotFound
	}
	return c, nil
}

func (s *MemoryStore) Clear(_ context.Context, host string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.items, host)
	return nil
}
