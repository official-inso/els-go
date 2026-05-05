package els

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func newTestServer(handler http.HandlerFunc) *httptest.Server {
	return httptest.NewServer(handler)
}

func newTestClient(t *testing.T, endpoint string, opts ...func(*Config)) *Client {
	t.Helper()
	cfg := Config{
		Endpoint:      endpoint,
		APIKey:        "test-key",
		AppSlug:       "test-app",
		DeploymentEnv: "DEV",
		ServiceName:   "test-service",
		BatchSize:     5,
		BatchInterval: 100 * time.Millisecond,
		BufferSize:    100,
		MaxRetries:    1,
		RetryBaseDelay: 10 * time.Millisecond,
		Timeout:       2 * time.Second,
		FlushTimeout:  2 * time.Second,
		BufferDir:     t.TempDir(),
	}
	for _, opt := range opts {
		opt(&cfg)
	}
	c, err := New(cfg)
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}
	return c
}

func TestNew_RequiresEndpoint(t *testing.T) {
	_, err := New(Config{APIKey: "key"})
	if err == nil {
		t.Fatal("expected error for missing Endpoint")
	}
}

func TestNew_RequiresAPIKey(t *testing.T) {
	_, err := New(Config{Endpoint: "http://localhost"})
	if err == nil {
		t.Fatal("expected error for missing APIKey")
	}
}

func TestCaptureError_NilIsNoop(t *testing.T) {
	srv := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("should not be called for nil error")
	})
	defer srv.Close()

	c := newTestClient(t, srv.URL)
	defer c.Close()

	c.CaptureError(nil)
	time.Sleep(200 * time.Millisecond)
}

func TestCaptureError_SendsBatch(t *testing.T) {
	var received atomic.Int32
	srv := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		var req batchRequest
		json.NewDecoder(r.Body).Decode(&req)
		received.Add(int32(len(req.Errors)))
		w.WriteHeader(200)
		json.NewEncoder(w).Encode(BatchResult{Created: len(req.Errors)})
	})
	defer srv.Close()

	c := newTestClient(t, srv.URL)

	for i := 0; i < 10; i++ {
		c.CaptureError(errors.New("test error"))
	}

	c.Flush()
	c.Close()

	if got := received.Load(); got != 10 {
		t.Fatalf("expected 10 entries, got %d", got)
	}
}

func TestCaptureMessage_SetsLevel(t *testing.T) {
	var receivedLevel string
	srv := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		var req batchRequest
		json.NewDecoder(r.Body).Decode(&req)
		if len(req.Errors) > 0 {
			receivedLevel = req.Errors[0].Level
		}
		w.WriteHeader(200)
	})
	defer srv.Close()

	c := newTestClient(t, srv.URL)
	c.CaptureMessage("hello", LevelInfo)
	c.Flush()
	c.Close()

	if receivedLevel != LevelInfo {
		t.Fatalf("expected level %q, got %q", LevelInfo, receivedLevel)
	}
}

func TestMinLevel_FiltersDebug(t *testing.T) {
	var received atomic.Int32
	srv := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		var req batchRequest
		json.NewDecoder(r.Body).Decode(&req)
		received.Add(int32(len(req.Errors)))
		w.WriteHeader(200)
	})
	defer srv.Close()

	c := newTestClient(t, srv.URL, func(cfg *Config) {
		cfg.MinLevel = LevelWarning
	})

	c.CaptureMessage("debug msg", LevelDebug)
	c.CaptureMessage("info msg", LevelInfo)
	c.CaptureMessage("warning msg", LevelWarning)
	c.CaptureError(errors.New("error"))

	c.Flush()
	c.Close()

	if got := received.Load(); got != 2 {
		t.Fatalf("expected 2 entries (warning+error), got %d", got)
	}
}

func TestBeforeSend_CanDropEntry(t *testing.T) {
	var received atomic.Int32
	srv := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		var req batchRequest
		json.NewDecoder(r.Body).Decode(&req)
		received.Add(int32(len(req.Errors)))
		w.WriteHeader(200)
	})
	defer srv.Close()

	c := newTestClient(t, srv.URL, func(cfg *Config) {
		cfg.BeforeSend = func(e *ErrorEntry) *ErrorEntry {
			if e.Message == "drop me" {
				return nil
			}
			return e
		}
	})

	c.CaptureError(errors.New("drop me"))
	c.CaptureError(errors.New("keep me"))

	c.Flush()
	c.Close()

	if got := received.Load(); got != 1 {
		t.Fatalf("expected 1 entry, got %d", got)
	}
}

