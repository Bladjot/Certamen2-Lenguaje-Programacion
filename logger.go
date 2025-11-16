package main

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"
)

// Logger maneja la escritura concurrente de entradas JSON line-by-line.
type Logger struct {
	mu   sync.Mutex
	enc  *json.Encoder
	file *os.File
}

func NewLogger(path string) (*Logger, error) {
	file, err := os.Create(path)
	if err != nil {
		return nil, err
	}
	enc := json.NewEncoder(file)
	return &Logger{enc: enc, file: file}, nil
}

func (l *Logger) Close() error {
	if l == nil || l.file == nil {
		return nil
	}
	return l.file.Close()
}

func (l *Logger) Log(entry LogEntry) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if entry.WallTime.IsZero() {
		entry.WallTime = time.Now()
	}
	if err := l.enc.Encode(entry); err != nil {
		fmt.Fprintf(os.Stderr, "logger error: %v\n", err)
	}
}

// LogEntry describe los eventos significativos a registrar.
type LogEntry struct {
	WallTime     time.Time      `json:"wall_time"`
	Entity       string         `json:"entity"`
	Event        string         `json:"event"`
	SimTime      int            `json:"sim_time"`
	WorkerID     int            `json:"worker_id,omitempty"`
	EventID      int            `json:"event_id,omitempty"`
	Target       int            `json:"target_worker,omitempty"`
	RollbackFrom int            `json:"rollback_from,omitempty"`
	RollbackTo   int            `json:"rollback_to,omitempty"`
	Details      map[string]any `json:"details,omitempty"`
}
