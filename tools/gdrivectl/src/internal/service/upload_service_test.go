package service

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/Alechan/ai-resources/tools/gdrivectl/src/internal/fail"
	"github.com/Alechan/ai-resources/tools/gdrivectl/src/internal/googleapi"
)

type uploadTokenProviderStub struct {
	token string
	err   error
}

func (s uploadTokenProviderStub) AccessToken(context.Context) (string, error) {
	return s.token, s.err
}

type uploadDriveStub struct {
	tokenSeen string
	reqSeen   googleapi.UploadRequest
	resp      map[string]any
	err       error
}

func (s *uploadDriveStub) Probe(context.Context, string) error { return errors.New("not implemented") }
func (s *uploadDriveStub) Search(context.Context, string, googleapi.SearchRequest) (map[string]any, error) {
	return nil, errors.New("not implemented")
}
func (s *uploadDriveStub) FileMeta(context.Context, string, googleapi.FileMetaRequest) (map[string]any, error) {
	return nil, errors.New("not implemented")
}
func (s *uploadDriveStub) ExportDoc(context.Context, string, googleapi.ExportRequest) ([]byte, error) {
	return nil, errors.New("not implemented")
}
func (s *uploadDriveStub) Upload(_ context.Context, token string, req googleapi.UploadRequest) (map[string]any, error) {
	s.tokenSeen = token
	s.reqSeen = req
	if s.err != nil {
		return nil, s.err
	}
	if s.resp == nil {
		s.resp = map[string]any{"id": "uploaded-1"}
	}
	return s.resp, nil
}

func TestUploadServiceRunAppliesDefaults(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "notes.txt")
	content := []byte("hello upload")
	if err := os.WriteFile(p, content, 0o644); err != nil {
		t.Fatalf("write temp file: %v", err)
	}

	drive := &uploadDriveStub{}
	svc := NewUploadService(uploadTokenProviderStub{token: "tok"}, drive)

	got, err := svc.Run(context.Background(), UploadRequest{Path: p})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if got["id"] != "uploaded-1" {
		t.Fatalf("response id = %#v, want uploaded-1", got["id"])
	}
	if drive.tokenSeen != "tok" {
		t.Fatalf("token seen = %q, want tok", drive.tokenSeen)
	}
	if drive.reqSeen.Name != "notes.txt" {
		t.Fatalf("upload name = %q, want notes.txt", drive.reqSeen.Name)
	}
	if drive.reqSeen.MIME != "text/plain; charset=utf-8" {
		t.Fatalf("upload mime = %q, want text/plain; charset=utf-8", drive.reqSeen.MIME)
	}
	if !bytes.Equal(drive.reqSeen.Content, content) {
		t.Fatalf("upload content mismatch")
	}
}

func TestUploadServiceRunUsesExplicitFields(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "payload.bin")
	if err := os.WriteFile(p, []byte{0x01, 0x02, 0x03}, 0o644); err != nil {
		t.Fatalf("write temp file: %v", err)
	}

	drive := &uploadDriveStub{}
	svc := NewUploadService(uploadTokenProviderStub{token: "tok"}, drive)

	_, err := svc.Run(context.Background(), UploadRequest{
		Path:     p,
		Name:     "renamed.dat",
		ParentID: "folder-123",
		MIME:     "application/octet-stream",
		Fields:   "id,name",
	})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	if drive.reqSeen.Name != "renamed.dat" {
		t.Fatalf("upload name = %q, want renamed.dat", drive.reqSeen.Name)
	}
	if drive.reqSeen.ParentID != "folder-123" {
		t.Fatalf("parent id = %q, want folder-123", drive.reqSeen.ParentID)
	}
	if drive.reqSeen.MIME != "application/octet-stream" {
		t.Fatalf("mime = %q, want application/octet-stream", drive.reqSeen.MIME)
	}
	if drive.reqSeen.Fields != "id,name" {
		t.Fatalf("fields = %q, want id,name", drive.reqSeen.Fields)
	}
}

func TestUploadServiceRunValidationFailures(t *testing.T) {
	svc := NewUploadService(uploadTokenProviderStub{token: "tok"}, &uploadDriveStub{})

	_, err := svc.Run(context.Background(), UploadRequest{})
	assertFailCategory(t, err, "validation")

	_, err = svc.Run(context.Background(), UploadRequest{Path: t.TempDir()})
	assertFailCategory(t, err, "validation")
}

func TestUploadServiceRunPropagatesTokenError(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "file.txt")
	if err := os.WriteFile(p, []byte("x"), 0o644); err != nil {
		t.Fatalf("write temp file: %v", err)
	}

	tokenErr := fail.NewAuth("token failed", "login")
	svc := NewUploadService(uploadTokenProviderStub{err: tokenErr}, &uploadDriveStub{})
	_, err := svc.Run(context.Background(), UploadRequest{Path: p})
	if !errors.Is(err, tokenErr) {
		t.Fatalf("error = %v, want %v", err, tokenErr)
	}
}

func assertFailCategory(t *testing.T, err error, want string) {
	t.Helper()
	var fe *fail.Error
	if !errors.As(err, &fe) {
		t.Fatalf("error type = %T, want *fail.Error", err)
	}
	if fe.Category != want {
		t.Fatalf("category = %q, want %q", fe.Category, want)
	}
}
