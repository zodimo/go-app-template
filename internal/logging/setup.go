package logging

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"

	"gopkg.in/natefinch/lumberjack.v2"
)

var (
	initOnce          sync.Once
	initialized       atomic.Bool
	logger            *Logger
	slogger           *slog.Logger
	slogHandler       slog.Handler
	slogStdErrHandler slog.Handler
	slogLeveler       = new(slog.LevelVar)
)

func Setup(config LogConfig) error {
	var setupErr error

	if !config.IsValid() {
		return fmt.Errorf("config has errors")
	}

	initOnce.Do(func() {
		if err := os.MkdirAll(config.Directory.UnwrapUnsafe(), 0755); err != nil {
			setupErr = fmt.Errorf("failed to create log directory: %w", err)
			return
		}

		// Panic logs must land in the configured log directory, not the process CWD.
		SetupPanicLogLocation(config.Directory.UnwrapUnsafe())

		logFile := filepath.Join(config.Directory.UnwrapUnsafe(), config.Filename.UnwrapUnsafe())
		debug := config.Debug.UnwrapUnsafe()

		fileWriter := &lumberjack.Logger{
			Filename:   logFile,
			MaxSize:    config.MaxSize.UnwrapUnsafe(),
			MaxBackups: config.MaxBackups.UnwrapUnsafe(),
			MaxAge:     config.MaxAge.UnwrapUnsafe(),
			Compress:   config.Compress.UnwrapUnsafe(),
		}

		logWriter := io.MultiWriter(fileWriter)
		stderrLogWriter := io.MultiWriter(fileWriter, os.Stderr)

		slogLeveler.Set(slog.LevelInfo)
		if debug {
			slogLeveler.Set(slog.LevelDebug)
		}

		slogHandler = slog.NewJSONHandler(logWriter, &slog.HandlerOptions{
			Level:     slogLeveler,
			AddSource: debug,
		})
		slogStdErrHandler = slog.NewJSONHandler(stderrLogWriter, &slog.HandlerOptions{
			Level:     slogLeveler,
			AddSource: debug,
		})

		if debug {
			slogHandler = slogStdErrHandler
		}

		slogger = slog.New(slogHandler)
		if debug {
			slogger = slog.New(slogStdErrHandler)
		}
		slog.SetDefault(slogger)

		logger = NewLogger("root")
		initialized.Store(true)
	})

	return setupErr
}

func SetDefaultLogger(l *Logger) {
	if l == nil {
		panic("Logger cannot be nil")
	}
	logger = l
	slog.SetDefault(logger.Logger)
}

func Initialized() bool {
	return initialized.Load()
}

func ResetLoggerState() {
	initOnce = sync.Once{}
	initialized.Store(false)
}
