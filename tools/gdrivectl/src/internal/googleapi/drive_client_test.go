package googleapi

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Alechan/ai-resources/tools/gdrivectl/src/internal/fail"
)

func TestDriveSearchSuccessFixture(t *testing.T) {
	payload := readFixture(t, "drive/search_success.json")
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/drive/v3/files" {
			t.Fatalf("path = %q, want /drive/v3/files", r.URL.Path)
		}
		_, _ = w.Write(payload)
	}))
	defer srv.Close()

	c := NewDriveClient(testHTTPClient(srv.URL))
	got, err := c.Search(context.Background(), "tok", SearchRequest{
		Query: "name contains 'RFC'",
	})
	if err != nil {
		t.Fatalf("Search() error = %v", err)
	}
	files, ok := got["files"].([]any)
	if !ok || len(files) != 1 {
		t.Fatalf("files = %#v, want single element", got["files"])
	}
}

func TestDriveFileMetaSuccessFixture(t *testing.T) {
	payload := readFixture(t, "drive/file_meta_success.json")
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/drive/v3/files/file-123" {
			t.Fatalf("path = %q, want /drive/v3/files/file-123", r.URL.Path)
		}
		_, _ = w.Write(payload)
	}))
	defer srv.Close()

	c := NewDriveClient(testHTTPClient(srv.URL))
	got, err := c.FileMeta(context.Background(), "tok", FileMetaRequest{ID: "file-123"})
	if err != nil {
		t.Fatalf("FileMeta() error = %v", err)
	}
	if got["id"] != "file-123" {
		t.Fatalf("id = %#v, want file-123", got["id"])
	}
}

func TestDriveExportSuccess(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/drive/v3/files/doc-1/export" {
			t.Fatalf("path = %q, want /drive/v3/files/doc-1/export", r.URL.Path)
		}
		_, _ = w.Write([]byte("exported body"))
	}))
	defer srv.Close()

	c := NewDriveClient(testHTTPClient(srv.URL))
	got, err := c.ExportDoc(context.Background(), "tok", ExportRequest{ID: "doc-1", MIME: "text/plain"})
	if err != nil {
		t.Fatalf("ExportDoc() error = %v", err)
	}
	if string(got) != "exported body" {
		t.Fatalf("ExportDoc() body = %q, want %q", string(got), "exported body")
	}
}

func TestDriveUploadSuccess(t *testing.T) {
	payload := []byte(`{"id":"up-1","name":"report.txt","mimeType":"text/plain"}`)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("method = %q, want %q", r.Method, http.MethodPost)
		}
		if r.URL.Path != "/upload/drive/v3/files" {
			t.Fatalf("path = %q, want /upload/drive/v3/files", r.URL.Path)
		}
		q := r.URL.Query()
		if q.Get("uploadType") != "multipart" {
			t.Fatalf("uploadType = %q, want multipart", q.Get("uploadType"))
		}
		if q.Get("supportsAllDrives") != "true" {
			t.Fatalf("supportsAllDrives = %q, want true", q.Get("supportsAllDrives"))
		}
		if q.Get("fields") != "id,name,mimeType" {
			t.Fatalf("fields = %q, want id,name,mimeType", q.Get("fields"))
		}

		mediaType, params, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
		if err != nil {
			t.Fatalf("parse content-type: %v", err)
		}
		if mediaType != "multipart/related" {
			t.Fatalf("content-type media = %q, want multipart/related", mediaType)
		}
		boundary := params["boundary"]
		if boundary == "" {
			t.Fatalf("missing multipart boundary")
		}
		mr := multipart.NewReader(r.Body, boundary)

		metaPart, err := mr.NextPart()
		if err != nil {
			t.Fatalf("next metadata part: %v", err)
		}
		if got := metaPart.Header.Get("Content-Type"); got != "application/json; charset=UTF-8" {
			t.Fatalf("metadata content-type = %q", got)
		}
		metaBytes, err := io.ReadAll(metaPart)
		if err != nil {
			t.Fatalf("read metadata part: %v", err)
		}
		var meta map[string]any
		if err := json.Unmarshal(metaBytes, &meta); err != nil {
			t.Fatalf("metadata json: %v", err)
		}
		if meta["name"] != "report.txt" {
			t.Fatalf("metadata name = %#v, want report.txt", meta["name"])
		}
		parents, ok := meta["parents"].([]any)
		if !ok || len(parents) != 1 || parents[0] != "folder-1" {
			t.Fatalf("metadata parents = %#v, want [folder-1]", meta["parents"])
		}

		mediaPart, err := mr.NextPart()
		if err != nil {
			t.Fatalf("next media part: %v", err)
		}
		if got := mediaPart.Header.Get("Content-Type"); got != "text/plain" {
			t.Fatalf("media content-type = %q, want text/plain", got)
		}
		mediaBytes, err := io.ReadAll(mediaPart)
		if err != nil {
			t.Fatalf("read media part: %v", err)
		}
		if string(mediaBytes) != "hello drive" {
			t.Fatalf("media body = %q, want hello drive", string(mediaBytes))
		}
		if _, err := mr.NextPart(); !errors.Is(err, io.EOF) {
			t.Fatalf("unexpected extra multipart part err=%v", err)
		}

		_, _ = w.Write(payload)
	}))
	defer srv.Close()

	c := NewDriveClient(testHTTPClient(srv.URL))
	got, err := c.Upload(context.Background(), "tok", UploadRequest{
		Name:     "report.txt",
		ParentID: "folder-1",
		MIME:     "text/plain",
		Fields:   "id,name,mimeType",
		Content:  []byte("hello drive"),
	})
	if err != nil {
		t.Fatalf("Upload() error = %v", err)
	}
	if got["id"] != "up-1" {
		t.Fatalf("id = %#v, want up-1", got["id"])
	}
}

