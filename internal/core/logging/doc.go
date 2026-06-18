// Package logging provides MockTrue's structured logging facade over log/slog.
//
// Logger wraps *slog.Logger and supports Named scopes (returns a new logger
// tagged with a module name, leaving the original unchanged).
package logging
