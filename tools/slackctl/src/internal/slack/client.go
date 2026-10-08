package slack

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/Alechan/ai-resources/tools/slackctl/src/internal/auth"
)

type ErrorKind string

const (
	ErrorAuth       ErrorKind = "auth"
	ErrorPermission ErrorKind = "permission"
	ErrorNotFound   ErrorKind = "not_found"
	ErrorMalformed  ErrorKind = "malformed"
	ErrorTransport  ErrorKind = "transport"
	ErrorAPI        ErrorKind = "api"
)

type APIError struct {
	Kind       ErrorKind
	StatusCode int
	Code       string
}

func (e *APIError) Error() string {
	if e.Code != "" {
		return fmt.Sprintf("Slack API %s error: %s", e.Kind, e.Code)
	}
	return fmt.Sprintf("Slack API %s error (HTTP %d)", e.Kind, e.StatusCode)
}

type Sleeper func(context.Context, time.Duration) error
type Logger func(string)
type Option func(*Client)

func WithHTTPClient(client *http.Client) Option { return func(c *Client) { c.httpClient = client } }
func WithSleeper(s Sleeper) Option              { return func(c *Client) { c.sleep = s } }
func WithLogger(logger Logger) Option           { return func(c *Client) { c.log = logger } }
func WithRequestDelay(delay time.Duration) Option {
	return func(c *Client) { c.requestDelay = delay }
}
func WithMaxAttempts(attempts int) Option { return func(c *Client) { c.maxAttempts = attempts } }

type Client struct {
	baseURL      string
	credential   auth.Credential
	httpClient   *http.Client
	sleep        Sleeper
	log          Logger
	requestDelay time.Duration
	maxAttempts  int
}

func NewClient(baseURL string, credential auth.Credential, options ...Option) *Client {
	client := &Client{
		baseURL:      strings.TrimSuffix(baseURL, "/"),
		credential:   credential,
		httpClient:   &http.Client{Timeout: 30 * time.Second},
		sleep:        sleepContext,
		log:          func(string) {},
		requestDelay: 500 * time.Millisecond,
		maxAttempts:  3,
	}
	for _, option := range options {
		option(client)
	}
	return client
}

