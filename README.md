# Go VictoriaLogs Integration

A Go application demonstrating integration with VictoriaLogs for centralized logging, featuring a RESTful API with structured logging and distributed tracing capabilities.

## Overview

This project provides a production-ready logging solution using VictoriaLogs, with:
- **Async batch logging** for high-performance log ingestion
- **Distributed tracing** support with trace_id and user_id
- **Structured logging** with custom fields
- **RESTful API** with user management endpoints
- **Request middleware** for automatic trace injection
- **Graceful shutdown** handling

## Architecture

```
├── cmd/
│   └── main.go                 # Application entry point, HTTP handlers
├── internal/
│   ├── logger/
│   │   ├── interface.go        # Logger interface definitions
│   │   ├── config.go           # Logger configuration
│   │   └── victorialogs.go     # VictoriaLogs implementation
│   └── service/
│       └── user_service.go     # User service with logging
├── config/
│   └── config.go               # Application configuration (placeholder)
├── test/
│   └── api_test.http           # HTTP request tests
├── docker-compose.yaml         # VictoriaLogs setup
└── go.mod
```

## Features

### Logger Features
- **Log Levels**: DEBUG, INFO, WARN, ERROR, FATAL
- **Async Processing**: Buffered channel with batch sending
- **Auto Retry**: Configurable retry mechanism with exponential backoff
- **Context Support**: Automatic trace_id and user_id extraction from context
- **Custom Fields**: Flexible field attachment to log entries
- **Batch Operations**: Efficient batch logging support
- **Graceful Shutdown**: Proper buffer flushing on application shutdown

### API Endpoints
- `GET /health` - Health check endpoint
- `POST /users?username=<name>&email=<email>` - Create user
- `GET /users/{id}` - Get user by ID

### Middleware
- **Trace Middleware**: Automatic trace_id generation and injection
- **Request Logging**: Logs method, path, user agent, and remote IP

## Prerequisites

- Go 1.24.2 or higher
- Docker & Docker Compose (for VictoriaLogs)

## Quick Start

### 1. Start VictoriaLogs

```bash
docker-compose up -d
```

VictoriaLogs will be available at:
- Ingestion endpoint: `http://localhost:9428`
- Query endpoint: `http://localhost:9427`

### 2. Run the application

```bash
go run cmd/main.go
```

The API server starts on `http://localhost:8080`

### 3. Test the API

Using the provided HTTP test file:

```bash
# Health check
curl http://localhost:8080/health

# Create user
curl -X POST "http://localhost:8080/users?username=johndoe&email=john@example.com"

# Get user
curl http://localhost:8080/users/user_1729488000
```

Or use `test/api_test.http` with REST Client extension in VSCode/IntelliJ.

## Configuration

### Logger Configuration

```go
config := &logger.Config{
    VictoriaLogsURL: "http://localhost:9428/insert/jsonline",
    ServiceName:     "demo-api",
    BatchSize:       50,           // Logs per batch
    FlushInterval:   3 * time.Second,  // Flush frequency
    MaxRetries:      3,            // Retry attempts
    Timeout:         5 * time.Second,  // HTTP timeout
    BufferSize:      500,          // Channel buffer size
    Async:           true,         // Enable async mode
}
```

### Environment Variables

- `VICTORIA_LOGS_URL`: VictoriaLogs ingestion endpoint (default: `http://localhost:9428/insert/jsonline`)
- `PORT`: API server port (default: `8080`)

## Usage Examples

### Basic Logging

```go
logger.Info(ctx, "User created successfully", map[string]interface{}{
    "user_id": "user_123",
    "username": "johndoe",
})
```

### Context-aware Logging

```go
ctx := context.WithValue(context.Background(), "trace_id", "trace_abc123")
ctx = context.WithValue(ctx, "user_id", "user_456")

logger.Info(ctx, "Processing request", map[string]interface{}{
    "action": "get_user",
})
```

### Batch Logging

