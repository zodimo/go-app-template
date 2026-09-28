package logging

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/zodimo/go-app-template/internal/core"
	"github.com/zodimo/go-maybe"
)

// loggerTestContext is a minimal core.ConfigContext so Setup can validate a
// LogConfig in this test without importing the config package (which would form
// an import cycle).
type loggerTestContext struct{}

func (loggerTestContext) GetEnvPrefix() string                      { return "GOCLI_LOG" }
func (loggerTestContext) GetPrefix() string                         { return "log" }
func (c loggerTestContext) WithEnvPrefix(string) core.ConfigContext { return c }
func (c loggerTestContext) WithPrefix(string) core.ConfigContext    { return c }

func TestGetLogger_NilSafeAfterFailedSetup(t *testing.T) {
	ResetLoggerState()
	// Establish the "logging was never successfully set up" precondition
	// deterministically, regardless of any earlier test that ran Setup in this
	// process. These are the package globals the nil-safe paths read.
	logger = nil
	slogHandler = nil
	slogStdErrHandler = nil
	t.Cleanup(ResetLoggerState)

	// A zero-value LogConfig is NOT a reliable failure: Validate() dereferences
	// its nil embedded core.ConfigContext and panics before Setup can return an
	// error. Instead build a fully valid config whose log directory cannot be
	// created (a file blocks the path), which makes Setup return an error.
	blocker := filepath.Join(t.TempDir(), "not-a-directory")
	if err := os.WriteFile(blocker, []byte("x"), 0o644); err != nil {
		t.Fatalf("write blocker: %v", err)
	}

	cfg := LogConfig{
		Directory:  maybe.Some(filepath.Join(blocker, "logs")),
		Filename:   maybe.Some("app.log"),
		MaxSize:    maybe.Some(50),
		MaxBackups: maybe.Some(3),
		MaxAge:     maybe.Some(30),
		Compress:   maybe.Some(true),
		Debug:      maybe.Some(false),
	}
	cfg.WithConfigContext(loggerTestContext{})

	if err := Setup(cfg); err == nil {
		t.Fatal("Setup should fail when the log directory cannot be created")
	}

	if GetLogger() == nil {
		t.Fatal("GetLogger() returned nil after failed Setup")
	}
	if nl := GetLogger().NewLogger("x"); nl == nil {
		t.Fatal(`GetLogger().NewLogger("x") returned nil after failed Setup`)
	}

	if nl := (&Logger{}).NewLogger("x"); nl == nil {
		t.Fatal(`(&Logger{}).NewLogger("x") returned nil`)
	}
	if w := (&Logger{}).With("k", "v"); w == nil {
		t.Fatal(`(&Logger{}).With("k", "v") returned nil`)
	}

	// A discarded call through the nil-safe logger must not panic.
	GetLogger().NewLogger("x").Info("hi")
}

func TestNewLogger_NilSafeBeforeSetup(t *testing.T) {
	ResetLoggerState()
	slogHandler = nil
	slogStdErrHandler = nil
	t.Cleanup(ResetLoggerState)

	l := NewLogger("x")
	if l == nil {
		t.Fatal(`NewLogger("x") returned nil before Setup`)
	}
	l.Info("hi")

	se := NewStdErrLogger("x")
	if se == nil {
		t.Fatal(`NewStdErrLogger("x") returned nil before Setup`)
	}
	se.Info("hi")
}
