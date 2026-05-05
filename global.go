package els

import (
	"context"
	"sync"
)

var (
	globalMu     sync.RWMutex
	globalClient *Client
)

// Init initializes the global ELS client. After calling Init, you can use
// package-level functions (CaptureError, CaptureMessage, etc.) without
// passing the client explicitly. This is the recommended approach for
// most applications.
//
//	els.Init(els.Config{
//	    Endpoint: "https://api.example.com/els",
//	    APIKey:   "your-key",
//	    AppSlug:  "my-app",
//	})
//	defer els.Close()
//
//	els.CaptureError(err, els.WithURL("/api"))
func Init(config Config) error {
	c, err := New(config)
	if err != nil {
		return err
	}

	globalMu.Lock()
	defer globalMu.Unlock()

	// Close previous global client if any
	if globalClient != nil {
		globalClient.Close()
	}
	globalClient = c
	return nil
}

// GetClient returns the global client instance, or nil if Init was not called.
func GetClient() *Client {
	globalMu.RLock()
	defer globalMu.RUnlock()
	return globalClient
}

// Close shuts down the global client. Should be called before application exit.
// Equivalent to defer els.Close() after els.Init().
func Close() error {
	globalMu.Lock()
	defer globalMu.Unlock()
	if globalClient == nil {
		return nil
	}
	err := globalClient.Close()
	globalClient = nil
	return err
}

// Package-level capture functions that delegate to the global client.
// These are no-ops if Init was not called.

// CaptureErrorGlobal captures an error using the global client.
// Alias: use the method on *Client when you have a reference.
func CaptureErrorGlobal(err error, opts ...CaptureOption) {
	globalMu.RLock()
	c := globalClient
	globalMu.RUnlock()
	if c != nil {
		c.CaptureError(err, opts...)
	}
}

// CaptureMessageGlobal captures a message using the global client.
func CaptureMessageGlobal(msg string, level string, opts ...CaptureOption) {
	globalMu.RLock()
	c := globalClient
	globalMu.RUnlock()
	if c != nil {
		c.CaptureMessage(msg, level, opts...)
	}
}

// SendSyncGlobal sends an error synchronously using the global client.
func SendSyncGlobal(ctx context.Context, err error, opts ...CaptureOption) error {
	globalMu.RLock()
	c := globalClient
	globalMu.RUnlock()
	if c == nil {
		return nil
	}
	return c.SendSync(ctx, err, opts...)
}

// FlushGlobal flushes the global client's pending entries.
func FlushGlobal() {
	globalMu.RLock()
	c := globalClient
	globalMu.RUnlock()
	if c != nil {
		c.Flush()
	}
}
