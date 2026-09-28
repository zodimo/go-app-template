package logging

import (
	"testing"

	"github.com/zodimo/go-app-template/internal/core"
	"github.com/zodimo/go-app-template/internal/identity"
	"github.com/zodimo/go-maybe"
)

// panicDirTestContext is a minimal core.ConfigContext so Setup can validate a
// LogConfig without importing the config package (which would form a cycle).
type panicDirTestContext struct{}

func (panicDirTestContext) GetEnvPrefix() string                      { return identity.EnvPrefix + "_LOG" }
func (panicDirTestContext) GetPrefix() string                         { return "log" }
func (c panicDirTestContext) WithEnvPrefix(string) core.ConfigContext { return c }
func (c panicDirTestContext) WithPrefix(string) core.ConfigContext    { return c }

// directory from the loaded log configuration, so recovered panics land in the
// configured directory rather than the process CWD.
func TestSetup_SetsPanicLogLocation(t *testing.T) {
	ResetLoggerState()
	dir := t.TempDir()

	cfg := LogConfig{
		Directory:  maybe.Some(dir),
		Filename:   maybe.Some("app.log"),
		MaxSize:    maybe.Some(50),
		MaxBackups: maybe.Some(3),
		MaxAge:     maybe.Some(30),
		Compress:   maybe.Some(true),
		Debug:      maybe.Some(false),
	}
	cfg.WithConfigContext(panicDirTestContext{})
	if err := Setup(cfg); err != nil {
		t.Fatalf("Setup returned error: %v", err)
	}
	if panicLogLocation != dir {
		t.Fatalf("panicLogLocation = %q, want %q", panicLogLocation, dir)
	}
}
