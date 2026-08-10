package service

import (
	"context"
	"errors"
	"testing"

	"github.com/Alechan/ai-resources/tools/slackctl/src/internal/auth"
	"github.com/Alechan/ai-resources/tools/slackctl/src/internal/slack"
)

type doctorClient struct {
	auth slack.AuthInfo
	err  error
}

func (c doctorClient) AuthTest(context.Context) (slack.AuthInfo, error) { return c.auth, c.err }
func (c doctorClient) ConversationsProbe(context.Context) error         { return c.err }

func TestDoctorSuccess(t *testing.T) {
	store := auth.NewMemoryStore()
	credential := auth.Credential{WorkspaceHost: "alpha.slack.com", WorkspaceID: "T11111111", Token: auth.NewSecret("generated-token"), Cookie: auth.NewSecret("generated-cookie")}
	if err := store.Save(t.Context(), credential); err != nil {
		t.Fatal(err)
	}
	report, err := Doctor(t.Context(), store, "alpha.slack.com", func(auth.Credential) DoctorClient {
		return doctorClient{auth: slack.AuthInfo{UserID: "U11111111", UserName: "Example User", WorkspaceID: "T11111111"}}
	})
	if err != nil {
		t.Fatal(err)
	}
	if !report.CredentialsFound || !report.WorkspaceReachable || !report.AuthValid || !report.ConversationAPI || report.AuthenticatedUser != "Example User" {
		t.Fatalf("report = %#v", report)
	}
}

func TestDoctorMissingAndInvalidCredentials(t *testing.T) {
	if report, err := Doctor(t.Context(), auth.NewMemoryStore(), "alpha.slack.com", nil); err == nil || report.CredentialsFound {
		t.Fatalf("missing report=%#v error=%v", report, err)
	}
	store := auth.NewMemoryStore()
	_ = store.Save(t.Context(), auth.Credential{WorkspaceHost: "alpha.slack.com"})
	report, err := Doctor(t.Context(), store, "alpha.slack.com", func(auth.Credential) DoctorClient {
		return doctorClient{err: errors.New("synthetic auth failure")}
	})
	if err == nil || report.AuthValid {
		t.Fatalf("invalid report=%#v error=%v", report, err)
	}
}
