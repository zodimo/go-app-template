package logging

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime/debug"
	"time"
)

var (
	panicLogLocation = ""
	panicAppName     = filepath.Base(os.Args[0])
)

func SetupPanicLogLocation(logLocation string) {
	panicLogLocation = logLocation
}

func SetupPanicRecovery(appName, logLocation string) {
	panicAppName = filepath.Base(appName)
	panicLogLocation = logLocation
}

func RecoverPanic(componentName string, exitCode *uint8, cleanup func()) {
	if r := recover(); r != nil {
		if exitCode != nil {
			*exitCode = 2
		}
		stack := debug.Stack()
		fmt.Fprintf(os.Stderr, "Panic in %s: %v\nStack Trace:\n%s\n", componentName, r, stack)

		filePath := filepath.Join(panicLogLocation, fmt.Sprintf(
			"%s-panic-%s-%s.log", panicAppName, componentName, time.Now().Format("20060102-150405")))

		if initialized.Load() {
			GetLogger().Error("panic recovered",
				"component", componentName,
				"error", r,
				"stack", string(stack),
				"panic_log", filePath,
			)
		}

		if file, err := os.Create(filePath); err == nil {
			_, _ = fmt.Fprintf(file, "Panic in %s: %v\n\n", componentName, r)
			_, _ = fmt.Fprintf(file, "Time: %s\n\n", time.Now().Format(time.RFC3339))
			_, _ = fmt.Fprintf(file, "Stack Trace:\n%s\n", stack)
			_ = file.Sync()
			_ = file.Close()
		} else {
			fmt.Fprintf(os.Stderr, "Error creating panic log file at %q: %v\n", filePath, err)
		}

		if cleanup != nil {
			cleanup()
		}
	}
}
