package service

import (
	"context"
	"os"

	"github.com/Alechan/ai-resources/tools/ddctl/src/internal/auth"
	"github.com/Alechan/ai-resources/tools/ddctl/src/internal/datadogapi"
)

// DoctorReport summarizes the health of ddctl prerequisites.
type DoctorReport struct {
	CookiesPath      string `json:"cookies_path"`
	CookiesFileFound bool   `json:"cookies_file_found"`
	SessionCookies   int    `json:"session_cookies"`
	DataDogReachable bool   `json:"datadog_reachable"`
	Note             string `json:"note,omitempty"`
}

// DoctorService checks Chrome cookies and DataDog connectivity.
type DoctorService struct {
	cookies *auth.ChromeCookieProvider
	dd      *datadogapi.Client
}

// NewDoctorService creates a DoctorService.
func NewDoctorService(cookies *auth.ChromeCookieProvider, dd *datadogapi.Client) *DoctorService {
	return &DoctorService{cookies: cookies, dd: dd}
}

// Run performs all doctor checks and returns a DoctorReport.
// It never returns an error; failures are reflected in the report fields.
func (s *DoctorService) Run(ctx context.Context) (DoctorReport, error) {
	r := DoctorReport{CookiesPath: s.cookies.Path()}

	if _, err := os.Stat(s.cookies.Path()); err == nil {
		r.CookiesFileFound = true
	}

	if cookies, err := s.cookies.Cookies(); err == nil {
		r.SessionCookies = len(cookies)
	}

	if s.dd.Probe(ctx, "/api/v1/validate") {
		r.DataDogReachable = true
	}

	if r.CookiesFileFound && r.SessionCookies > 0 && r.DataDogReachable {
		r.Note = "all checks passed"
	}

	return r, nil
}
