package datadogapi

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/Alechan/ai-resources/tools/ddctl/src/internal/fail"
)

func apiErrorFromResponse(status int, body string) *fail.Error {
	details := redactResponseBody(body)
	message := fmt.Sprintf("HTTP %d", status)
	serverErrors := parseDatadogAPIErrors(body)
	if len(serverErrors) > 0 {
		message = serverErrors[0]
	} else if msg := plainTextAPIErrorMessage(body); msg != "" {
		message = msg
	}
	return &fail.Error{
		Category:   "api",
		Message:    message,
		Action:     "inspect API response",
		Details:    details,
		HTTPStatus: status,
		Errors:     serverErrors,
	}
}

func parseDatadogAPIErrors(body string) []string {
	var parsed struct {
		Errors []string `json:"errors"`
		Error  string   `json:"error"`
	}
	if err := json.Unmarshal([]byte(body), &parsed); err != nil {
		return nil
	}
	serverErrors := append([]string(nil), parsed.Errors...)
	if len(serverErrors) == 0 && parsed.Error != "" {
		serverErrors = []string{parsed.Error}
	}
	return serverErrors
}

func plainTextAPIErrorMessage(body string) string {
	trimmed := strings.TrimSpace(body)
	if trimmed == "" || json.Valid([]byte(trimmed)) {
		return ""
	}
	if len(trimmed) > 200 {
		return trimmed[:200] + "…"
	}
	return trimmed
}
