// Package errors provides PortWeave's unified error model.
//
// Error is the canonical error type carrying a stable Code, a human-readable
// Message, an optional Cause, and immutable context Fields. All PortWeave
// packages wrap errors with context using Wrap/WithField rather than mutating
// existing values.
package errors
