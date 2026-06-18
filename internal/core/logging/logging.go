package logging

import (
	"log/slog"
	"os"
)

// Logger is MockTrue's logging facade. It embeds *slog.Logger so all standard
// slog methods are available, and adds Named which returns a new Logger
// tagged with a module name (the receiver is unchanged). The effective log
// level is controlled by the handler passed to New, per slog conventions.
type Logger struct {
	*slog.Logger
}

// New constructs a Logger over the given handler. The handler (e.g. a
// TextHandler built with HandlerOptions{Level: ...}) governs level filtering.
func New(handler slog.Handler) *Logger {
	return &Logger{Logger: slog.New(handler)}
}

// Named returns a new Logger with a "module" attribute. The receiver is not
// modified (immutability).
func (l *Logger) Named(name string) *Logger {
	return &Logger{Logger: l.With(slog.String("module", name))}
}

// Default returns a reasonable development Logger writing to stderr at debug
// level with a text handler.
func Default() *Logger {
	return &Logger{Logger: slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))}
}
