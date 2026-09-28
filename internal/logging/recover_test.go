package logging

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRecoverPanic_NilLogger(t *testing.T) {
	// Ensure logger is not initialized
	ResetLoggerState()

	// Redirect the panic log to a temp dir so the test does not leave a
	// panic log in the package directory (mirrors SingleStackTrace).
	SetupPanicRecovery("test-app", t.TempDir())

	// Capture stderr
	old := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	var code uint8
	func() {
		defer RecoverPanic("test", &code, nil)
		panic("deliberate panic")
	}()

	_ = w.Close()
	os.Stderr = old

	// A recovered panic must contribute the panic exit code (2).
	if code != 2 {
		t.Fatalf("expected exit code 2, got %d", code)
	}

	// The panic message must be reported on stderr.
	got, _ := io.ReadAll(r)
	if !strings.Contains(string(got), "deliberate panic") {
		t.Fatalf("stderr missing panic message: %q", got)
	}
}

func TestRecoverPanic_SingleStackTrace(t *testing.T) {
	tmpDir := t.TempDir()
	SetupPanicRecovery("test-app", tmpDir)
	ResetLoggerState()

	func() {
		defer RecoverPanic("test", nil, nil)
		panic("stack trace test")
	}()

	entries, _ := filepath.Glob(filepath.Join(tmpDir, "*-panic-test-*.log"))
	if len(entries) != 1 {
		t.Fatalf("expected 1 panic log, got %d", len(entries))
	}
	data, _ := os.ReadFile(entries[0])
	content := string(data)

	// The panic log should contain the panic message
	if !strings.Contains(content, "stack trace test") {
		t.Fatal("panic log missing panic message")
	}
}

func TestSetupPanicRecovery_SanitizesAppName(t *testing.T) {
	SetupPanicRecovery("/usr/local/bin/../../evil/myapp", "/tmp/panics")
	if panicAppName != "myapp" {
		t.Fatalf("expected 'myapp', got '%s'", panicAppName)
	}

	SetupPanicRecovery("/path/to/myapp", "/tmp/panics")
	if panicAppName != "myapp" {
		t.Fatalf("expected 'myapp', got '%s'", panicAppName)
	}
}

func TestRecoverPanic_SetsExitCode(t *testing.T) {
	SetupPanicRecovery("test-app", t.TempDir())
	ResetLoggerState()

	var code uint8
	func() {
		defer RecoverPanic("test", &code, nil)
		panic("exit code test")
	}()

	if code == 0 {
		t.Fatal("expected non-zero exit code after recovered panic, got 0")
	}
	if code == 1 {
		t.Fatalf("expected panic code 2, got ordinary error code %d", code)
	}
	if code != 2 {
		t.Fatalf("expected exit code 2, got %d", code)
	}
}
