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

// Bootstrap builds a command logger using the shared VERBOSE env convention.
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

// New builds the default project logger.
func New(command string, verbose bool) (*zap.Logger, error) {
	return newWithSyncer(command, verbose, nil)
}

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

// Sync flushes buffered logs and ignores common stdio sync errors.
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
