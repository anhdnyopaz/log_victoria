package logger

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"
)

type VictoriaLogsLogger struct {
	config    *Config
	client    *http.Client
	buffer    chan LogEntry
	batchChan chan []LogEntry
	wg        sync.WaitGroup
	ctx       context.Context
	cancel    context.CancelFunc

	//Context Fields
	contextFields map[string]interface{}
	serviceName   string
	mu            sync.RWMutex //Need to know RWMutex
}

type VictoriaLogsEntry struct {
	Msg    string    `json:"_msg"`
	Time   time.Time `json:"_time"`
	Stream string    `json:"_stream,omitempty"`
	// Custom fields
	Level   string `json:"level,omitempty"`
	Service string `json:"service,omitempty"`
	TraceId string `json:"trace_id,omitempty"`
	UserId  string `json:"user_id,omitempty"`
	// AdditionalFields
	Fields map[string]interface{} `json:"fields,omitempty"`
}

func (v *VictoriaLogsLogger) WithContext(ctx context.Context) Logger {
	newLogger := &VictoriaLogsLogger{
		config:        v.config,
		client:        v.client,
		buffer:        v.buffer,
		batchChan:     v.batchChan,
		ctx:           ctx,
		cancel:        v.cancel,
		contextFields: make(map[string]interface{}),
		serviceName:   v.serviceName,
	}
	v.mu.RLock()
	for k, v := range v.contextFields {
		newLogger.contextFields[k] = v
	}
	v.mu.RUnlock()

	return newLogger
}

func (v *VictoriaLogsLogger) WithFields(fields map[string]interface{}) Logger {
	newLogger := &VictoriaLogsLogger{
		config:        v.config,
		client:        v.client,
		buffer:        v.buffer,
		batchChan:     v.batchChan,
		ctx:           v.ctx,
		cancel:        v.cancel,
		contextFields: make(map[string]interface{}),
		serviceName:   v.serviceName,
	}
	v.mu.RLock()
	for k, v := range v.contextFields {
		newLogger.contextFields[k] = v
	}
	v.mu.RUnlock()

	for k, v := range fields {
		newLogger.contextFields[k] = v
	}
	return newLogger
}

func (v *VictoriaLogsLogger) WithService(service string) Logger {
	newLogger := &VictoriaLogsLogger{
		config:        v.config,
		client:        v.client,
		buffer:        v.buffer,
		batchChan:     v.batchChan,
		ctx:           v.ctx,
		cancel:        v.cancel,
		contextFields: make(map[string]interface{}),
		serviceName:   service,
	}
	v.mu.RLock()
	for k, v := range v.contextFields {
		newLogger.contextFields[k] = v
	}
	v.mu.RUnlock()

	return newLogger
}

func (v *VictoriaLogsLogger) Debug(ctx context.Context, msg string, fields map[string]interface{}) {
	v.log(ctx, DEBUG, msg, fields)
}

func (v *VictoriaLogsLogger) Info(ctx context.Context, msg string, fields map[string]interface{}) {
	v.log(ctx, INFO, msg, fields)
}

func (v *VictoriaLogsLogger) Warn(ctx context.Context, msg string, fields map[string]interface{}) {
	v.log(ctx, WARN, msg, fields)
}

func (v *VictoriaLogsLogger) Error(ctx context.Context, msg string, fields map[string]interface{}) {
	v.log(ctx, ERROR, msg, fields)
}

func (v *VictoriaLogsLogger) Fatal(ctx context.Context, msg string, fields map[string]interface{}) {
	v.log(ctx, FATAL, msg, fields)
}

func (v *VictoriaLogsLogger) BatchLog(entries []LogEntry) error {
	if v.config.Async {
		for _, entry := range entries {
			select {
			case v.buffer <- entry:
			default:
				return fmt.Errorf("buffer full")
			}
		}
		return nil
	}
	v.sendBatch(entries)
	return nil
}

// Flush Đảm bảo tất cả các logs được gửi
func (v *VictoriaLogsLogger) Flush() error {
	if !v.config.Async {
		return nil
	}

	//Đợi buffer rỗng
	for len(v.buffer) > 0 {
		time.Sleep(100 * time.Millisecond)
	}
	return nil
}

func (v *VictoriaLogsLogger) Close() error {
	v.cancel()
	v.wg.Wait()
	close(v.buffer)
	close(v.batchChan)
	return nil
}

