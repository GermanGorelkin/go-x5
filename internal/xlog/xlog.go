// Package xlog provides structured logging helpers built on top of zap.
// It includes Bootstrap for CLI command initialization, a shared Config
// builder, and a safe Sync wrapper that silences common stdio errors.
package xlog

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"syscall"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Bootstrap builds a ready-to-use logger for the given CLI command.
// It reads the VERBOSE environment variable to decide the log level
// (debug when true, info otherwise) and returns the logger, the parsed
// verbose flag, and any error encountered during setup.
func Bootstrap(command string) (*zap.Logger, bool, error) {
	verboseValue := os.Getenv("VERBOSE")
	verbose := false
	if verboseValue != "" {
		value, err := strconv.ParseBool(verboseValue)
		if err != nil {
			return nil, false, fmt.Errorf("failed to parse VERBOSE=%q: %w", verboseValue, err)
		}
		verbose = value
	}

	logger, err := New(command, verbose)
	if err != nil {
		return nil, false, err
	}

	return logger, verbose, nil
}

// Config returns the shared zap configuration for the project.
// When verbose is true the level is set to Debug; otherwise Info.
// Output is JSON-encoded and written to stderr.
func Config(verbose bool) zap.Config {
	level := zap.NewAtomicLevelAt(zap.InfoLevel)
	if verbose {
		level = zap.NewAtomicLevelAt(zap.DebugLevel)
	}

	return zap.Config{
		Level:            level,
		Development:      false,
		Encoding:         "json",
		OutputPaths:      []string{"stderr"},
		ErrorOutputPaths: []string{"stderr"},
		EncoderConfig: zapcore.EncoderConfig{
			TimeKey:        "ts",
			LevelKey:       "level",
			NameKey:        "logger",
			CallerKey:      "caller",
			MessageKey:     "msg",
			StacktraceKey:  "stacktrace",
			LineEnding:     zapcore.DefaultLineEnding,
			EncodeLevel:    zapcore.LowercaseLevelEncoder,
			EncodeTime:     zapcore.ISO8601TimeEncoder,
			EncodeDuration: zapcore.StringDurationEncoder,
			EncodeCaller:   zapcore.ShortCallerEncoder,
		},
	}
}

// New builds the default project logger for the named command.
// It delegates to newWithSyncer with a nil sink so that the zap
// config's OutputPaths (stderr) are used.
func New(command string, verbose bool) (*zap.Logger, error) {
	return newWithSyncer(command, verbose, nil)
}

// newWithSyncer is the internal constructor shared by New and tests.
// When sink is nil the logger is built from the standard Config; otherwise
// the provided WriteSyncer is used as the log destination, which allows
// tests to capture output without touching stderr.
func newWithSyncer(command string, verbose bool, sink zapcore.WriteSyncer) (*zap.Logger, error) {
	cfg := Config(verbose)

	var (
		logger *zap.Logger
		err    error
	)
	if sink == nil {
		logger, err = cfg.Build(zap.AddCaller(), zap.AddStacktrace(zapcore.ErrorLevel))
	} else {
		core := zapcore.NewCore(
			zapcore.NewJSONEncoder(cfg.EncoderConfig),
			sink,
			cfg.Level,
		)
		logger = zap.New(core, zap.AddCaller(), zap.AddStacktrace(zapcore.ErrorLevel))
	}
	if err != nil {
		return nil, fmt.Errorf("failed to build logger: %w", err)
	}
	if command != "" {
		logger = logger.With(zap.String("command", command))
	}

	return logger, nil
}

// Sync flushes any buffered log entries. It silently ignores ENOTTY and
// EINVAL errors that occur when stderr is not a real file (e.g. in
// containers or CI pipelines), preventing noisy exit-time warnings.
func Sync(logger *zap.Logger) {
	if logger == nil {
		return
	}

	err := logger.Sync()
	if err == nil {
		return
	}
	if errors.Is(err, syscall.ENOTTY) || errors.Is(err, syscall.EINVAL) {
		return
	}

	msg := err.Error()
	if strings.Contains(msg, "inappropriate ioctl for device") || strings.Contains(msg, "invalid argument") {
		return
	}
}
