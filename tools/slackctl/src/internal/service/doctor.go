package service

import (
	"context"
	"errors"

	"github.com/Alechan/ai-resources/tools/slackctl/src/internal/auth"
	"github.com/Alechan/ai-resources/tools/slackctl/src/internal/slack"
)

type DoctorClient interface {
	AuthTest(context.Context) (slack.AuthInfo, error)
	ConversationsProbe(context.Context) error
}

type DoctorReport struct {
	CredentialStore    string `json:"credential_store"`
	CredentialsFound   bool   `json:"credentials_found"`
	WorkspaceReachable bool   `json:"workspace_reachable"`
	AuthValid          bool   `json:"auth_valid"`
	AuthenticatedUser  string `json:"authenticated_user,omitempty"`
	ConversationAPI    bool   `json:"conversation_api"`
}

func Doctor(ctx context.Context, store auth.Store, workspaceHost string, factory func(auth.Credential) DoctorClient) (DoctorReport, error) {
	report := DoctorReport{CredentialStore: "macOS Keychain (slackctl / " + workspaceHost + ")"}
	credential, err := store.Load(ctx, workspaceHost)
	if err != nil {
		return report, errors.New("credentials not found; run slackctl init")
	}
	report.CredentialsFound = true
	client := factory(credential)
	info, err := client.AuthTest(ctx)
	if err != nil {
		var apiError *slack.APIError
		if errors.As(err, &apiError) && apiError.Kind != slack.ErrorTransport {
			report.WorkspaceReachable = true
		}
		return report, errors.New("Slack authentication failed; initialize fresh credentials")
	}
	report.WorkspaceReachable = true
	report.AuthValid = true
	report.AuthenticatedUser = info.UserName
	if report.AuthenticatedUser == "" {
		report.AuthenticatedUser = info.UserID
	}
	if err := client.ConversationsProbe(ctx); err != nil {
		return report, errors.New("Slack conversation API check failed")
	}
	report.ConversationAPI = true
	return report, nil
}