func (v *VictoriaLogsLogger) startAsyncProcessing() {
	v.wg.Add(1)
	go func() {
		defer v.wg.Done()
		ticker := time.NewTicker(v.config.FlushInterval)
		defer ticker.Stop()

		batch := v.NewLoggerEntryBatch()

		for {
			select {
			case entry := <-v.buffer:
				batch = append(batch, entry)
				v.sendBatch(batch)
				batch = v.NewLoggerEntryBatch()
			case <-ticker.C:
				if len(batch) > 0 {
					v.sendBatch(batch)
				}
				batch = v.NewLoggerEntryBatch()
			case <-v.ctx.Done():
				if len(batch) > 0 {
					v.sendBatch(batch)
				}
				return
			}
		}
	}()
}

func (v *VictoriaLogsLogger) NewLoggerEntryBatch() []LogEntry {
	return make([]LogEntry, 0, v.config.BatchSize)
}

func (v *VictoriaLogsLogger) sendBatch(batch []LogEntry) {
	fmt.Printf("Send batch %v\n", batch)
	if len(batch) == 0 {
		return
	}

	//Convert to JSONL format
	var buff bytes.Buffer
	for _, entry := range batch {
		vlEntry := VictoriaLogsEntry{
			Msg:     entry.Message,
			Time:    time.Unix(0, entry.Timestamp).UTC(),
			Level:   entry.Level.String(),
			Service: entry.Service,
			TraceId: entry.TraceID,
			UserId:  entry.UserID,
			Fields:  entry.Fields,
		}

		data, err := json.Marshal(vlEntry)
		if err != nil {
			fmt.Println(err.Error())
		}
		fmt.Printf("Send log data: %v\n", entry)
		if err != nil {
			continue
		}
		buff.Write(data)
		buff.WriteByte('\n')
	}

	//Retry logic
	for i := 0; i < v.config.MaxRetries; i++ {
		if err := v.sendToVictoriaLogs(buff.Bytes()); err == nil {
			return
		} else {
			fmt.Println(err)
		}
		time.Sleep(time.Duration(i+1) * time.Second)
	}
}

func (v *VictoriaLogsLogger) sendToVictoriaLogs(data []byte) error {
	req, err := http.NewRequestWithContext(
		v.ctx,
		"POST",
		v.config.VictoriaLogsURL,
		bytes.NewReader(data),
	)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-ndjson")
	resp, err := v.client.Do(req)
	if err != nil {
		return err
	}

	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			return
		}
	}(resp.Body)

	if resp.StatusCode >= 400 {
		return fmt.Errorf("VictoriaLogs returned status code %d", resp.StatusCode)
	}

	return nil
}

func (v *VictoriaLogsLogger) log(ctx context.Context, info LogLevel, msg string, fields map[string]interface{}) {
	entry := v.createLogEntry(info, msg, fields)

	if traceID := ctx.Value("trace_id"); traceID != nil {
		if tid, ok := traceID.(string); ok {
			entry.TraceID = tid
		}
	}

	if userId := ctx.Value("user_id"); userId != nil {
		if uid, ok := userId.(string); ok {
			entry.UserID = uid
		}
	}

	if v.config.Async {
		select {
		case v.buffer <- entry:
		default:
		}
	} else {
		v.sendBatch([]LogEntry{entry})
	}

}

func (v *VictoriaLogsLogger) createLogEntry(level LogLevel, msg string, fields map[string]interface{}) LogEntry {
	v.mu.RLock()
	defer v.mu.RUnlock()

	entry := LogEntry{
		Level:     level,
		Message:   msg,
		Timestamp: time.Now().UnixNano(),
		Service:   v.serviceName,
		Fields:    fields,
	}
	for k, v := range v.contextFields {
		entry.Fields[k] = v
	}
	return entry
}

func NewVictoriaLogsLogger(config *Config) (*VictoriaLogsLogger, error) {
	if config == nil {
		config = DefaultConfig()
	}

	ctx, cancel := context.WithCancel(context.Background())

	logger := &VictoriaLogsLogger{
		config: config,
		client: &http.Client{
			Timeout: config.Timeout,
		},
		buffer:        make(chan LogEntry, config.BufferSize),
		batchChan:     make(chan []LogEntry, config.BufferSize),
		ctx:           ctx,
		cancel:        cancel,
		contextFields: make(map[string]interface{}),
		serviceName:   config.ServiceName,
	}

	if config.Async {
		logger.startAsyncProcessing()
	}
	return logger, nil
}
