package fail

import (
	"fmt"
)

type Error struct {
	Code    Code
	Message string
}

func (e *Error) Error() string {
	return fmt.Sprintf("[%s] %s", e.Code, e.Message)
}

func New(code Code, msg string) error {
	return &Error{Code: code, Message: msg}
}
