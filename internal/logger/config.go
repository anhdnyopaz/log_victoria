package logger

import "time"

type Config struct {
	VictoriaLogsURL string        `yaml:"victoria_logs_url"`
	ServiceName     string        `yaml:"service_name"`
	BatchSize       int           `yaml:"batch_size"`
	FlushInterval   time.Duration `yaml:"flush_interval"`
	MaxRetries      int           `yaml:"max_retries"`
	Timeout         time.Duration `yaml:"timeout"`
	BufferSize      int           `yaml:"buffer_size"`
	Async           bool          `yaml:"async"`
}

func DefaultConfig() *Config {
	return &Config{
		VictoriaLogsURL: "http://localhost:9428/insert/jsonline",
		ServiceName:     "default-service",
		BatchSize:       100,
		FlushInterval:   5 * time.Second,
		MaxRetries:      3,
		Timeout:         30 * time.Second,
		BufferSize:      1000,
		Async:           true,
	}
}
