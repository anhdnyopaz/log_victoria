package logger

import "context"

type LogLevel int

const (
	DEBUG LogLevel = iota
	INFO
	WARN
	ERROR
	FATAL
)

type LogEntry struct {
	Level     LogLevel               `json:"level"`
	Message   string                 `json:"message"`
	Timestamp int64                  `json:"timestamp"`
	Service   string                 `json:"service"`
	TraceID   string                 `json:"trace_id,omitempty"`
	UserID    string                 `json:"user_id,omitempty"`
	Fields    map[string]interface{} `json:"fields,omitempty"`
}

type Logger interface {
	Debug(ctx context.Context, msg string, fields map[string]interface{})
	Info(ctx context.Context, msg string, fields map[string]interface{})
	Warn(ctx context.Context, msg string, fields map[string]interface{})
	Error(ctx context.Context, msg string, fields map[string]interface{})
	Fatal(ctx context.Context, msg string, fields map[string]interface{})

	// BatchLog Batch operations
	BatchLog(entries []LogEntry) error
	Flush() error
	Close() error
}

type ContextLogger interface {
	WithContext(ctx context.Context) Logger
	WithFields(fields map[string]interface{}) Logger
	WithService(service string) Logger
}