func sleepContext(ctx context.Context, delay time.Duration) error {
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

type envelope struct {
	OK    bool   `json:"ok"`
	Error string `json:"error"`
}

func (c *Client) call(ctx context.Context, method string, values url.Values, output any) ([]byte, error) {
	if values == nil {
		values = make(url.Values)
	}
	values.Set("token", c.credential.Token.Reveal())
	for attempt := 1; attempt <= c.maxAttempts; attempt++ {
		if c.requestDelay > 0 {
			if err := c.sleep(ctx, c.requestDelay); err != nil {
				return nil, err
			}
		}
		request, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/api/"+method, strings.NewReader(values.Encode()))
		if err != nil {
			return nil, errors.New("could not create Slack request")
		}
		request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		request.Header.Set("Cookie", c.credential.Cookie.Reveal())
		c.log(fmt.Sprintf("Slack request method=%s attempt=%d", method, attempt))
		response, err := c.httpClient.Do(request)
		if err != nil {
			if attempt < c.maxAttempts {
				if sleepErr := c.sleep(ctx, backoff(attempt)); sleepErr != nil {
					return nil, sleepErr
				}
				continue
			}
			return nil, &APIError{Kind: ErrorTransport}
		}
		data, readErr := io.ReadAll(io.LimitReader(response.Body, 32<<20))
		_ = response.Body.Close()
		if readErr != nil {
			return nil, &APIError{Kind: ErrorTransport}
		}
		if response.StatusCode == http.StatusTooManyRequests && attempt < c.maxAttempts {
			delay := retryAfter(response.Header.Get("Retry-After"))
			if err := c.sleep(ctx, delay); err != nil {
				return nil, err
			}
			continue
		}
		if response.StatusCode >= 500 && attempt < c.maxAttempts {
			if err := c.sleep(ctx, backoff(attempt)); err != nil {
				return nil, err
			}
			continue
		}
		if response.StatusCode < 200 || response.StatusCode >= 300 {
			kind := ErrorAPI
			switch response.StatusCode {
			case http.StatusUnauthorized:
				kind = ErrorAuth
			case http.StatusForbidden:
				kind = ErrorPermission
			case http.StatusNotFound:
				kind = ErrorNotFound
			}
			return nil, &APIError{Kind: kind, StatusCode: response.StatusCode}
		}
		var status envelope
		if err := json.Unmarshal(data, &status); err != nil {
			return nil, &APIError{Kind: ErrorMalformed, StatusCode: response.StatusCode}
		}
		if !status.OK {
			return nil, classifyAPIError(status.Error)
		}
		if err := json.Unmarshal(data, output); err != nil {
			return nil, &APIError{Kind: ErrorMalformed, StatusCode: response.StatusCode}
		}
		return data, nil
	}
	return nil, &APIError{Kind: ErrorTransport}
}

func retryAfter(value string) time.Duration {
	seconds, err := strconv.Atoi(value)
	if err != nil || seconds < 0 {
		return time.Second
	}
	return time.Duration(seconds) * time.Second
}

func backoff(attempt int) time.Duration {
	return time.Duration(1<<(attempt-1)) * 100 * time.Millisecond
}

func classifyAPIError(code string) error {
	kind := ErrorAPI
	switch code {
	case "invalid_auth", "account_inactive", "token_revoked", "not_authed":
		kind = ErrorAuth
	case "missing_scope", "not_in_channel", "restricted_action":
		kind = ErrorPermission
	case "user_not_found", "channel_not_found":
		kind = ErrorNotFound
	}
	return &APIError{Kind: kind, Code: code}
}

type AuthInfo struct {
	UserID      string `json:"user_id"`
	UserName    string `json:"user"`
	WorkspaceID string `json:"team_id"`
}

func (c *Client) AuthTest(ctx context.Context) (AuthInfo, error) {
	var result AuthInfo
	_, err := c.call(ctx, "auth.test", nil, &result)
	return result, err
}

type Message struct {
	Type        string       `json:"type,omitempty"`
	User        string       `json:"user,omitempty"`
	BotID       string       `json:"bot_id,omitempty"`
	Username    string       `json:"username,omitempty"`
	Text        string       `json:"text,omitempty"`
	Timestamp   string       `json:"ts"`
	ThreadTS    string       `json:"thread_ts,omitempty"`
	ReplyCount  int          `json:"reply_count,omitempty"`
	Attachments []Attachment `json:"attachments,omitempty"`
	Reactions   []Reaction   `json:"reactions,omitempty"`
}

type Reaction struct {
	Name  string   `json:"name"`
	Count int      `json:"count"`
	Users []string `json:"users"`
}

type Attachment struct {
	Text     string `json:"text,omitempty"`
	Fallback string `json:"fallback,omitempty"`
}

type ResponseMetadata struct {
	NextCursor string `json:"next_cursor"`
}

type HistoryPage struct {
	Messages         []Message        `json:"messages"`
	HasMore          bool             `json:"has_more"`
	ResponseMetadata ResponseMetadata `json:"response_metadata"`
}

func (c *Client) History(ctx context.Context, channel, oldest, latest, cursor string, limit int) (HistoryPage, []byte, error) {
	values := paginationValues(channel, cursor, limit)
	values.Set("inclusive", "true")
	if oldest != "" {
		values.Set("oldest", oldest)
	}
	if latest != "" {
		values.Set("latest", latest)
	}
	var page HistoryPage
	raw, err := c.call(ctx, "conversations.history", values, &page)
	return page, raw, err
}

func (c *Client) Replies(ctx context.Context, channel, timestamp, cursor string, limit int) (HistoryPage, []byte, error) {
	values := paginationValues(channel, cursor, limit)
	values.Set("ts", timestamp)
	var page HistoryPage
	raw, err := c.call(ctx, "conversations.replies", values, &page)
	return page, raw, err
}

func paginationValues(channel, cursor string, limit int) url.Values {
	values := url.Values{"channel": {channel}, "limit": {strconv.Itoa(limit)}}
	if cursor != "" {
		values.Set("cursor", cursor)
	}
	return values
}

type UserProfile struct {
	DisplayName string `json:"display_name"`
	RealName    string `json:"real_name"`
}

type User struct {
	ID      string      `json:"id"`
	Name    string      `json:"name"`
	Deleted bool        `json:"deleted"`
	Profile UserProfile `json:"profile"`
}

func (c *Client) UserInfo(ctx context.Context, userID string) (User, error) {
	var result struct {
		User User `json:"user"`
	}
	_, err := c.call(ctx, "users.info", url.Values{"user": {userID}}, &result)
	return result.User, err
}

type Conversation struct {
	ID        string `json:"id"`
	IsChannel bool   `json:"is_channel"`
	IsGroup   bool   `json:"is_group"`
	IsIM      bool   `json:"is_im"`
	IsMPIM    bool   `json:"is_mpim"`
	IsPrivate bool   `json:"is_private"`
}

func (c *Client) ConversationInfo(ctx context.Context, channel string) (Conversation, error) {
	var result struct {
		Channel Conversation `json:"channel"`
	}
	_, err := c.call(ctx, "conversations.info", url.Values{"channel": {channel}}, &result)
	return result.Channel, err
}

func (c *Client) ConversationsProbe(ctx context.Context) error {
	var result struct {
		Channels []Conversation `json:"channels"`
	}
	_, err := c.call(ctx, "conversations.list", url.Values{"limit": {"1"}}, &result)
	return err
}
