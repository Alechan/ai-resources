package chat

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/Alechan/ai-resources/tools/gchatctl/src/internal/auth"
	"github.com/Alechan/ai-resources/tools/gchatctl/src/internal/members"
	"github.com/Alechan/ai-resources/tools/gchatctl/src/internal/search"
	"github.com/Alechan/ai-resources/tools/gchatctl/src/internal/spaces"
	"github.com/Alechan/ai-resources/tools/gchatctl/src/internal/topics"
)

type ErrorKind string

const (
	ErrorAuth      ErrorKind = "auth"
	ErrorTransport ErrorKind = "transport"
	ErrorAPI       ErrorKind = "api"
)

type APIError struct {
	Kind       ErrorKind
	StatusCode int
}

func (e *APIError) Error() string {
	return fmt.Sprintf("Google Chat API %s error (HTTP %d)", e.Kind, e.StatusCode)
}

type Client struct {
	credential auth.Credential
	httpClient *http.Client
	persist    func(context.Context, auth.Credential) error
	refreshed  bool
}

func NewClient(credential auth.Credential, httpClient *http.Client) *Client {
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 30 * time.Second}
	}
	return &Client{credential: credential, httpClient: httpClient}
}

func IsAuth(err error) bool {
	var api *APIError
	return errors.As(err, &api) && api.Kind == ErrorAuth
}

func ProbeStatus(err error) string {
	if err == nil {
		return "ok"
	}
	var api *APIError
	if !errors.As(err, &api) {
		return "http_other"
	}
	switch {
	case api.Kind == ErrorTransport:
		return "transport"
	case api.StatusCode == http.StatusUnauthorized:
		return "http_401"
	case api.StatusCode == http.StatusForbidden:
		return "http_403"
	default:
		return "http_other"
	}
}

func (c *Client) ListTopics(ctx context.Context) (topics.Page, []byte, error) {
	if !strings.Contains(c.credential.URLPath, "list_topics") {
		return topics.Page{}, nil, errors.New("stored capture is not list_topics; copy a list_topics cURL and rerun gchatctl init")
	}
	if c.credential.Body.Reveal() == "" {
		return topics.Page{}, nil, errors.New("stored capture has no body; copy a list_topics cURL and rerun gchatctl init")
	}
	return c.do(ctx, c.credential.Body.Reveal(), "")
}

func (c *Client) ListTopicsPage(ctx context.Context, opts topics.RequestOptions) (topics.Page, error) {
	body, err := topics.RewriteBody(c.credential.Body.Reveal(), opts)
	if err != nil {
		return topics.Page{}, err
	}
	page, _, err := c.do(ctx, body, opts.SpaceID)
	return page, err
}

func (c *Client) do(ctx context.Context, body, spaceID string) (topics.Page, []byte, error) {
	if body == "" {
		return topics.Page{}, nil, errors.New("stored capture has no body; copy a list_topics cURL and rerun gchatctl init")
	}
	raw, err := c.call(ctx, body, spaceID)
	if err != nil {
		return topics.Page{}, nil, err
	}
	page, err := topics.Parse(raw)
	if err != nil {
		return topics.Page{}, raw, err
	}
	return page, raw, nil
}

func (c *Client) Heartbeat(ctx context.Context) error {
	body, ok := topics.HeartbeatBody(c.credential.Body.Reveal())
	if !ok {
		return nil
	}
	_, err := c.callPath(ctx, "/u/ACCOUNT/api/heartbeat", body, "")
	return err
}

func (c *Client) PaginatedWorld(ctx context.Context) (spaces.World, error) {
	template, ok := c.credential.OperationTemplate("paginated_world")
	if !ok || template.Body == "" || !strings.Contains(template.URLPath, "paginated_world") {
		return spaces.World{}, errors.New("stored paginated_world capture is missing; run gchatctl learn paginated_world")
	}
	raw, err := c.callPath(ctx, template.URLPath, template.Body, "")
	if err != nil {
		return spaces.World{}, err
	}
	return spaces.Parse(raw)
}

func (c *Client) SearchMessages(ctx context.Context, query string) (search.Result, error) {
	template, ok := c.credential.OperationTemplate("search.messages")
	if !ok || template.Body == "" || !strings.Contains(strings.ToLower(template.URLPath+template.RawQuery), "batchexecute") {
		return search.Result{}, errors.New("stored search capture is missing; run gchatctl learn search.messages")
	}
	body, err := search.RewriteQuery(template.Body, query)
	if err != nil {
		return search.Result{}, err
	}
	path := template.URLPath
	if template.RawQuery != "" {
		path += "?" + template.RawQuery
	}
	raw, err := c.callWith(ctx, path, body, "", template.ContentType)
	if err != nil {
		return search.Result{}, err
	}
	result, err := search.Parse(raw)
	if err != nil {
		return search.Result{}, err
	}
	result.Query = query
	return result, nil
}