func TestDriveUploadStatusErrorMapping(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		fixture    string
		wantCat    string
	}{
		{name: "unauthorized auth", statusCode: http.StatusUnauthorized, fixture: "drive/error_401.json", wantCat: "auth"},
		{name: "forbidden scope", statusCode: http.StatusForbidden, fixture: "drive/error_403.json", wantCat: "scope"},
		{name: "server api", statusCode: http.StatusInternalServerError, fixture: "drive/error_500.json", wantCat: "api"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			payload := readFixture(t, tt.fixture)
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.statusCode)
				_, _ = w.Write(payload)
			}))
			defer srv.Close()

			c := NewDriveClient(testHTTPClient(srv.URL))
			_, err := c.Upload(context.Background(), "tok", UploadRequest{
				Name:    "x.txt",
				MIME:    "text/plain",
				Content: []byte("x"),
			})
			assertFailCategoryStrict(t, err, tt.wantCat)
		})
	}
}

func TestDriveStatusErrorMapping(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		fixture    string
		wantCat    string
	}{
		{name: "unauthorized auth", statusCode: http.StatusUnauthorized, fixture: "drive/error_401.json", wantCat: "auth"},
		{name: "forbidden scope", statusCode: http.StatusForbidden, fixture: "drive/error_403.json", wantCat: "scope"},
		{name: "server api", statusCode: http.StatusInternalServerError, fixture: "drive/error_500.json", wantCat: "api"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			payload := readFixture(t, tt.fixture)
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.statusCode)
				_, _ = w.Write(payload)
			}))
			defer srv.Close()

			c := NewDriveClient(testHTTPClient(srv.URL))
			_, err := c.Search(context.Background(), "tok", SearchRequest{Query: "x"})
			assertFailCategoryStrict(t, err, tt.wantCat)
		})
	}
}

func TestDriveSearchMalformedJSONMapsToAPI(t *testing.T) {
	payload := readFixture(t, "drive/malformed.json")
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(payload)
	}))
	defer srv.Close()

	c := NewDriveClient(testHTTPClient(srv.URL))
	_, err := c.Search(context.Background(), "tok", SearchRequest{Query: "x"})
	assertFailCategoryStrict(t, err, "api")
}

func TestDriveNetworkErrorMapsToNetwork(t *testing.T) {
	c := NewDriveClient(testHTTPClientWithErr(errors.New("dial tcp: no such host")))
	_, err := c.Search(context.Background(), "tok", SearchRequest{Query: "x"})
	assertFailCategoryStrict(t, err, "network")
}

func assertFailCategoryStrict(t *testing.T, err error, want string) {
	t.Helper()
	var fe *fail.Error
	if !errors.As(err, &fe) {
		t.Fatalf("error type = %T, want *fail.Error (err=%v)", err, err)
	}
	if fe.Category != want {
		t.Fatalf("category = %q, want %q", fe.Category, want)
	}
}
