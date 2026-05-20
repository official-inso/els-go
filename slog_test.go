package els

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func newSlogTestClient(t *testing.T, url string) *Client {
	t.Helper()
	c, err := New(Config{
		endpoint:      url,
		APIKey:        "k",
		AppSlug:       "test-app",
		ServiceName:   "test-svc",
		BatchSize:     1,
		BatchInterval: 20 * time.Millisecond,
		BufferSize:    50,
		MaxRetries:    0,
		Timeout:       2 * time.Second,
		FlushTimeout:  2 * time.Second,
		BufferDir:     t.TempDir(),
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return c
}

func TestLevelFromSlog(t *testing.T) {
	cases := map[slog.Level]Level{
		slog.LevelDebug: LevelDebug,
		slog.LevelInfo:  LevelInfo,
		slog.LevelWarn:  LevelWarning,
		slog.LevelError: LevelError,
	}
	for in, want := range cases {
		if got := LevelFromSlog(in); got != want {
			t.Errorf("LevelFromSlog(%v) = %q, want %q", in, got, want)
		}
	}
}

func TestLevel_Helpers(t *testing.T) {
	if !LevelError.Valid() || LevelError.Priority() != 3 {
		t.Fatalf("LevelError helpers wrong: valid=%v prio=%d", LevelError.Valid(), LevelError.Priority())
	}
	if Level("bogus").Valid() || Level("bogus").Priority() != -1 {
		t.Fatal("unknown level should be invalid with priority -1")
	}
	if LevelDebug.Priority() >= LevelCritical.Priority() {
		t.Fatal("debug must rank below critical")
	}
}

// TestSlogHandler_CapturesErrorWithStack verifies that an slog.Error carrying
// an error-typed attribute is sent with a populated multi-frame Stack and the
// error text in Meta — the developer's main ask.
func TestSlogHandler_CapturesErrorWithStack(t *testing.T) {
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

	c := newSlogTestClient(t, srv.URL)
	logger := slog.New(SlogHandler(c, nil))

	logger.Error("db query failed", "err", errors.New("connection reset"), "table", "users")
	c.Flush()
	c.Close()

	if got.Level != LevelError {
		t.Fatalf("level = %q, want error", got.Level)
	}
	if got.Stack == "" || !strings.Contains(got.Stack, "\n") {
		t.Fatalf("expected multi-frame stack, got %q", got.Stack)
	}
	if got.Meta["error"] != "connection reset" {
		t.Fatalf("expected error text in Meta, got %v", got.Meta["error"])
	}
	if got.ServiceName != "test-svc" {
		t.Fatalf("expected serviceName default applied, got %q", got.ServiceName)
	}
}

// TestSlogHandler_InfoNoStack verifies non-error records don't get a full stack.
func TestSlogHandler_InfoNoStack(t *testing.T) {
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

	c := newSlogTestClient(t, srv.URL)
	logger := slog.New(SlogHandler(c, &SlogHandlerOptions{AddSource: false, CaptureStackOnError: true}))

	logger.Info("cache warm", "keys", 128)
	c.Flush()
	c.Close()

	if got.Level != LevelInfo {
		t.Fatalf("level = %q, want info", got.Level)
	}
	if got.Stack != "" {
		t.Fatalf("info record should have no stack, got %q", got.Stack)
	}
}
