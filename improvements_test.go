package els

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

// TestFlush_WaitsForInFlight verifies that Flush() blocks until batches that
// the sender pool is still delivering have actually been sent (not just until
// the queue is empty).
func TestFlush_WaitsForInFlight(t *testing.T) {
	var received atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(40 * time.Millisecond) // simulate a slow server
		var req batchRequest
		_ = json.NewDecoder(r.Body).Decode(&req)
		received.Add(int32(len(req.Errors)))
		w.WriteHeader(200)
	}))
	defer srv.Close()

	c := newTestClient(t, srv.URL, func(cfg *Config) {
		cfg.BatchSize = 1 // each entry becomes its own batch
	})
	defer c.Close()

	const n = 6
	for i := 0; i < n; i++ {
		c.CaptureError(errors.New("boom"))
	}

	c.Flush()

	if got := received.Load(); got != n {
		t.Fatalf("Flush returned before all in-flight batches were sent: got %d, want %d", got, n)
	}
}

// TestQueueOverflow_NotifiesOnError verifies OnError is invoked when the queue
// overflows (rate-limited), in addition to the Dropped stat.
func TestQueueOverflow_NotifiesOnError(t *testing.T) {
	block := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-block // hold the sender so the queue backs up
		w.WriteHeader(200)
	}))
	defer srv.Close()
	defer close(block)

	var notified atomic.Bool
	c := newTestClient(t, srv.URL, func(cfg *Config) {
		cfg.BatchSize = 1
		cfg.BufferSize = 1
		cfg.SenderConcurrency = 1
		cfg.OnError = func(err error) {
			if err != nil && strings.Contains(err.Error(), "queue full") {
				notified.Store(true)
			}
		}
	})
	defer c.Close()

	// Flood far beyond capacity while the single sender is blocked.
	for i := 0; i < 1000; i++ {
		c.CaptureError(errors.New("flood"))
	}

	if c.GetStats().Dropped == 0 {
		t.Fatal("expected some dropped entries")
	}
	if !notified.Load() {
		t.Fatal("expected OnError to be notified about queue overflow")
	}
}

// TestCaptureCtx_AttachesIDs verifies request/trace IDs from context land in Meta.
func TestCaptureCtx_AttachesIDs(t *testing.T) {
	var got ErrorEntry
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req batchRequest
		_ = json.NewDecoder(r.Body).Decode(&req)
		if len(req.Errors) > 0 {
			got = *req.Errors[0]
		}
		w.WriteHeader(200)
	}))
	defer srv.Close()

	c := newTestClient(t, srv.URL)

	ctx := ContextWithRequestID(context.Background(), "req-123")
	ctx = ContextWithTraceID(ctx, "trace-abc")

	c.CaptureErrorCtx(ctx, errors.New("ctx error"))
	c.Flush()
	c.Close()

	if got.Meta["requestId"] != "req-123" {
		t.Fatalf("requestId = %v, want req-123", got.Meta["requestId"])
	}
	if got.Meta["traceId"] != "trace-abc" {
		t.Fatalf("traceId = %v, want trace-abc", got.Meta["traceId"])
	}
	if got.Stack == "" {
		t.Fatal("CaptureErrorCtx should still capture a stack trace")
	}
}

// TestMaxRetries_NegativeDisables verifies MaxRetries < 0 disables retries.
func TestMaxRetries_NegativeDisables(t *testing.T) {
	var attempts atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts.Add(1)
		w.WriteHeader(500) // always fail (retryable)
	}))
	defer srv.Close()

	c := newTestClient(t, srv.URL, func(cfg *Config) {
		cfg.BatchSize = 1
		cfg.MaxRetries = -1 // disable retries
		cfg.BufferDir = t.TempDir()
	})

	c.CaptureError(errors.New("boom"))
	c.Flush()
	c.Close()

	if got := attempts.Load(); got != 1 {
		t.Fatalf("expected exactly 1 attempt with retries disabled, got %d", got)
	}
}

func BenchmarkCaptureError(b *testing.B) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	}))
	defer srv.Close()

	c, _ := New(Config{
		endpoint:   srv.URL,
		APIKey:     "k",
		AppSlug:    "bench",
		BufferSize: 100000,
		BufferDir:  b.TempDir(),
	})
	defer c.Close()

	err := errors.New("benchmark error")
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		c.CaptureError(err)
	}
}

func BenchmarkCaptureMessage(b *testing.B) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	}))
	defer srv.Close()

	c, _ := New(Config{
		endpoint:   srv.URL,
		APIKey:     "k",
		AppSlug:    "bench",
		BufferSize: 100000,
		BufferDir:  b.TempDir(),
	})
	defer c.Close()

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		c.CaptureMessage("benchmark message", LevelInfo)
	}
}
