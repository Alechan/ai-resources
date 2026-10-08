package fail

import "errors"

type Error struct {
	Category string `json:"category"`
	Message  string `json:"message"`
	Action   string `json:"action,omitempty"`
	Details  string `json:"details,omitempty"`
}

func (e *Error) Error() string { return e.Message }

func NewValidation(msg, action string) *Error {
	return &Error{Category: "validation", Message: msg, Action: action}
}
func NewConfig(msg, action string) *Error {
	return &Error{Category: "config", Message: msg, Action: action}
}
func NewAuth(msg, action string) *Error {
	return &Error{Category: "auth", Message: msg, Action: action}
}
func NewScope(msg, action string) *Error {
	return &Error{Category: "scope", Message: msg, Action: action}
}
func NewNetwork(msg, action string) *Error {
	return &Error{Category: "network", Message: msg, Action: action}
}
func NewAPI(msg, action, details string) *Error {
	return &Error{Category: "api", Message: msg, Action: action, Details: details}
}

func ExitCode(err error) int {
	if err == nil {
		return CodeOK
	}
	var e *Error
	if !errors.As(err, &e) {
		return CodeAPI
	}
	switch e.Category {
	case "validation", "config":
		return CodeValidation
	case "auth", "scope":
		return CodeAuth
	case "network":
		return CodeNetwork
	default:
		return CodeAPI
	}
}