```go
entries := []logger.LogEntry{
    {Level: logger.INFO, Message: "Log 1", ...},
    {Level: logger.WARN, Message: "Log 2", ...},
}
logger.BatchLog(entries)
```

### Custom Logger Instance

```go
userLogger := logger.WithFields(map[string]interface{}{
    "module": "user-service",
}).WithService("user-api")

userLogger.Info(ctx, "User operation", nil)
```

## Log Entry Structure

Logs are sent to VictoriaLogs in JSONL format:

```json
{
  "_msg": "Create new User",
  "_time": "2024-10-21T06:30:00Z",
  "level": "INFO",
  "service": "demo-api",
  "trace_id": "trace_1729488000123456789",
  "user_id": "user_123",
  "fields": {
    "username": "johndoe",
    "email": "john@example.com",
    "action": "create_user_success",
    "duration": 105
  }
}
```

## Querying Logs

Query logs via VictoriaLogs UI or API:

```bash
# Query by service
curl "http://localhost:9427/select/logsql/query" -d 'query=_stream:{service="demo-api"}'

# Query by trace_id
curl "http://localhost:9427/select/logsql/query" -d 'query=trace_id:trace_*'

# Query by log level
curl "http://localhost:9427/select/logsql/query" -d 'query=level:ERROR'
```

## Performance Considerations

### Async Mode Benefits
- Non-blocking log operations
- Automatic batching reduces HTTP requests
- Configurable buffer prevents memory overflow

### Retry Logic
- Exponential backoff: 1s, 2s, 3s delays
- Prevents log loss during temporary network issues
- Max 3 retries by default

### Resource Management
- Buffer size: 500 entries (configurable)
- Batch size: 50 entries (configurable)
- Flush interval: 3 seconds (configurable)

## Graceful Shutdown

The application handles shutdown gracefully:

1. Catches SIGTERM/SIGINT signals
2. Stops accepting new requests
3. Flushes remaining logs
4. Closes logger and HTTP server
5. Waits for in-flight requests (2s timeout)

```go
defer cleanup()  // Ensures logger.Close() is called
```

## Error Handling

### Logger Errors
- Buffer full → Logs dropped (default behavior)
- Network errors → Retry with exponential backoff
- Serialization errors → Log skipped, continue processing

### API Errors
- Invalid username → Returns 500 Internal Server Error
- Missing parameters → Handled by handler logic

## Development

### Project Structure

- **cmd/main.go**: HTTP server setup, handlers, middleware
- **internal/logger**: Logger implementation and interfaces
- **internal/service**: Business logic with logging integration
- **config**: Configuration management (currently placeholder)
- **test**: API testing files

### Adding New Endpoints

1. Create handler function with logger parameter
2. Register route with mux router
3. Use middleware for automatic tracing
4. Add structured logging with appropriate fields

Example:

```go
func deleteUserHandler(userService *service.UserService, logger *logger.VictoriaLogsLogger) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        logger.Info(r.Context(), "Delete user request", map[string]interface{}{
            "user_id": mux.Vars(r)["id"],
        })
        // Implementation
    }
}
```

## Troubleshooting

### Logs not appearing in VictoriaLogs

1. Check VictoriaLogs is running: `docker-compose ps`
2. Verify connection: `curl http://localhost:9428`
3. Check application logs for send errors
4. Ensure `Async: true` with sufficient buffer size

### High memory usage

- Reduce `BufferSize` in configuration
- Decrease `FlushInterval` for more frequent flushes
- Lower `BatchSize` for smaller batches

### Missing trace_id in logs

- Ensure `traceMiddleware` is registered
- Verify context propagation in handler chain
- Check context value key matches: `"trace_id"`

## License

This is a demonstration project for educational purposes.

## References

- [VictoriaLogs Documentation](https://docs.victoriametrics.com/victorialogs/)
- [VictoriaLogs Query Language](https://docs.victoriametrics.com/victorialogs/logsql/)
- [Gorilla Mux](https://github.com/gorilla/mux)