func (c *Client) ListMembers(ctx context.Context, spaceID string) (members.Roster, error) {
	raw, err := c.callRewritten(ctx, "list_members", spaceID, members.RewriteSpace)
	if err != nil {
		return members.Roster{}, err
	}
	return members.Parse(raw)
}

func (c *Client) GetGroup(ctx context.Context, spaceID string) (spaces.Group, error) {
	raw, err := c.callRewritten(ctx, "get_group", spaceID, spaces.RewriteSpace)
	if err != nil {
		return spaces.Group{}, err
	}
	return spaces.ParseGroup(raw)
}

func (c *Client) callRewritten(ctx context.Context, operation, spaceID string, rewrite func(string, string) (string, error)) ([]byte, error) {
	template, ok := c.credential.OperationTemplate(operation)
	if !ok || template.Body == "" || !strings.Contains(template.URLPath, operation) {
		return nil, fmt.Errorf("stored %s capture is missing; run gchatctl learn %s", operation, operation)
	}
	body, err := rewrite(template.Body, spaceID)
	if err != nil {
		return nil, err
	}
	path := template.URLPath
	if template.RawQuery != "" {
		path += "?" + template.RawQuery
	}
	return c.callWith(ctx, path, body, spaceID, template.ContentType)
}

func (c *Client) call(ctx context.Context, body, spaceID string) ([]byte, error) {
	return c.callPath(ctx, "", body, spaceID)
}

func (c *Client) callPath(ctx context.Context, urlPath, body, spaceID string) ([]byte, error) {
	return c.callWith(ctx, urlPath, body, spaceID, "")
}

func (c *Client) callWith(ctx context.Context, urlPath, body, spaceID, contentType string) ([]byte, error) {
	raw, err := c.roundTrip(ctx, urlPath, body, spaceID, contentType)
	if err != nil && IsAuth(err) && !c.refreshed {
		if _, ok := c.refreshXSRF(ctx); ok {
			c.refreshed = true
			raw, err = c.roundTrip(ctx, urlPath, body, spaceID, contentType)
		}
	}
	return raw, err
}

func (c *Client) roundTrip(ctx context.Context, urlPath, body, spaceID, contentType string) ([]byte, error) {
	account := c.credential.AccountKey
	if account == "" {
		account = "0"
	}
	path := urlPath
	if path == "" {
		path = c.credential.URLPath
	}
	if path == "" || (urlPath == "" && !strings.Contains(path, "list_topics")) {
		path = "/u/ACCOUNT/api/list_topics"
	}
	path = strings.Replace(path, "/u/ACCOUNT", "/u/"+account, 1)
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	host := c.credential.RequestHost
	if host == "" {
		host = "chat.google.com"
	}
	endpoint := requestScheme(host) + "://" + host + path
	if !strings.Contains(path, "?") && c.credential.RawQuery != "" {
		endpoint += "?" + c.credential.RawQuery
	}
	method := c.credential.Method
	if method == "" {
		method = http.MethodPost
	}
	request, err := http.NewRequestWithContext(ctx, method, endpoint, strings.NewReader(body))
	if err != nil {
		return nil, err
	}
	if contentType == "" {
		contentType = c.credential.ContentType
	}
	if contentType == "" {
		contentType = "application/json+protobuf"
	}
	referer := "https://chat.google.com/"
	if spaceID != "" {
		referer = "https://chat.google.com/room/" + spaceID
	}
	request.Header.Set("Accept", "*/*")
	request.Header.Set("Content-Type", contentType)
	request.Header.Set("Cookie", c.credential.Cookie.Reveal())
	request.Header.Set("Origin", "https://chat.google.com")
	request.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	if authz := c.credential.Authorization.Reveal(); authz != "" {
		request.Header.Set("Authorization", authz)
	}
	for name, value := range c.credential.ExtraHeaders {
		if value.Reveal() == "" {
			continue
		}
		if spaceID != "" && strings.EqualFold(name, "x-goog-pageid") {
			continue
		}
		request.Header.Set(name, value.Reveal())
	}
	request.Header.Set("Referer", referer)
	if spaceID != "" {
		request.Header.Set("x-goog-chat-space-id", spaceID)
	}
	response, err := c.httpClient.Do(request)
	if err != nil {
		return nil, &APIError{Kind: ErrorTransport}
	}
	defer response.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(response.Body, 32<<20))
	if err != nil {
		return nil, &APIError{Kind: ErrorTransport, StatusCode: response.StatusCode}
	}
	if response.StatusCode == http.StatusUnauthorized || response.StatusCode == http.StatusForbidden {
		return nil, &APIError{Kind: ErrorAuth, StatusCode: response.StatusCode}
	}
	if response.StatusCode < 200 || response.StatusCode > 299 {
		return nil, &APIError{Kind: ErrorAPI, StatusCode: response.StatusCode}
	}
	return bytes.Clone(raw), nil
}

func requestScheme(host string) string {
	if strings.HasPrefix(host, "127.0.0.1:") || strings.HasPrefix(host, "localhost:") {
		return "http"
	}
	return "https"
}
