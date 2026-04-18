package service

import (
	"context"

	"github.com/Alechan/ai-resources/tools/ddctl/src/internal/auth"
	"github.com/Alechan/ai-resources/tools/ddctl/src/internal/datadogapi"
)

// DoctorReport summarizes the health of ddctl prerequisites.
type DoctorReport struct {
	CredentialStore  string `json:"credential_store"`
	CredentialsFound bool   `json:"credentials_found"`
	SessionCookies   int    `json:"session_cookies"`
	DataDogReachable bool   `json:"datadog_reachable"`
	Note             string `json:"note,omitempty"`
}

// DoctorService checks Keychain credentials and DataDog connectivity.
type DoctorService struct {
	auth *auth.KeychainProvider
	dd   *datadogapi.Client
}

// NewDoctorService creates a DoctorService.
func NewDoctorService(auth *auth.KeychainProvider, dd *datadogapi.Client) *DoctorService {
	return &DoctorService{auth: auth, dd: dd}
}

// Run performs all doctor checks and returns a DoctorReport.
// It never returns an error; failures are reflected in the report fields.
func (s *DoctorService) Run(ctx context.Context) (DoctorReport, error) {
	r := DoctorReport{CredentialStore: s.auth.Path()}

	if cookies, err := s.auth.Cookies(); err == nil {
		r.CredentialsFound = true
		r.SessionCookies = len(cookies)
	} else {
		r.Note = `run "ddctl init" to store your DataDog session`
	}

	if s.dd.Probe(ctx, "/api/v1/validate") {
		r.DataDogReachable = true
	}

	if r.CredentialsFound && r.SessionCookies > 0 && r.DataDogReachable {
		r.Note = "all checks passed"
	}

	return r, nil
}
