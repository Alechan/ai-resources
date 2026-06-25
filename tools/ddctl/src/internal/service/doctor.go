package service

import (
	"context"
	"net/http"
	"time"

	"github.com/Alechan/ai-resources/tools/ddctl/src/internal/datadogapi"
)

type doctorAuthProvider interface {
	Path() string
	Cookies() ([]*http.Cookie, error)
}

// DoctorReport summarizes the health of ddctl prerequisites.
type DoctorReport struct {
	CredentialStore  string `json:"credential_store"`
	CredentialsFound bool   `json:"credentials_found"`
	SessionCookies   int    `json:"session_cookies"`
	DataDogReachable bool   `json:"datadog_reachable"`
	AuthQueryValid   bool   `json:"auth_query_valid"`
	Note             string `json:"note,omitempty"`
}

// DoctorService checks Keychain credentials and DataDog connectivity.
type DoctorService struct {
	auth doctorAuthProvider
	dd   *datadogapi.Client
}

// NewDoctorService creates a DoctorService.
func NewDoctorService(auth doctorAuthProvider, dd *datadogapi.Client) *DoctorService {
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
		if err := s.runAuthQuery(ctx); err == nil {
			r.AuthQueryValid = true
		} else {
			r.Note = `auth query failed; cookies may be stale, run "ddctl init" and paste a fresh cURL`
		}
	}

	if r.CredentialsFound && r.SessionCookies > 0 && r.DataDogReachable && r.AuthQueryValid {
		r.Note = "all checks passed"
	}

	return r, nil
}

func (s *DoctorService) runAuthQuery(ctx context.Context) error {
	now := time.Now().UnixMilli()
	body := map[string]any{
		"list": map[string]any{
			"columns": []map[string]any{
				{"field": map[string]any{"path": "status_line"}},
				{"field": map[string]any{"path": "timestamp"}},
				{"field": map[string]any{"path": "host"}},
				{"field": map[string]any{"path": "service"}},
				{"field": map[string]any{"path": "content"}},
			},
			"sorts":                []map[string]any{{"time": map[string]any{"order": "desc"}}},
			"limit":                1,
			"time":                 map[string]any{"from": now - 5*60*1000, "to": now},
			"includeEvents":        true,
			"includeEventContents": true,
			"computeCount":         true,
			"indexes":              []string{"*"},
			"executionInfo":        map[string]any{},
		},
		"querySourceId": "logs_explorer",
	}
	var out map[string]any
	return s.dd.Post(ctx, "/api/v1/logs-analytics/list?type=logs", body, &out)
}
