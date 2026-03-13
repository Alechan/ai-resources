package fail

import (
	"context"
	"errors"
	"testing"
)

type timeoutErr struct{}

func (timeoutErr) Error() string   { return "i/o timeout" }
func (timeoutErr) Timeout() bool   { return true }
func (timeoutErr) Temporary() bool { return false }

func TestExitCode(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want int
	}{
		{name: "nil", err: nil, want: CodeOK},
		{name: "validation", err: NewValidation("bad", "fix"), want: CodeValidation},
		{name: "config", err: NewConfig("bad", "fix"), want: CodeValidation},
		{name: "auth", err: NewAuth("bad", "fix"), want: CodeAuth},
		{name: "scope", err: NewScope("bad", "fix"), want: CodeAuth},
		{name: "network", err: NewNetwork("bad", "fix"), want: CodeNetwork},
		{name: "api", err: NewAPI("bad", "fix", ""), want: CodeAPI},
		{name: "unknown category", err: &Error{Category: "other", Message: "x"}, want: CodeAPI},
		{name: "generic error", err: errors.New("x"), want: CodeAPI},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ExitCode(tt.err); got != tt.want {
				t.Fatalf("ExitCode() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestMapNetworkOrAPI(t *testing.T) {
	tests := []struct {
		name         string
		err          error
		wantCategory string
	}{
		{name: "nil", err: nil, wantCategory: ""},
		{name: "net timeout interface", err: timeoutErr{}, wantCategory: "network"},
		{name: "deadline exceeded text", err: errors.New("context deadline exceeded"), wantCategory: "network"},
		{name: "timeout text", err: errors.New("operation timeout"), wantCategory: "network"},
		{name: "no such host", err: errors.New("dial tcp: no such host"), wantCategory: "network"},
		{name: "connection refused", err: errors.New("dial tcp: connection refused"), wantCategory: "network"},
		{name: "context deadline exact", err: errors.New(context.DeadlineExceeded.Error()), wantCategory: "network"},
		{name: "generic", err: errors.New("bad response"), wantCategory: "api"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := MapNetworkOrAPI(tt.err)
			if tt.wantCategory == "" {
				if got != nil {
					t.Fatalf("MapNetworkOrAPI() = %v, want nil", got)
				}
				return
			}
			var fe *Error
			if !errors.As(got, &fe) {
				t.Fatalf("MapNetworkOrAPI() error type = %T, want *fail.Error", got)
			}
			if fe.Category != tt.wantCategory {
				t.Fatalf("category = %q, want %q", fe.Category, tt.wantCategory)
			}
		})
	}
}
