package service

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/Alechan/ai-resources/tools/ddctl/src/internal/datadogapi"
)

type doctorRoundTripper func(*http.Request) (*http.Response, error)

func (f doctorRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

type doctorAuthProviderStub struct {
	path       string
	cookies    []*http.Cookie
	cookiesErr error
}

func (d doctorAuthProviderStub) Path() string { return d.path }

func (d doctorAuthProviderStub) Cookies() ([]*http.Cookie, error) {
	if d.cookiesErr != nil {
		return nil, d.cookiesErr
	}
	return d.cookies, nil
}

func TestDoctorRun_FailsAuthValidationWhenQueryReturns401(t *testing.T) {
	t.Parallel()

	httpClient := &http.Client{
		Transport: doctorRoundTripper(func(req *http.Request) (*http.Response, error) {
			if req.URL.Path == "/api/v1/validate" {
				return &http.Response{
					StatusCode: http.StatusOK,
					Header:     make(http.Header),
					Body:       io.NopCloser(strings.NewReader(`{}`)),
				}, nil
			}
			return &http.Response{
				StatusCode: http.StatusUnauthorized,
				Header:     make(http.Header),
				Body:       io.NopCloser(strings.NewReader(`{"errors":["unauthorized"]}`)),
			}, nil
		}),
	}

	dd := datadogapi.NewClient(httpClient, "datadoghq.com", doctorAuthProviderStub{
		path:    "stub-path",
		cookies: []*http.Cookie{{Name: "dogweb", Value: "x"}, {Name: "dd_csrf_token", Value: "y"}},
	})

	svc := NewDoctorService(doctorAuthProviderStub{
		path:    "stub-path",
		cookies: []*http.Cookie{{Name: "dogweb", Value: "x"}, {Name: "dd_csrf_token", Value: "y"}},
	}, dd)

	report, err := svc.Run(context.Background())
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if report.AuthQueryValid {
		t.Fatalf("AuthQueryValid = true, want false")
	}
	if report.Note == "" {
		t.Fatalf("Note is empty, want actionable failure note")
	}
}

func TestDoctorRun_PassesWhenAuthQuerySucceeds(t *testing.T) {
	t.Parallel()

	httpClient := &http.Client{
		Transport: doctorRoundTripper(func(req *http.Request) (*http.Response, error) {
			if req.URL.Path == "/api/v1/validate" {
				return &http.Response{
					StatusCode: http.StatusOK,
					Header:     make(http.Header),
					Body:       io.NopCloser(strings.NewReader(`{}`)),
				}, nil
			}
			return &http.Response{
				StatusCode: http.StatusOK,
				Header:     make(http.Header),
				Body:       io.NopCloser(strings.NewReader(`{"hitCount":1,"result":{"events":[],"paging":{"after":""}}}`)),
			}, nil
		}),
	}

	authStub := doctorAuthProviderStub{
		path:    "stub-path",
		cookies: []*http.Cookie{{Name: "dogweb", Value: "x"}, {Name: "dd_csrf_token", Value: "y"}},
	}

	dd := datadogapi.NewClient(httpClient, "datadoghq.com", authStub)
	svc := NewDoctorService(authStub, dd)

	report, err := svc.Run(context.Background())
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if !report.AuthQueryValid {
		t.Fatalf("AuthQueryValid = false, want true")
	}
}

func TestDoctorRun_AuthQueryNotExecutedWithoutCredentials(t *testing.T) {
	t.Parallel()

	httpClient := &http.Client{
		Transport: doctorRoundTripper(func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusOK,
				Header:     make(http.Header),
				Body:       io.NopCloser(strings.NewReader(`{}`)),
			}, nil
		}),
	}

	authStub := doctorAuthProviderStub{
		path:       "stub-path",
		cookiesErr: fmt.Errorf("no credentials"),
	}
	dd := datadogapi.NewClient(httpClient, "datadoghq.com", authStub)
	svc := NewDoctorService(authStub, dd)

	report, err := svc.Run(context.Background())
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if report.AuthQueryValid {
		t.Fatalf("AuthQueryValid = true, want false when credentials are missing")
	}
}
