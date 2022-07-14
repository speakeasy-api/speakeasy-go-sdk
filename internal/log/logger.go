package log

import (
	"context"
	"fmt"
	"os"
	"reflect"
	"sync"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var (
	once      sync.Once
	zapLogger *zap.Logger
	config    *zap.Config
)

type contextKey string

func (c contextKey) String() string {
	return "logger-" + string(c)
}

const (
	loggerContextKey = contextKey("context")
)

// From returns the logger associated with the given context.
func From(ctx context.Context) *zap.Logger {
	if l, ok := ctx.Value(loggerContextKey).(*zap.Logger); ok {
		return l
	}
	return logger()
}

// WithFields returns a new context with the given fields added to the logger.
func WithFields(ctx context.Context, fields ...zap.Field) context.Context {
	if len(fields) == 0 {
		return ctx
	}
	return With(ctx, From(ctx).With(fields...))
}

// Sync flushes any buffered log entries.
func Sync(ctx context.Context) error {
	return From(ctx).Sync()
}

// With returns a new context with the given logger added to the context.
func With(ctx context.Context, l *zap.Logger) context.Context {
	return context.WithValue(ctx, loggerContextKey, l)
}

// Logger returns the global logger.
func Logger() *zap.Logger {
	once.Do(func() {
		if zapLogger = zap.L(); isNopLogger(zapLogger) {
			config = createConfig()
			var err error
			zapLogger, err = config.Build()
			if err != nil {
				fmt.Printf("Logger init failed with error: %s\n", err.Error())
				zapLogger = zap.NewNop()
			}
		}
	})

	return zapLogger
}

func createConfig() *zap.Config {
	development := false
	level := zap.NewAtomicLevelAt(zap.InfoLevel)

	env := os.Getenv("SPEAKEASY_ENVIRONMENT")
	if env == "local" || env == "docker" {
		development = true
		level = zap.NewAtomicLevelAt(zap.DebugLevel)
	}

	return &zap.Config{
		Level:       level,
		Development: development,
		Encoding:    "json",
		EncoderConfig: zapcore.EncoderConfig{
			MessageKey:    "message",
			LevelKey:      "level",
			TimeKey:       "time",
			NameKey:       "name",
			CallerKey:     "caller",
			StacktraceKey: "stacktrace",
			LineEnding:    zapcore.DefaultLineEnding,
			// https://godoc.org/go.uber.org/zap/zapcore#EncoderConfig
			// EncodeName is optional but all others must be set
			EncodeLevel:    zapcore.CapitalLevelEncoder,
			EncodeTime:     zapcore.ISO8601TimeEncoder,
			EncodeDuration: zapcore.StringDurationEncoder,
			EncodeCaller:   zapcore.ShortCallerEncoder,
		},
		OutputPaths:      []string{"stdout"},
		ErrorOutputPaths: []string{"stderr"},
	}
}

func isNopLogger(logger *zap.Logger) bool {
	return reflect.DeepEqual(zap.NewNop(), logger)
}
