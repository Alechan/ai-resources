package fail

import "errors"

type Error struct {
	Category    string `json:"type"`
	Message     string `json:"message"`
	Action      string `json:"action,omitempty"`
	Details     string `json:"details,omitempty"`
	Resource    string `json:"resource,omitempty"`
	WidgetIndex string `json:"widget_index,omitempty"`
	WidgetTitle string `json:"widget_title,omitempty"`
	QueryIndex  *int   `json:"query_index,omitempty"`
}

func (e *Error) Error() string { return e.Message }

func (e *Error) Envelope() map[string]any {
	out := map[string]any{
		"error": map[string]any{
			"type":    e.Category,
			"message": e.Message,
		},
	}
	m := out["error"].(map[string]any)
	if e.Action != "" {
		m["action"] = e.Action
	}
	if e.Details != "" {
		m["details"] = e.Details
	}
	if e.Resource != "" {
		m["resource"] = e.Resource
	}
	if e.WidgetIndex != "" {
		m["widget_index"] = e.WidgetIndex
	}
	if e.WidgetTitle != "" {
		m["widget_title"] = e.WidgetTitle
	}
	if e.QueryIndex != nil {
		m["query_index"] = *e.QueryIndex
	}
	return out
}

func NewValidation(msg, action string) *Error {
	return &Error{Category: "validation", Message: msg, Action: action}
}

func NewResourceValidation(resource, msg, action string) *Error {
	return &Error{Category: "validation", Resource: resource, Message: msg, Action: action}
}

func NewQueryValidation(resource, widgetIndex, widgetTitle string, queryIndex int, msg, action string) *Error {
	idx := queryIndex
	return &Error{
		Category:    "validation",
		Resource:    resource,
		Message:     msg,
		Action:      action,
		WidgetIndex: widgetIndex,
		WidgetTitle: widgetTitle,
		QueryIndex:  &idx,
	}
}

func NewConfig(msg, action string) *Error {
	return &Error{Category: "config", Message: msg, Action: action}
}
func NewAuth(msg, action string) *Error {
	return &Error{Category: "auth", Message: msg, Action: action}
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
	case "auth":
		return CodeAuth
	case "network":
		return CodeNetwork
	default:
		return CodeAPI
	}
}
