package commands

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/zodimo/go-app-template/internal/logging"
)

// runEntrypointShape mirrors cmd/cli/main.go's defer order in miniature: the
// RecoverPanic defer is registered last (so it runs first), recovers the panic,
// and writes the panic exit code (2) into the named return value. In the real
// entrypoint the os.Exit defer runs after RecoverPanic and observes that value.
func runEntrypointShape() (exitCode uint8) {
	defer logging.RecoverPanic("entrypoint-test", &exitCode, nil)
	panic("entrypoint defer order test")
}

func TestEntrypointDeferOrder_RecoverPanicRunsBeforeExitDefer(t *testing.T) {
	logging.ResetLoggerState()
	// Panic logs must land in a temp dir, not the package/repo directory.
	logging.SetupPanicRecovery("entrypoint-test-app", t.TempDir())
	t.Cleanup(logging.ResetLoggerState)

	var code uint8
	stderr := captureStderr(t, func() {
		code = runEntrypointShape()
	})

	// RecoverPanic runs before the exit defer, so the recovered panic
	// contributes exit code 2 (non-zero and distinct from the ordinary-error 1).
	if code != 2 {
		t.Fatalf("expected panic exit code 2, got %d", code)
	}
	if !strings.Contains(stderr, "entrypoint defer order test") {
		t.Fatalf("captured stderr missing panic message: %q", stderr)
	}

	// The panic report was redirected to the temp dir, so no *-panic-*.log may
	// remain in the working directory.
	if leftovers, _ := filepath.Glob("*-panic-*.log"); len(leftovers) > 0 {
		t.Fatalf("panic log left in working directory: %v", leftovers)
	}
}
