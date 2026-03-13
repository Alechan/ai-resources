package googleapi

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"net/url"
	"strconv"

	"github.com/Alechan/ai-resources/tools/gdrivectl/src/internal/fail"
)

type DriveClient interface {
	Probe(ctx context.Context, token string) error
	Search(ctx context.Context, token string, req SearchRequest) (map[string]any, error)
	FileMeta(ctx context.Context, token string, req FileMetaRequest) (map[string]any, error)
	ExportDoc(ctx context.Context, token string, req ExportRequest) ([]byte, error)
	Upload(ctx context.Context, token string, req UploadRequest) (map[string]any, error)
}

type DriveHTTPClient struct {
	http *http.Client
}

func NewDriveClient(httpClient *http.Client) *DriveHTTPClient {
	return &DriveHTTPClient{http: httpClient}
}

func (c *DriveHTTPClient) Probe(ctx context.Context, token string) error {
	u := "https://www.googleapis.com/drive/v3/files/root?fields=id"
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := c.http.Do(req)
	if err != nil {
		return fail.MapNetworkOrAPI(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusUnauthorized {
		return fail.NewAuth("drive probe unauthorized", "run gcloud auth login")
	}
	if resp.StatusCode == http.StatusForbidden {
		return fail.NewScope("drive probe forbidden", "run: gcloud auth login --enable-gdrive-access --update-adc")
	}
	if resp.StatusCode >= 300 {
		b, _ := io.ReadAll(resp.Body)
		return fail.NewAPI(fmt.Sprintf("drive probe failed with status %d", resp.StatusCode), "check Drive API access", string(b))
	}
	return nil
}

func (c *DriveHTTPClient) Search(ctx context.Context, token string, req SearchRequest) (map[string]any, error) {
	v := url.Values{}
	v.Set("q", req.Query)
	if req.Corpora == "" {
		req.Corpora = "allDrives"
	}
	v.Set("corpora", req.Corpora)
	v.Set("includeItemsFromAllDrives", "true")
	v.Set("supportsAllDrives", "true")
	if req.DriveID != "" {
		v.Set("driveId", req.DriveID)
	}
	if req.PageSize <= 0 {
		req.PageSize = 100
	}
	v.Set("pageSize", strconv.Itoa(req.PageSize))
	if req.Fields != "" {
		v.Set("fields", req.Fields)
	}
	u := "https://www.googleapis.com/drive/v3/files?" + v.Encode()
	reqHTTP, _ := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	reqHTTP.Header.Set("Authorization", "Bearer "+token)
	resp, err := c.http.Do(reqHTTP)
	if err != nil {
		return nil, fail.MapNetworkOrAPI(err)
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	if resp.StatusCode == http.StatusUnauthorized {
		return nil, fail.NewAuth("drive search unauthorized", "run gcloud auth login")
	}
	if resp.StatusCode == http.StatusForbidden {
		return nil, fail.NewScope("drive search forbidden", "run: gcloud auth login --enable-gdrive-access --update-adc")
	}
	if resp.StatusCode >= 300 {
		return nil, fail.NewAPI(fmt.Sprintf("drive search failed with status %d", resp.StatusCode), "check query and access", string(b))
	}
	var out map[string]any
	if err := json.Unmarshal(b, &out); err != nil {
		return nil, fail.NewAPI("failed to parse drive search response", "retry with --debug", err.Error())
	}
	return out, nil
}

func (c *DriveHTTPClient) FileMeta(ctx context.Context, token string, req FileMetaRequest) (map[string]any, error) {
	v := url.Values{}
	v.Set("supportsAllDrives", "true")
	if req.Fields != "" {
		v.Set("fields", req.Fields)
	}
	u := fmt.Sprintf("https://www.googleapis.com/drive/v3/files/%s?%s", url.PathEscape(req.ID), v.Encode())
	reqHTTP, _ := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	reqHTTP.Header.Set("Authorization", "Bearer "+token)
	resp, err := c.http.Do(reqHTTP)
	if err != nil {
		return nil, fail.MapNetworkOrAPI(err)
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 300 {
		if resp.StatusCode == http.StatusUnauthorized {
			return nil, fail.NewAuth("file-meta unauthorized", "run gcloud auth login")
		}
		if resp.StatusCode == http.StatusForbidden {
			return nil, fail.NewScope("file-meta forbidden", "run: gcloud auth login --enable-gdrive-access --update-adc")
		}
		return nil, fail.NewAPI(fmt.Sprintf("file-meta failed with status %d", resp.StatusCode), "verify file id and access", string(b))
	}
	var out map[string]any
	if err := json.Unmarshal(b, &out); err != nil {
		return nil, fail.NewAPI("failed to parse file-meta response", "retry with --debug", err.Error())
	}
	return out, nil
}

func (c *DriveHTTPClient) ExportDoc(ctx context.Context, token string, req ExportRequest) ([]byte, error) {
	v := url.Values{}
	v.Set("mimeType", req.MIME)
	u := fmt.Sprintf("https://www.googleapis.com/drive/v3/files/%s/export?%s", url.PathEscape(req.ID), v.Encode())
	reqHTTP, _ := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	reqHTTP.Header.Set("Authorization", "Bearer "+token)
	resp, err := c.http.Do(reqHTTP)
	if err != nil {
		return nil, fail.MapNetworkOrAPI(err)
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 300 {
		if resp.StatusCode == http.StatusUnauthorized {
			return nil, fail.NewAuth("doc-export unauthorized", "run gcloud auth login")
		}
		if resp.StatusCode == http.StatusForbidden {
			return nil, fail.NewScope("doc-export forbidden", "run: gcloud auth login --enable-gdrive-access --update-adc")
		}
		return nil, fail.NewAPI(fmt.Sprintf("doc-export failed with status %d", resp.StatusCode), "verify document id/mime type", string(b))
	}
	return b, nil
}

func (c *DriveHTTPClient) Upload(ctx context.Context, token string, req UploadRequest) (map[string]any, error) {
	v := url.Values{}
	v.Set("uploadType", "multipart")
	v.Set("supportsAllDrives", "true")
	if req.Fields != "" {
		v.Set("fields", req.Fields)
	}
	u := "https://www.googleapis.com/upload/drive/v3/files?" + v.Encode()

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	meta := map[string]any{
		"name": req.Name,
	}
	if req.ParentID != "" {
		meta["parents"] = []string{req.ParentID}
	}
	metaBytes, err := json.Marshal(meta)
	if err != nil {
		return nil, fail.NewAPI("failed to encode upload metadata", "retry with --debug", err.Error())
	}

	metaHeader := textproto.MIMEHeader{}
	metaHeader.Set("Content-Type", "application/json; charset=UTF-8")
	metaPart, err := writer.CreatePart(metaHeader)
	if err != nil {
		return nil, fail.NewAPI("failed to build upload metadata part", "retry with --debug", err.Error())
	}
	if _, err := metaPart.Write(metaBytes); err != nil {
		return nil, fail.NewAPI("failed to write upload metadata part", "retry with --debug", err.Error())
	}

	mediaHeader := textproto.MIMEHeader{}
	mediaHeader.Set("Content-Type", req.MIME)
	mediaPart, err := writer.CreatePart(mediaHeader)
	if err != nil {
		return nil, fail.NewAPI("failed to build upload media part", "retry with --debug", err.Error())
	}
	if _, err := mediaPart.Write(req.Content); err != nil {
		return nil, fail.NewAPI("failed to write upload media part", "retry with --debug", err.Error())
	}
	if err := writer.Close(); err != nil {
		return nil, fail.NewAPI("failed to finalize upload body", "retry with --debug", err.Error())
	}

	reqHTTP, _ := http.NewRequestWithContext(ctx, http.MethodPost, u, body)
	reqHTTP.Header.Set("Authorization", "Bearer "+token)
	reqHTTP.Header.Set("Content-Type", "multipart/related; boundary="+writer.Boundary())

	resp, err := c.http.Do(reqHTTP)
	if err != nil {
		return nil, fail.MapNetworkOrAPI(err)
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 300 {
		if resp.StatusCode == http.StatusUnauthorized {
			return nil, fail.NewAuth("upload unauthorized", "run gcloud auth login")
		}
		if resp.StatusCode == http.StatusForbidden {
			return nil, fail.NewScope("upload forbidden", "run: gcloud auth login --enable-gdrive-access --update-adc")
		}
		return nil, fail.NewAPI(fmt.Sprintf("upload failed with status %d", resp.StatusCode), "verify upload path, parent folder id, and access", string(b))
	}

	var out map[string]any
	if err := json.Unmarshal(b, &out); err != nil {
		return nil, fail.NewAPI("failed to parse upload response", "retry with --debug", err.Error())
	}
	return out, nil
}
