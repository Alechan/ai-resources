package curlparse

import (
	"strings"
	"testing"
)

func TestParseSupportedForms(t *testing.T) {
	token := "generated-token-" + t.Name()
	cookie := "generated-cookie-" + t.Name()
	tests := []struct {
		name  string
		input string
	}{
		{"cookie flag and urlencoded token", "curl 'https://alpha.slack.com/api/auth.test' -b '" + cookie + "' --data-urlencode 'token=" + token + "'"},
		{"cookie header and raw token", "curl 'https://alpha.slack.com/api/auth.test' -H 'Cookie: " + cookie + "' --data-raw 'token=" + token + "&limit=1'"},
		{"multipart token", "curl https://alpha.slack.com/api/auth.test -H \"cookie: " + cookie + "\" -F \"token=" + token + "\""},
		{"multiline", "curl \\\n  'https://alpha.slack.com/api/auth.test' \\\n  -b '" + cookie + "' \\\n  --data 'token=" + token + "'"},
		{"ANSI C quoted data", "curl https://alpha.slack.com/api/auth.test -b '" + cookie + "' --data-raw $'token=" + token + "&note=line\\nvalue'"},
		{"multipart raw body", "curl https://alpha.slack.com/api/auth.test -b '" + cookie + "' -H 'content-type: multipart/form-data; boundary=Boundary' --data-raw $'--Boundary\\r\\nContent-Disposition: form-data; name=\"token\"\\r\\n\\r\\n" + token + "\\r\\n--Boundary--'"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := Parse(tc.input)
			if err != nil {
				t.Fatal(err)
			}
			if got.WorkspaceHost != "alpha.slack.com" || got.Token.Reveal() != token || got.Cookie.Reveal() != cookie {
				t.Fatalf("parse = %#v", got)
			}
		})
	}
}

func TestParseRejectsInvalidInputWithoutLeaks(t *testing.T) {
	token := "generated-sensitive-token-" + t.Name()
	cookie := "generated-sensitive-cookie-" + t.Name()
	tests := []string{
		"curl https://alpha.slack.com/api/auth.test -b '" + cookie + "'",
		"curl https://alpha.slack.com/api/auth.test --data 'token=" + token + "'",
		"curl https://invalid.example/api -b '" + cookie + "' --data 'token=" + token + "'",
		"curl https://alpha.slack.com/api https://beta.slack.com/api -b '" + cookie + "' --data 'token=" + token + "'",
		"curl 'https://alpha.slack.com/api -b x",
	}
	for _, input := range tests {
		_, err := Parse(input)
		if err == nil {
			t.Fatalf("Parse(%q) unexpectedly succeeded", input)
		}
		if strings.Contains(err.Error(), token) || strings.Contains(err.Error(), cookie) {
			t.Fatalf("error leaked a secret: %v", err)
		}
	}
}
