package fail

import (
	"context"
	"testing"
)

func TestMapNetworkOrAPI_ContextDeadline(t *testing.T) {
	t.Parallel()

	err := MapNetworkOrAPI(context.DeadlineExceeded)
	if err == nil {
		t.Fatal("expected error")
	}
	if ExitCode(err) != CodeNetwork {
		t.Fatalf("ExitCode() = %d, want %d", ExitCode(err), CodeNetwork)
	}
}
