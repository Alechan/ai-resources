package auth

import (
	"fmt"
	"runtime"

	"github.com/Alechan/ai-resources/tools/ddctl/src/internal/fail"
)

// RequireDarwin returns an error when ddctl is not running on macOS.
func RequireDarwin() error {
	if runtime.GOOS != "darwin" {
		return fail.NewConfig(
			fmt.Sprintf("ddctl requires macOS for Keychain session storage (GOOS=%s)", runtime.GOOS),
			"run ddctl on macOS",
		)
	}
	return nil
}
