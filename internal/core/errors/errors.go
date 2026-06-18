package errors

import (
	stderrors "errors"
	"fmt"
)

// Code is a stable, machine-readable error classification surfaced to the
// frontend and logs. Keep values lower_snake identifiers.
type Code string

const (
	CodeInvalid      Code = "invalid"      // input validation failed
	CodeNotFound     Code = "not_found"
	CodeConflict     Code = "conflict"
	CodeIO           Code = "io"
	CodePlatform     Code = "platform"
	CodeModuleInit   Code = "module_init"
	CodeModuleStart  Code = "module_start"
	CodeModuleStop   Code = "module_stop"
	CodeInternal     Code = "internal"
)

// Error is MockTrue's canonical error type. Instances are immutable after
// construction: use WithField to derive a new Error carrying extra context
// rather than mutating an existing one.
type Error struct {
	Code    Code
	Message string
	Cause   error           // may be nil
	Fields  map[string]any  // may be nil; never mutate in place
}

// Error formats as "code: message" or "code: message: cause".
func (e *Error) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s: %s: %s", e.Code, e.Message, e.Cause)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

// Unwrap returns the underlying cause for errors.Is/errors.As traversal.
func (e *Error) Unwrap() error {
	return e.Cause
}

// New constructs an Error with no cause. Empty code or message is a
// programmer error and panics.
func New(code Code, message string) *Error {
	if code == "" {
		panic("errors.New: code must not be empty")
	}
	if message == "" {
		panic("errors.New: message must not be empty")
	}
	return &Error{Code: code, Message: message}
}

// Wrap constructs an Error wrapping cause. A nil cause is a programmer error
// and panics — use New for causeless errors.
func Wrap(code Code, message string, cause error) *Error {
	if cause == nil {
		panic("errors.Wrap: cause must not be nil")
	}
	if code == "" {
		panic("errors.Wrap: code must not be empty")
	}
	if message == "" {
		panic("errors.Wrap: message must not be empty")
	}
	return &Error{Code: code, Message: message, Cause: cause}
}

// WithField returns a new Error with an additional context field. The
// receiver is left unchanged (immutability).
func (e *Error) WithField(key string, val any) *Error {
	fields := make(map[string]any, len(e.Fields)+1)
	for k, v := range e.Fields {
		fields[k] = v
	}
	fields[key] = val
	return &Error{
		Code:    e.Code,
		Message: e.Message,
		Cause:   e.Cause,
		Fields:  fields,
	}
}

// AsCode extracts the Code from err if an *Error is present in its chain;
// otherwise it returns CodeInternal (the safe default). A nil err yields
// CodeInternal.
func AsCode(err error) Code {
	if err == nil {
		return CodeInternal
	}
	var target *Error
	if stderrors.As(err, &target) {
		return target.Code
	}
	return CodeInternal
}
