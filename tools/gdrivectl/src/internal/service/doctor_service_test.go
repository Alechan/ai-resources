package service

import (
	"context"
	"errors"
	"testing"

	"github.com/Alechan/ai-resources/tools/gdrivectl/src/internal/googleapi"
)

type fakeTokenProvider struct {
	token string
	err   error
}

func (f fakeTokenProvider) AccessToken(context.Context) (string, error) {
	return f.token, f.err
}

type fakeDriveClient struct {
	err error
}

func (f fakeDriveClient) Probe(context.Context, string) error { return f.err }
func (f fakeDriveClient) Search(context.Context, string, googleapi.SearchRequest) (map[string]any, error) {
	return nil, errors.New("not implemented")
}
func (f fakeDriveClient) FileMeta(context.Context, string, googleapi.FileMetaRequest) (map[string]any, error) {
	return nil, errors.New("not implemented")
}
func (f fakeDriveClient) ExportDoc(context.Context, string, googleapi.ExportRequest) ([]byte, error) {
	return nil, errors.New("not implemented")
}
func (f fakeDriveClient) Upload(context.Context, string, googleapi.UploadRequest) (map[string]any, error) {
	return nil, errors.New("not implemented")
}

type fakeDocsClient struct {
	err error
}

func (f fakeDocsClient) Probe(context.Context, string) error { return f.err }
func (f fakeDocsClient) DocTabs(context.Context, string, googleapi.DocTabsRequest) (map[string]any, error) {
	return nil, errors.New("not implemented")
}

func TestDoctorRunReportsConfiguredBinaryState(t *testing.T) {
	svc := NewDoctorService("/usr/local/bin/gcloud", true,
		fakeTokenProvider{token: "tok"},
		fakeDriveClient{},
		fakeDocsClient{},
	)
	got, err := svc.Run(context.Background())
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if got.GcloudBin != "/usr/local/bin/gcloud" {
		t.Fatalf("gcloud_bin = %q", got.GcloudBin)
	}
	if !got.GcloudExists {
		t.Fatalf("gcloud_exists = false, want true")
	}
	if !got.TokenOK || !got.DriveOK || !got.DocsOK {
		t.Fatalf("unexpected check booleans: %#v", got)
	}
}

func TestDoctorRunKeepsExistsFalseWhenUnresolved(t *testing.T) {
	svc := NewDoctorService("gcloud", false,
		fakeTokenProvider{err: errors.New("auth fail")},
		fakeDriveClient{},
		fakeDocsClient{},
	)
	got, err := svc.Run(context.Background())
	if err == nil {
		t.Fatalf("Run() err = nil, want non-nil")
	}
	if got.GcloudExists {
		t.Fatalf("gcloud_exists = true, want false")
	}
	if got.TokenOK {
		t.Fatalf("token_ok = true, want false")
	}
}
