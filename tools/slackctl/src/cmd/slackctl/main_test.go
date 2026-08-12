package main

import (
	"bytes"
	"context"
	"strings"
	"testing"
	"time"

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

type rangeRecordingClient struct {
	fakeClient
	oldest string
	latest string
}

func (c *rangeRecordingClient) History(_ context.Context, _, oldest, latest, _ string, _ int) (slack.HistoryPage, []byte, error) {
	c.oldest = oldest
	c.latest = latest
	return slack.HistoryPage{}, []byte(`{"ok":true,"messages":[]}`), nil
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
	for _, expected := range []string{
		"init",
		"doctor",
		"conversation export",
		"--workspace",
		"--timeout",
		"--debug",
		"--json",
		"https://example.slack.com/archives/C22222222/p1786455295071869",
		"inclusive lower root-message boundary",
		"replies outside root-message time bounds",
	} {
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

func TestResolveExportWorkspace(t *testing.T) {
	tests := []struct {
		name                     string
		permalink, explicit, env string
		want                     string
		wantErr                  string
	}{
		{"permalink alone", "example.slack.com", "", "", "example.slack.com", ""},
		{"matching explicit workspace", "example.slack.com", "example.slack.com", "", "example.slack.com", ""},
		{"matching environment workspace", "example.slack.com", "", "example.slack.com", "example.slack.com", ""},
		{"all sources agree", "example.slack.com", "example.slack.com", "example.slack.com", "example.slack.com", ""},
		{"explicit conflicts with permalink", "example.slack.com", "other.slack.com", "", "", "does not match"},
		{"environment conflicts with permalink", "example.slack.com", "", "other.slack.com", "", "does not match"},
		{"explicit keeps precedence over environment for ID", "", "example.slack.com", "other.slack.com", "example.slack.com", ""},
		{"explicit for ID", "", "example.slack.com", "", "example.slack.com", ""},
		{"environment for app URL", "", "", "example.slack.com", "example.slack.com", ""},
		{"missing for ID", "", "", "", "", "required"},
		{"invalid host", "", "https://example.slack.com", "", "", "Slack workspace host"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := resolveExportWorkspace(tc.permalink, tc.explicit, tc.env)
			if tc.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
					t.Fatalf("error = %v, want containing %q", err, tc.wantErr)
				}
				return
			}
			if err != nil || got != tc.want {
				t.Fatalf("got %q, %v; want %q", got, err, tc.want)
			}
		})
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

func TestExportAppURLRemainsSupported(t *testing.T) {
	app, out, errOut := testApp()
	_ = app.store.Save(t.Context(), auth.Credential{
		WorkspaceHost: "example.slack.com",
		WorkspaceID:   "T11111111",
	})
	code := app.run(t.Context(), []string{
		"conversation", "export",
		"https://app.slack.com/client/T11111111/C22222222",
		"--workspace", "example.slack.com",
		"--all",
		"--format", "markdown",
		"--stdout",
		"--request-delay", "0s",
	})
	if code != exitOK || !strings.Contains(out.String(), "# Slack conversation export — C22222222") {
		t.Fatalf("exit=%d stdout=%s stderr=%s", code, out.String(), errOut.String())
	}
}

func TestExportPermalinkSelectsWorkspaceAndRange(t *testing.T) {
	app, out, errOut := testApp()
	_ = app.store.Save(t.Context(), auth.Credential{
		WorkspaceHost: "example.slack.com",
		WorkspaceID:   "T11111111",
	})
	code := app.run(t.Context(), []string{
		"conversation", "export",
		"https://example.slack.com/archives/C22222222/p1786455295071869",
		"--to", "2026-08-12T10:00:00Z",
		"--format", "markdown",
		"--stdout",
		"--request-delay", "0s",
	})
	if code != exitOK || !strings.Contains(out.String(), "# Slack conversation export — C22222222") {
		t.Fatalf("exit=%d stdout=%s stderr=%s", code, out.String(), errOut.String())
	}
}

func TestExportPermalinkPassesExactInclusiveRangeToSlack(t *testing.T) {
	app, _, errOut := testApp()
	fixedNow := time.Date(2026, 8, 12, 9, 32, 6, 437000000, time.UTC)
	app.now = func() time.Time { return fixedNow }
	recorder := &rangeRecordingClient{}
	app.clientFactory = func(auth.Credential, clientOptions) slackAPI { return recorder }
	_ = app.store.Save(t.Context(), auth.Credential{
		WorkspaceHost: "example.slack.com",
		WorkspaceID:   "T11111111",
	})
	code := app.run(t.Context(), []string{
		"conversation", "export",
		"https://example.slack.com/archives/C22222222/p1786455295071869",
		"--format", "markdown",
		"--stdout",
		"--request-delay", "0s",
	})
	if code != exitOK {
		t.Fatalf("exit=%d stderr=%s", code, errOut.String())
	}
	if recorder.oldest != "1786455295.071869" || recorder.latest != "1786527126.437000" {
		t.Fatalf("history range = [%s, %s]", recorder.oldest, recorder.latest)
	}
}

func TestExportPermalinkRejectsConflicts(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want string
	}{
		{
			name: "workspace mismatch",
			args: []string{"--workspace", "other.slack.com"},
			want: "does not match",
		},
		{
			name: "all conflicts",
			args: []string{"--all"},
			want: "--all",
		},
		{
			name: "from conflicts",
			args: []string{"--from", "now-1d"},
			want: "--from",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			app, _, errOut := testApp()
			_ = app.store.Save(t.Context(), auth.Credential{WorkspaceHost: "example.slack.com", WorkspaceID: "T11111111"})
			args := []string{
				"conversation", "export",
				"https://example.slack.com/archives/C22222222/p1786455295071869",
				"--format", "markdown",
				"--stdout",
			}
			args = append(args, tc.args...)
			if code := app.run(t.Context(), args); code == exitOK || !strings.Contains(errOut.String(), tc.want) {
				t.Fatalf("stderr = %q, want containing %q", errOut.String(), tc.want)
			}
		})
	}
}

func TestExportPermalinkMissingCredentialsNamesWorkspace(t *testing.T) {
	app, _, errOut := testApp()
	code := app.run(t.Context(), []string{
		"conversation", "export",
		"https://example.slack.com/archives/C22222222/p1786455295071869",
		"--format", "markdown",
		"--stdout",
	})
	if code == exitOK || !strings.Contains(errOut.String(), "example.slack.com") {
		t.Fatalf("exit=%d stderr=%s", code, errOut.String())
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
