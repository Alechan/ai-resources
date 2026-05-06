package auth

import (
	"github.com/alejandro-danos/jenkinsctl/internal/app"
	"testing"
)

func TestBasicAuth(t *testing.T) {
	cfg := &app.Config{Username: "u", APIToken: "t"}
	provider := New(cfg)
	u, tkn := provider.BasicAuth()
	if u != "u" || tkn != "t" {
		t.Error("credentials not returned correctly")
	}
}