func TestDiskBuffer_PersistsOnFailure(t *testing.T) {
	tmpDir := t.TempDir()
	failCount := 0

	srv := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		failCount++
		w.WriteHeader(500)
		w.Write([]byte("server error"))
	})
	defer srv.Close()

	c := newTestClient(t, srv.URL, func(cfg *Config) {
		cfg.BufferDir = tmpDir
		cfg.MaxRetries = 0 // No retries — fail immediately
	})

	c.CaptureError(errors.New("will buffer"))
	c.Flush()
	c.Close()

	// Check that buffer file was created
	bufPath := filepath.Join(tmpDir, ".els-buffer.jsonl")
	data, err := os.ReadFile(bufPath)
	if err != nil {
		t.Fatalf("buffer file not created: %v", err)
	}
	if len(data) == 0 {
		t.Fatal("buffer file is empty")
	}

	var entry ErrorEntry
	if err := json.Unmarshal(data[:len(data)-1], &entry); err != nil {
		t.Fatalf("invalid buffer entry: %v", err)
	}
	if entry.Message != "will buffer" {
		t.Fatalf("unexpected message: %s", entry.Message)
	}
}

func TestDiskBuffer_FlushOnRestart(t *testing.T) {
	tmpDir := t.TempDir()
	bufPath := filepath.Join(tmpDir, ".els-buffer.jsonl")

	// Pre-seed buffer file
	entry := ErrorEntry{
		Message:   "buffered entry",
		URL:       "/test",
		Timestamp: time.Now().UTC().Format(time.RFC3339Nano),
		Level:     LevelError,
		Source:    SourceServer,
	}
	data, _ := json.Marshal(entry)
	os.WriteFile(bufPath, append(data, '\n'), 0644)

	var received atomic.Int32
	srv := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		var req batchRequest
		json.NewDecoder(r.Body).Decode(&req)
		received.Add(int32(len(req.Errors)))
		w.WriteHeader(200)
	})
	defer srv.Close()

	// New client should flush disk buffer on startup
	c := newTestClient(t, srv.URL, func(cfg *Config) {
		cfg.BufferDir = tmpDir
	})
	time.Sleep(300 * time.Millisecond)
	c.Close()

	if got := received.Load(); got != 1 {
		t.Fatalf("expected 1 buffered entry flushed, got %d", got)
	}

	// Buffer file should be removed after successful flush
	if _, err := os.Stat(bufPath); !os.IsNotExist(err) {
		t.Fatal("buffer file should be removed after flush")
	}
}

func TestClose_Concurrent(t *testing.T) {
	srv := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	})
	defer srv.Close()

	c := newTestClient(t, srv.URL)

	var wg sync.WaitGroup
	// Concurrent captures and close — should not panic
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			c.CaptureError(errors.New("concurrent"))
		}()
	}

	wg.Wait()
	c.Close()
	// Double close should be safe
	c.Close()
}

func TestSendSync_Success(t *testing.T) {
	srv := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(201)
	})
	defer srv.Close()

	c := newTestClient(t, srv.URL)
	defer c.Close()

	err := c.SendSync(context.Background(), errors.New("sync error"), WithURL("/test"))
	if err != nil {
		t.Fatalf("SendSync failed: %v", err)
	}
}

func TestSendSync_ReturnsError(t *testing.T) {
	srv := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(400)
		w.Write([]byte("bad request"))
	})
	defer srv.Close()

	c := newTestClient(t, srv.URL, func(cfg *Config) {
		cfg.MaxRetries = 0
	})
	defer c.Close()

	err := c.SendSync(context.Background(), errors.New("sync error"), WithURL("/test"))
	if err == nil {
		t.Fatal("expected error from SendSync")
	}

	var se *SendError
	if !As(err, &se) {
		t.Fatalf("expected *SendError, got %T", err)
	}
	if se.IsRetryable {
		t.Fatal("400 should not be retryable")
	}
}

func TestHealth_Success(t *testing.T) {
	srv := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/health" {
			w.WriteHeader(200)
			return
		}
		w.WriteHeader(404)
	})
	defer srv.Close()

	c := newTestClient(t, srv.URL)
	defer c.Close()

	err := c.Health(context.Background())
	if err != nil {
		t.Fatalf("Health check failed: %v", err)
	}
}

