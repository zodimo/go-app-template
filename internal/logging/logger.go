package logging

import (
	"fmt"
	"log/slog"
)

type Logger struct {
	*slog.Logger
	component string
}

func (l *Logger) NewLogger(component string) *Logger {
	if l == nil || l.Logger == nil {
		return noopLogger(component)
	}
	if l.component == "root" {
		return &Logger{
			Logger:    l.Logger.With("component", component),
			component: component,
		}
	}
	return &Logger{
		Logger:    l.Logger.With("component", fmt.Sprintf("%s->%s", l.component, component)),
		component: fmt.Sprintf("%s->%s", l.component, component),
	}
}

func GetLogger() *Logger {
	if logger == nil || logger.Logger == nil {
		// Logging was never successfully set up (for example, the configured
		// log directory could not be created). Return a discard logger so
		// callers can log safely instead of dereferencing a nil pointer.
		return noopLogger("root")
	}
	return logger
}

func (l *Logger) With(args ...any) *Logger {
	if l == nil || l.Logger == nil {
		return noopLogger("root")
	}
	if len(args) == 0 {
		return l
	}
	return &Logger{
		component: l.component,
		Logger:    l.Logger.With(args...),
	}
}

func NewLogger(component string) *Logger {
	if slogHandler == nil {
		return noopLogger(component)
	}
	return &Logger{
		Logger:    slog.New(slogHandler).With("component", component),
		component: component,
	}
}

func NewStdErrLogger(component string) *Logger {
	if slogStdErrHandler == nil {
		return noopLogger(component)
	}
	return &Logger{
		Logger:    slog.New(slogStdErrHandler).With("component", component),
		component: component,
	}
}

func NewNoopLogger() *Logger {
	return &Logger{
		Logger: slog.New(slog.DiscardHandler),
	}
}

// noopLogger returns a logger that discards every record while still being
// safe to chain NewLogger/With on. Used when logging has not been configured.
func noopLogger(component string) *Logger {
	return &Logger{
		Logger:    slog.New(slog.DiscardHandler).With("component", component),
		component: component,
	}
}
