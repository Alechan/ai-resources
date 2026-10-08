package datadogapi

import (
	"fmt"
	"io"
	"regexp"
	"strings"
	"time"
)

const redactedLogEventsPlaceholder = "[response body redacted: may contain log events]"

var (
	emailLikeRe = regexp.MustCompile(`[A-Za-z0-9._%+\-]+@[A-Za-z0-9.\-]+\.[A-Za-z]{2,}`)
	tokenKeyRe  = regexp.MustCompile(`"(?:_authentication_token|dd_csrf_token|authorization|api_key|application_key)"\s*:\s*"[^"]*"`)
)

// DebugLogger receives redacted request diagnostics when --debug is set.
type DebugLogger interface {
	Printf(format string, args ...any)
}

type stderrDebugLogger struct {
	w io.Writer
}

func NewDebugLogger(w io.Writer) DebugLogger {
	if w == nil {
		return nil
	}
	return stderrDebugLogger{w: w}
}

func (l stderrDebugLogger) Printf(format string, args ...any) {
	fmt.Fprintf(l.w, "debug: "+format+"\n", args...)
}

func redactResponseBody(body string) string {
	body = strings.TrimSpace(body)
	if body == "" {
		return ""
	}
	lower := strings.ToLower(body)
	if strings.Contains(lower, `"events"`) || strings.Contains(lower, `"hitcount"`) {
		return redactedLogEventsPlaceholder
	}
	body = tokenKeyRe.ReplaceAllString(body, `"$1":"[redacted]"`)
	body = emailLikeRe.ReplaceAllString(body, "[redacted-email]")
	if len(body) > 512 {
		return body[:512] + "…"
	}
	return body
}

func (c *Client) logRequest(method, path string, status int, started time.Time) {
	if c.debug == nil {
		return
	}
	c.debug.Printf("%s %s -> %d (%s)", method, path, status, time.Since(started).Round(time.Millisecond))
}