func TestHealth_Failure(t *testing.T) {
	srv := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(503)
	})
	defer srv.Close()

	c := newTestClient(t, srv.URL)
	defer c.Close()

	err := c.Health(context.Background())
	if err == nil {
		t.Fatal("expected health check to fail")
	}
}

func TestSessionID_AutoGenerated(t *testing.T) {
	srv := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	})
	defer srv.Close()

	c := newTestClient(t, srv.URL)
	defer c.Close()

	sid := c.SessionID()
	if sid == "" {
		t.Fatal("session ID should be auto-generated")
	}
	if len(sid) < 10 {
		t.Fatalf("session ID too short: %s", sid)
	}
}

func TestSessionID_Override(t *testing.T) {
	srv := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	})
	defer srv.Close()

	c := newTestClient(t, srv.URL)
	defer c.Close()

	c.SetSessionID("custom-session")
	if got := c.SessionID(); got != "custom-session" {
		t.Fatalf("expected 'custom-session', got %q", got)
	}
}

func TestEnrichDefaults(t *testing.T) {
	var receivedEntry ErrorEntry
	srv := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		var req batchRequest
		json.NewDecoder(r.Body).Decode(&req)
		if len(req.Errors) > 0 {
			receivedEntry = req.Errors[0]
		}
		w.WriteHeader(200)
	})
	defer srv.Close()

	c := newTestClient(t, srv.URL)
	c.CaptureError(errors.New("test"), WithURL("/api"))
	c.Flush()
	c.Close()

	if receivedEntry.AppSlug != "test-app" {
		t.Fatalf("expected appSlug 'test-app', got %q", receivedEntry.AppSlug)
	}
	if receivedEntry.DeploymentEnv != "DEV" {
		t.Fatalf("expected deploymentEnv 'DEV', got %q", receivedEntry.DeploymentEnv)
	}
	if receivedEntry.ServiceName != "test-service" {
		t.Fatalf("expected serviceName 'test-service', got %q", receivedEntry.ServiceName)
	}
}

func TestOptions_Applied(t *testing.T) {
	var receivedEntry ErrorEntry
	srv := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		var req batchRequest
		json.NewDecoder(r.Body).Decode(&req)
		if len(req.Errors) > 0 {
			receivedEntry = req.Errors[0]
		}
		w.WriteHeader(200)
	})
	defer srv.Close()

	c := newTestClient(t, srv.URL)
	c.CaptureError(errors.New("opts test"),
		WithURL("/test"),
		WithLevel(LevelCritical),
		WithSource(SourceClient),
		WithUserAgent("TestBot/1.0"),
		WithLanguage("en-US"),
		WithReferrer("http://example.com"),
		WithMeta(map[string]any{"key": "value"}),
	)
	c.Flush()
	c.Close()

	if receivedEntry.URL != "/test" {
		t.Fatalf("URL: expected '/test', got %q", receivedEntry.URL)
	}
	if receivedEntry.Level != LevelCritical {
		t.Fatalf("Level: expected 'critical', got %q", receivedEntry.Level)
	}
	if receivedEntry.Source != SourceClient {
		t.Fatalf("Source: expected 'client', got %q", receivedEntry.Source)
	}
	if receivedEntry.UserAgent != "TestBot/1.0" {
		t.Fatalf("UserAgent: expected 'TestBot/1.0', got %q", receivedEntry.UserAgent)
	}
}

func TestMaxBufferFileSize_Enforced(t *testing.T) {
	tmpDir := t.TempDir()
	bufPath := filepath.Join(tmpDir, ".els-buffer.jsonl")

	// Create a file that's already at the limit
	bigData := make([]byte, 1024)
	os.WriteFile(bufPath, bigData, 0644)

	srv := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(500) // Always fail
	})
	defer srv.Close()

	var errorCalled atomic.Bool
	c := newTestClient(t, srv.URL, func(cfg *Config) {
		cfg.BufferDir = tmpDir
		cfg.MaxBufferFileSize = 1024 // 1KB limit
		cfg.MaxRetries = 0
		cfg.OnError = func(err error) {
			errorCalled.Store(true)
		}
	})

	c.CaptureError(errors.New("should be dropped"))
	c.Flush()
	c.Close()

	if !errorCalled.Load() {
		t.Fatal("OnError should have been called for buffer full")
	}

	// File should not have grown
	info, _ := os.Stat(bufPath)
	if info.Size() > 1024 {
		t.Fatalf("buffer file grew beyond limit: %d bytes", info.Size())
	}
}
