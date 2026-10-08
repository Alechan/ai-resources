package googleapi

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"github.com/Alechan/ai-resources/tools/gdrivectl/src/internal/fail"
)

type DocsClient interface {
	Probe(ctx context.Context, token string) error
	DocTabs(ctx context.Context, token string, req DocTabsRequest) (map[string]any, error)
}

type DocsHTTPClient struct {
	http *http.Client
}

func NewDocsClient(httpClient *http.Client) *DocsHTTPClient {
	return &DocsHTTPClient{http: httpClient}
}

func (c *DocsHTTPClient) Probe(ctx context.Context, token string) error {
	u := "https://docs.googleapis.com/v1/documents/invalid?fields=documentId"
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := c.http.Do(req)
	if err != nil {
		return fail.MapNetworkOrAPI(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound || resp.StatusCode == http.StatusOK {
		return nil
	}
	if resp.StatusCode == http.StatusUnauthorized {
		return fail.NewAuth("docs probe unauthorized", "run gcloud auth login")
	}
	if resp.StatusCode == http.StatusForbidden {
		return fail.NewScope("docs probe forbidden", "run: gcloud auth login --enable-gdrive-access --update-adc")
	}
	b, _ := io.ReadAll(resp.Body)
	return fail.NewAPI(fmt.Sprintf("docs probe failed with status %d", resp.StatusCode), "check Docs API access", string(b))
}

func (c *DocsHTTPClient) DocTabs(ctx context.Context, token string, req DocTabsRequest) (map[string]any, error) {
	fields := "documentId,title,tabs(tabProperties(tabId,title,index,parentTabId,nestingLevel),childTabs(tabProperties(tabId,title,index,parentTabId,nestingLevel),childTabs))"
	u := fmt.Sprintf("https://docs.googleapis.com/v1/documents/%s?includeTabsContent=true&fields=%s", url.PathEscape(req.ID), url.QueryEscape(fields))
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
			return nil, fail.NewAuth("doc-tabs unauthorized", "run gcloud auth login")
		}
		if resp.StatusCode == http.StatusForbidden {
			return nil, fail.NewScope("doc-tabs forbidden", "run: gcloud auth login --enable-gdrive-access --update-adc")
		}
		return nil, fail.NewAPI(fmt.Sprintf("doc-tabs failed with status %d", resp.StatusCode), "verify document id and docs API access", string(b))
	}
	var out map[string]any
	if err := json.Unmarshal(b, &out); err != nil {
		return nil, fail.NewAPI("failed to parse doc-tabs response", "retry with --debug", err.Error())
	}
	return out, nil
}
