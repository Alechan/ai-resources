package auth

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/Alechan/ai-resources/tools/gdrivectl/src/internal/fail"
)

func writeScript(t *testing.T, body string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "fake-gcloud.sh")
	content := "#!/bin/sh\nset -eu\n" + body + "\n"
	if err := os.WriteFile(p, []byte(content), 0o755); err != nil {
		t.Fatalf("write script: %v", err)
	}
	return p
}

func TestAccessTokenSuccess(t *testing.T) {
	bin := writeScript(t, `echo "  token-123  "`)
	p := NewGcloudTokenProvider(bin)

	got, err := p.AccessToken(context.Background())
	if err != nil {
		t.Fatalf("AccessToken() error = %v", err)
	}
	if got != "token-123" {
		t.Fatalf("AccessToken() = %q, want %q", got, "token-123")
	}
}

func TestAccessTokenEmptyBinaryPath(t *testing.T) {
	p := NewGcloudTokenProvider("   ")
	_, err := p.AccessToken(context.Background())
	assertFailCategory(t, err, "config")
}

func TestAccessTokenMissingBinary(t *testing.T) {
	p := NewGcloudTokenProvider(filepath.Join(t.TempDir(), "missing-gcloud"))
	_, err := p.AccessToken(context.Background())
	assertFailCategory(t, err, "config")
}

func TestAccessTokenScopeInsufficient(t *testing.T) {
	bin := writeScript(t, `echo "ACCESS_TOKEN_SCOPE_INSUFFICIENT" 1>&2; exit 1`)
	p := NewGcloudTokenProvider(bin)
	_, err := p.AccessToken(context.Background())
	assertFailCategory(t, err, "scope")
}

func TestAccessTokenGenericFailure(t *testing.T) {
	bin := writeScript(t, `echo "something failed" 1>&2; exit 1`)
	p := NewGcloudTokenProvider(bin)
	_, err := p.AccessToken(context.Background())
	assertFailCategory(t, err, "auth")
}

func TestAccessTokenConfigStorePermissionFailure(t *testing.T) {
	bin := writeScript(t, `echo "Unable to create private file /Users/test/.config/gcloud/credentials.db: Permission denied" 1>&2; exit 1`)
	p := NewGcloudTokenProvider(bin)
	_, err := p.AccessToken(context.Background())
	assertFailCategory(t, err, "config")
}

func TestAccessTokenEmptyOutput(t *testing.T) {
	bin := writeScript(t, `:`)
	p := NewGcloudTokenProvider(bin)
	_, err := p.AccessToken(context.Background())
	assertFailCategory(t, err, "auth")
}

func assertFailCategory(t *testing.T, err error, want string) {
	t.Helper()
	var fe *fail.Error
	if !errors.As(err, &fe) {
		t.Fatalf("error type = %T, want *fail.Error (err=%v)", err, err)
	}
	if fe.Category != want {
		t.Fatalf("category = %q, want %q", fe.Category, want)
	}
}
