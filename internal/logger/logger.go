package logger

import (
	"context"
	"io"
	"log/slog"
	"os"

	"go.opentelemetry.io/otel/trace"
)

// Logger is a structured logger that integrates with OpenTelemetry traces
type Logger struct {
	*slog.Logger
}

// Config holds logger configuration
type Config struct {
	Level      string // debug, info, warn, error
	Format     string // json, text
	LokiURL    string // Optional Loki endpoint
	Output     io.Writer
	AddSource  bool
}

var defaultLogger *Logger

// init creates a default logger
func init() {
	defaultLogger = NewLogger(Config{
		Level:     "info",
		Format:    "json",
		Output:    os.Stdout,
		AddSource: true,
	})
}

// NewLogger creates a new structured logger
func NewLogger(cfg Config) *Logger {
	var level slog.Level
	switch cfg.Level {
	case "debug":
		level = slog.LevelDebug
	case "info":
		level = slog.LevelInfo
	case "warn":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	default:
		level = slog.LevelInfo
	}

	output := cfg.Output
	if output == nil {
		output = os.Stdout
	}

	opts := &slog.HandlerOptions{
		Level:     level,
		AddSource: cfg.AddSource,
	}

	var handler slog.Handler
	if cfg.Format == "text" {
		handler = slog.NewTextHandler(output, opts)
	} else {
		handler = slog.NewJSONHandler(output, opts)
	}

	// If Loki URL is provided, wrap with Loki handler
	if cfg.LokiURL != "" {
		handler = NewLokiHandler(handler, cfg.LokiURL)
	}

	return &Logger{
		Logger: slog.New(handler),
	}
}

// Default returns the default logger
func Default() *Logger {
	return defaultLogger
}

// SetDefault sets the default logger
func SetDefault(l *Logger) {
	defaultLogger = l
}

// WithContext creates a logger with trace context
func (l *Logger) WithContext(ctx context.Context) *Logger {
	span := trace.SpanFromContext(ctx)
	if !span.IsRecording() {
		return l
	}

	spanCtx := span.SpanContext()
	return &Logger{
		Logger: l.With(
			slog.String("trace_id", spanCtx.TraceID().String()),
			slog.String("span_id", spanCtx.SpanID().String()),
		),
	}
}

// WithFields adds custom fields to the logger
func (l *Logger) WithFields(fields ...slog.Attr) *Logger {
	args := make([]any, len(fields))
	for i, f := range fields {
		args[i] = f
	}
	return &Logger{
		Logger: l.With(args...),
	}
}

// WithOrgProject adds organization and project IDs to the logger
func (l *Logger) WithOrgProject(orgID, projectID string) *Logger {
	return &Logger{
		Logger: l.With(
			slog.String("org_id", orgID),
			slog.String("project_id", projectID),
		),
	}
}

// Convenience methods that use the default logger

// Info logs an info message
func Info(msg string, args ...any) {
	defaultLogger.Info(msg, args...)
}

// Debug logs a debug message
func Debug(msg string, args ...any) {
	defaultLogger.Debug(msg, args...)
}

// Warn logs a warning message
func Warn(msg string, args ...any) {
	defaultLogger.Warn(msg, args...)
}

// Error logs an error message
func Error(msg string, args ...any) {
	defaultLogger.Error(msg, args...)
}

// InfoContext logs an info message with context
func InfoContext(ctx context.Context, msg string, args ...any) {
	defaultLogger.WithContext(ctx).Info(msg, args...)
}

// DebugContext logs a debug message with context
func DebugContext(ctx context.Context, msg string, args ...any) {
	defaultLogger.WithContext(ctx).Debug(msg, args...)
}

// WarnContext logs a warning message with context
func WarnContext(ctx context.Context, msg string, args ...any) {
	defaultLogger.WithContext(ctx).Warn(msg, args...)
}

// ErrorContext logs an error message with context
func ErrorContext(ctx context.Context, msg string, args ...any) {
	defaultLogger.WithContext(ctx).Error(msg, args...)
}
