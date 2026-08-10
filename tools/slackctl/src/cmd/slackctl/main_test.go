package main

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/Alechan/ai-resources/tools/slackctl/src/internal/auth"
	"github.com/Alechan/ai-resources/tools/slackctl/src/internal/slack"
)

type fakeClient struct{}

func (fakeClient) AuthTest(context.Context) (slack.AuthInfo, error) {
	return slack.AuthInfo{UserID: "U11111111", UserName: "Example User", WorkspaceID: "T11111111"}, nil
}
func (fakeClient) ConversationsProbe(context.Context) error { return nil }
func (fakeClient) History(context.Context, string, string, string, string, int) (slack.HistoryPage, []byte, error) {
	page := slack.HistoryPage{}
	return page, []byte(`{"ok":true,"messages":[]}`), nil
}
func (fakeClient) Replies(context.Context, string, string, string, int) (slack.HistoryPage, []byte, error) {
	page := slack.HistoryPage{}
	return page, []byte(`{"ok":true,"messages":[]}`), nil
}
func (fakeClient) UserInfo(context.Context, string) (slack.User, error) { return slack.User{}, nil }
func (fakeClient) ConversationInfo(context.Context, string) (slack.Conversation, error) {
	return slack.Conversation{ID: "C22222222", IsChannel: true}, nil
}

func testApp() (*application, *bytes.Buffer, *bytes.Buffer) {
	out, errOut := &bytes.Buffer{}, &bytes.Buffer{}
	return &application{
		store: auth.NewMemoryStore(),
		clientFactory: func(auth.Credential, clientOptions) slackAPI {
			return fakeClient{}
		},
		stdin:  strings.NewReader(""),
		stdout: out,
		stderr: errOut,
		getenv: func(string) string { return "" },
	}, out, errOut
}

func TestHelpListsCommandsAndFlags(t *testing.T) {
	app, out, _ := testApp()
	if code := app.run(t.Context(), []string{"--help"}); code != exitOK {
		t.Fatalf("exit=%d", code)
	}
	for _, expected := range []string{"init", "doctor", "conversation export", "--workspace", "--timeout", "--debug", "--json"} {
		if !strings.Contains(out.String(), expected) {
			t.Errorf("help missing %q:\n%s", expected, out.String())
		}
	}
}

func TestInitAndDoctorTextAndJSON(t *testing.T) {
	app, out, errOut := testApp()
	token := "generated-token-" + t.Name()
	cookie := "generated-cookie-" + t.Name()
	app.stdin = strings.NewReader("curl https://alpha.slack.com/api/auth.test -b '" + cookie + "' --data 'token=" + token + "'")
	if code := app.run(t.Context(), []string{"init"}); code != exitOK {
		t.Fatalf("init exit=%d stderr=%s", code, errOut.String())
	}
	out.Reset()
	if code := app.run(t.Context(), []string{"doctor", "--workspace", "alpha.slack.com"}); code != exitOK {
		t.Fatalf("doctor exit=%d stderr=%s", code, errOut.String())
	}
	if !strings.Contains(out.String(), "credentials found: true") || strings.Contains(out.String(), token) || strings.Contains(out.String(), cookie) {
		t.Fatalf("unsafe doctor output: %s", out.String())
	}
	out.Reset()
	if code := app.run(t.Context(), []string{"doctor", "--workspace", "alpha.slack.com", "--json"}); code != exitOK || !strings.Contains(out.String(), `"auth_valid": true`) {
		t.Fatalf("JSON doctor exit=%d output=%s", code, out.String())
	}
}

func TestExportFlagValidation(t *testing.T) {
	app, _, errOut := testApp()
	_ = app.store.Save(t.Context(), auth.Credential{WorkspaceHost: "alpha.slack.com", WorkspaceID: "T11111111"})
	code := app.run(t.Context(), []string{
		"conversation", "export", "C22222222",
		"--workspace", "alpha.slack.com",
		"--all", "--from", "2026-01-01T00:00:00Z",
		"--output", t.TempDir(),
	})
	if code == exitOK || !strings.Contains(errOut.String(), "mutually exclusive") {
		t.Fatalf("exit=%d stderr=%s", code, errOut.String())
	}
}

func TestExportStdoutSelectedFormat(t *testing.T) {
	app, out, errOut := testApp()
	_ = app.store.Save(t.Context(), auth.Credential{
		WorkspaceHost: "alpha.slack.com",
		WorkspaceID:   "T11111111",
		Token:         auth.NewSecret("generated-token"),
		Cookie:        auth.NewSecret("generated-cookie"),
	})
	code := app.run(t.Context(), []string{
		"conversation", "export", "C22222222",
		"--workspace", "alpha.slack.com",
		"--all", "--format", "markdown", "--stdout",
		"--request-delay", "0s",
	})
	if code != exitOK || !strings.Contains(out.String(), "# Slack conversation export — C22222222") {
		t.Fatalf("exit=%d stdout=%s stderr=%s", code, out.String(), errOut.String())
	}
}

func TestGlobalFlagsBeforeCommand(t *testing.T) {
	app, out, _ := testApp()
	_ = app.store.Save(t.Context(), auth.Credential{WorkspaceHost: "alpha.slack.com"})
	code := app.run(t.Context(), []string{"--workspace", "alpha.slack.com", "--json", "doctor"})
	if code != exitOK || !strings.Contains(out.String(), `"auth_valid": true`) {
		t.Fatalf("exit=%d output=%s", code, out.String())
	}
}
