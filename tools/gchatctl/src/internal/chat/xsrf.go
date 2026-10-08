package chat

import (
	"context"
	"io"
	"net/http"
	"regexp"

	"github.com/Alechan/ai-resources/tools/gchatctl/src/internal/auth"
)

const xsrfHeaderName = "x-framework-xsrf-token"

var wizXSRFPattern = regexp.MustCompile(`"SNlM0e"\s*:\s*"([^"]+)"`)

func ParseXSRF(body []byte) (string, bool) {
	match := wizXSRFPattern.FindSubmatch(body)
	if len(match) != 2 {
		return "", false
	}
	token := string(match[1])
	if len(token) < 8 {
		return "", false
	}
	return token, true
}

func (c *Client) WithPersist(persist func(context.Context, auth.Credential) error) *Client {
	c.persist = persist
	return c
}

func (c *Client) applyXSRF(token string) {
	if c.credential.ExtraHeaders == nil {
		c.credential.ExtraHeaders = map[string]auth.Secret{}
	}
	c.credential.ExtraHeaders[xsrfHeaderName] = auth.NewSecret(token)
}

func (c *Client) refreshXSRF(ctx context.Context) (string, bool) {
	account := c.credential.AccountKey
	if account == "" {
		account = "0"
	}
	host := c.credential.RequestHost
	if host == "" {
		host = "chat.google.com"
	}
	endpoint := requestScheme(host) + "://" + host + "/u/" + account + "/"
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return "", false
	}
	request.Header.Set("Accept", "text/html")
	request.Header.Set("Cookie", c.credential.Cookie.Reveal())
	request.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	response, err := c.httpClient.Do(request)
	if err != nil {
		return "", false
	}
	defer response.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(response.Body, 8<<20))
	if err != nil {
		return "", false
	}
	token, ok := ParseXSRF(raw)
	if !ok {
		return "", false
	}
	c.applyXSRF(token)
	if c.persist != nil {
		_ = c.persist(ctx, c.credential)
	}
	return token, true
}
