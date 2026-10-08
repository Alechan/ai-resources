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

type Template struct {
	Operation   string   `json:"operation"`
	Method      string   `json:"method"`
	URLPath     string   `json:"url_path"`
	RawQuery    string   `json:"raw_query,omitempty"`
	ContentType string   `json:"content_type,omitempty"`
	QueryNames  []string `json:"query_names,omitempty"`
	HeaderNames []string `json:"header_names,omitempty"`
	HasBody     bool     `json:"has_body"`
	Body        string   `json:"body,omitempty"`
}

type Credential struct {
	AccountKey    string
	RequestHost   string
	Method        string
	URLPath       string
	RawQuery      string
	ContentType   string
	Cookie        Secret
	Authorization Secret
	ExtraHeaders  map[string]Secret
	Body          Secret
	Templates     []Template
}

type credentialJSON struct {
	AccountKey    string            `json:"account_key"`
	RequestHost   string            `json:"request_host"`
	Method        string            `json:"method,omitempty"`
	URLPath       string            `json:"url_path,omitempty"`
	RawQuery      string            `json:"raw_query,omitempty"`
	ContentType   string            `json:"content_type,omitempty"`
	Cookie        string            `json:"cookie"`
	Authorization string            `json:"authorization,omitempty"`
	ExtraHeaders  map[string]string `json:"extra_headers,omitempty"`
	Body          string            `json:"body,omitempty"`
	Templates     []Template        `json:"templates,omitempty"`
}

func (c Credential) MarshalJSON() ([]byte, error) {
	extra := map[string]string{}
	for key, value := range c.ExtraHeaders {
		if value.Reveal() != "" {
			extra[key] = value.Reveal()
		}
	}
	if len(extra) == 0 {
		extra = nil
	}
	return json.Marshal(credentialJSON{
		AccountKey:    c.AccountKey,
		RequestHost:   c.RequestHost,
		Method:        c.Method,
		URLPath:       c.URLPath,
		RawQuery:      c.RawQuery,
		ContentType:   c.ContentType,
		Cookie:        c.Cookie.Reveal(),
		Authorization: c.Authorization.Reveal(),
		ExtraHeaders:  extra,
		Body:          c.Body.Reveal(),
		Templates:     c.Templates,
	})
}

func (c *Credential) UnmarshalJSON(data []byte) error {
	var raw credentialJSON
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	extra := map[string]Secret{}
	for key, value := range raw.ExtraHeaders {
		extra[key] = NewSecret(value)
	}
	if len(extra) == 0 {
		extra = nil
	}
	*c = Credential{
		AccountKey:    raw.AccountKey,
		RequestHost:   raw.RequestHost,
		Method:        raw.Method,
		URLPath:       raw.URLPath,
		RawQuery:      raw.RawQuery,
		ContentType:   raw.ContentType,
		Cookie:        NewSecret(raw.Cookie),
		Authorization: NewSecret(raw.Authorization),
		ExtraHeaders:  extra,
		Body:          NewSecret(raw.Body),
		Templates:     raw.Templates,
	}
	return nil
}

func (c Credential) LearnedOperations() []string {
	names := make([]string, 0, len(c.Templates))
	for _, template := range c.Templates {
		if template.Operation != "" {
			names = append(names, template.Operation)
		}
	}
	return names
}

func (c *Credential) UpsertTemplate(template Template) {
	for i, existing := range c.Templates {
		if existing.Operation == template.Operation {
			c.Templates[i] = template
			return
		}
	}
	c.Templates = append(c.Templates, template)
}

func (c Credential) OperationTemplate(name string) (Template, bool) {
	for _, template := range c.Templates {
		if template.Operation == name {
			return template, true
		}
	}
	return Template{}, false
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
	s.items[c.AccountKey] = c
	return nil
}

func (s *MemoryStore) Load(_ context.Context, account string) (Credential, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	c, ok := s.items[account]
	if !ok {
		return Credential{}, ErrNotFound
	}
	return c, nil
}

func (s *MemoryStore) Clear(_ context.Context, account string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.items, account)
	return nil
}
