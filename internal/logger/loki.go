package logger

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"sync"
	"time"
)

// LokiHandler wraps a slog.Handler and forwards logs to Grafana Loki
type LokiHandler struct {
	inner     slog.Handler
	lokiURL   string
	client    *http.Client
	buffer    []lokiEntry
	mu        sync.Mutex
	ticker    *time.Ticker
	stopChan  chan struct{}
	batchSize int
}

type lokiEntry struct {
	Timestamp string
	Line      string
	Labels    map[string]string
}

type lokiStreams struct {
	Streams []lokiStream `json:"streams"`
}

type lokiStream struct {
	Stream map[string]string `json:"stream"`
	Values [][]string        `json:"values"`
}

// NewLokiHandler creates a new Loki handler
func NewLokiHandler(inner slog.Handler, lokiURL string) *LokiHandler {
	h := &LokiHandler{
		inner:     inner,
		lokiURL:   lokiURL,
		client:    &http.Client{Timeout: 10 * time.Second},
		buffer:    make([]lokiEntry, 0, 100),
		ticker:    time.NewTicker(5 * time.Second),
		stopChan:  make(chan struct{}),
		batchSize: 100,
	}

	// Start background flusher
	go h.flusher()

	return h
}

// Enabled implements slog.Handler
func (h *LokiHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.inner.Enabled(ctx, level)
}

// Handle implements slog.Handler
func (h *LokiHandler) Handle(ctx context.Context, r slog.Record) error {
	// First, let the inner handler process it
	if err := h.inner.Handle(ctx, r); err != nil {
		return err
	}

	// Extract fields for Loki labels
	labels := make(map[string]string)
	labels["level"] = r.Level.String()

	// Build the log line
	var buf bytes.Buffer
	buf.WriteString(r.Message)

	r.Attrs(func(a slog.Attr) bool {
		// Use certain attributes as labels for better filtering in Loki
		switch a.Key {
		case "trace_id", "span_id", "org_id", "project_id", "service":
			labels[a.Key] = a.Value.String()
		default:
			buf.WriteString(fmt.Sprintf(" %s=%v", a.Key, a.Value))
		}
		return true
	})

	// Add to buffer
	h.mu.Lock()
	h.buffer = append(h.buffer, lokiEntry{
		Timestamp: fmt.Sprintf("%d", r.Time.UnixNano()),
		Line:      buf.String(),
		Labels:    labels,
	})
	shouldFlush := len(h.buffer) >= h.batchSize
	h.mu.Unlock()

	// Flush if buffer is full
	if shouldFlush {
		h.flush()
	}

	return nil
}

// WithAttrs implements slog.Handler
func (h *LokiHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &LokiHandler{
		inner:     h.inner.WithAttrs(attrs),
		lokiURL:   h.lokiURL,
		client:    h.client,
		buffer:    h.buffer,
		ticker:    h.ticker,
		stopChan:  h.stopChan,
		batchSize: h.batchSize,
	}
}

// WithGroup implements slog.Handler
func (h *LokiHandler) WithGroup(name string) slog.Handler {
	return &LokiHandler{
		inner:     h.inner.WithGroup(name),
		lokiURL:   h.lokiURL,
		client:    h.client,
		buffer:    h.buffer,
		ticker:    h.ticker,
		stopChan:  h.stopChan,
		batchSize: h.batchSize,
	}
}

// flusher runs in the background and flushes logs periodically
func (h *LokiHandler) flusher() {
	for {
		select {
		case <-h.ticker.C:
			h.flush()
		case <-h.stopChan:
			h.flush() // Final flush
			return
		}
	}
}

// flush sends buffered logs to Loki
func (h *LokiHandler) flush() {
	h.mu.Lock()
	if len(h.buffer) == 0 {
		h.mu.Unlock()
		return
	}
	entries := h.buffer
	h.buffer = make([]lokiEntry, 0, h.batchSize)
	h.mu.Unlock()

	// Group entries by labels
	streams := make(map[string]*lokiStream)
	for _, entry := range entries {
		// Create a key from labels
		labelKey := h.labelsToKey(entry.Labels)

		if stream, exists := streams[labelKey]; exists {
			stream.Values = append(stream.Values, []string{entry.Timestamp, entry.Line})
		} else {
			streams[labelKey] = &lokiStream{
				Stream: entry.Labels,
				Values: [][]string{{entry.Timestamp, entry.Line}},
			}
		}
	}

	// Convert to Loki format
	lokiStreams := lokiStreams{
		Streams: make([]lokiStream, 0, len(streams)),
	}
	for _, stream := range streams {
		lokiStreams.Streams = append(lokiStreams.Streams, *stream)
	}

	// Send to Loki
	if err := h.sendToLoki(lokiStreams); err != nil {
		// Log error but don't fail (best effort)
		fmt.Printf("Failed to send logs to Loki: %v\n", err)
	}
}

// sendToLoki sends logs to Loki push API
func (h *LokiHandler) sendToLoki(streams lokiStreams) error {
	body, err := json.Marshal(streams)
	if err != nil {
		return fmt.Errorf("failed to marshal logs: %w", err)
	}

	req, err := http.NewRequest("POST", h.lokiURL+"/loki/api/v1/push", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := h.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("loki returned error: %d - %s", resp.StatusCode, string(bodyBytes))
	}

	return nil
}

// labelsToKey creates a unique key from labels
func (h *LokiHandler) labelsToKey(labels map[string]string) string {
	var buf bytes.Buffer
	for k, v := range labels {
		buf.WriteString(k)
		buf.WriteString("=")
		buf.WriteString(v)
		buf.WriteString(",")
	}
	return buf.String()
}

// Close stops the flusher
func (h *LokiHandler) Close() {
	close(h.stopChan)
	h.ticker.Stop()
}
