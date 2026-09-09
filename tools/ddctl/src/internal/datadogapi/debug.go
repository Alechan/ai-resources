package datadogapi

import (
	"fmt"
	"io"
	"strings"
	"time"
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

func redactDetails(body string) string {
	body = strings.TrimSpace(body)
	if body == "" {
		return ""
	}
	lower := strings.ToLower(body)
	if strings.Contains(lower, `"events"`) || strings.Contains(lower, `"hitcount"`) {
		return "[response body redacted: may contain log events]"
	}
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
