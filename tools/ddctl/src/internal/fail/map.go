package fail

import (
	"context"
	"net"
	"strings"
)

func MapNetworkOrAPI(err error) error {
	if err == nil {
		return nil
	}
	if ne, ok := err.(net.Error); ok && ne.Timeout() {
		return NewNetwork("request timed out", "retry with --timeout 60s")
	}
	if strings.Contains(strings.ToLower(err.Error()), "deadline exceeded") || strings.Contains(strings.ToLower(err.Error()), "timeout") {
		return NewNetwork("request timed out", "retry with --timeout 60s")
	}
	if strings.Contains(strings.ToLower(err.Error()), "no such host") {
		return NewNetwork("host resolution failed", "verify network and DNS access")
	}
	if strings.Contains(strings.ToLower(err.Error()), "connection refused") {
		return NewNetwork("connection refused", "verify endpoint connectivity")
	}
	if strings.Contains(strings.ToLower(err.Error()), context.DeadlineExceeded.Error()) {
		return NewNetwork("request timed out", "retry with --timeout 60s")
	}
	return NewAPI(err.Error(), "inspect API response", "")
}
