package log

import (
	"context"
	"log"
	"reflect"
	"sync"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var (
	once   sync.Once
	logger *zap.Logger
	config *zap.Config
)

type contextKey string

const loggerKey contextKey = "logger"

// Logger returns the global logger.
func Logger() *zap.Logger {
	once.Do(func() {
		if logger = zap.L(); isNopLogger(logger) {
			config = createConfig()
			var err error
			logger, err = config.Build()
			if err != nil {
				log.Printf("Logger init failed with error: %s\n", err.Error())
				logger = zap.NewNop()
			}
		}
	})

	return logger
}

// From returns the logger associated with the given context.
func From(ctx context.Context) *zap.Logger {
	if l, ok := ctx.Value(loggerKey).(*zap.Logger); ok {
		return l
	}
	return Logger()
}

// With returns a new context with the provided logger.
func With(ctx context.Context, l *zap.Logger) context.Context {
	return context.WithValue(ctx, loggerKey, l)
}

// WithFields returns a new context with the provided fields attached to the logging context.
func WithFields(ctx context.Context, fields ...zap.Field) context.Context {
	if len(fields) == 0 {
		return ctx
	}
	return With(ctx, From(ctx).With(fields...))
}

func createConfig() *zap.Config {
	return &zap.Config{
		Level:       zap.NewAtomicLevelAt(zap.DebugLevel),
		Development: true,
		Encoding:    "console",
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
