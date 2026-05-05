package els

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"
)

// httpTransport handles HTTP communication with the ELS API.
type httpTransport struct {
	endpoint   string
	apiKey     string
	httpClient *http.Client
	maxRetries int
	baseDelay  time.Duration
}

func newHTTPTransport(config Config) *httpTransport {
	httpClient := config.HTTPClient
	if httpClient == nil {
		httpClient = &http.Client{Timeout: config.Timeout}
	}
	return &httpTransport{
		endpoint:   config.Endpoint,
		apiKey:     config.APIKey,
		httpClient: httpClient,
		maxRetries: config.MaxRetries,
		baseDelay:  config.RetryBaseDelay,
	}
}

// sendBatch sends a batch of entries to the ELS API.
// Returns nil on success, a *SendError on failure.
func (t *httpTransport) sendBatch(ctx context.Context, entries []ErrorEntry) error {
	payload, err := json.Marshal(batchRequest{Errors: entries})
	if err != nil {
		return newPermanentError(0, fmt.Errorf("marshal: %w", err))
	}
	return t.doWithRetry(ctx, "/errors/batch", payload)
}

// sendSingle sends a single entry to the ELS API.
func (t *httpTransport) sendSingle(ctx context.Context, entry ErrorEntry) error {
	payload, err := json.Marshal(entry)
	if err != nil {
		return newPermanentError(0, fmt.Errorf("marshal: %w", err))
	}
	return t.doWithRetry(ctx, "/errors", payload)
}

// doWithRetry performs an HTTP POST with retry logic.
func (t *httpTransport) doWithRetry(ctx context.Context, path string, payload []byte) error {
	url := t.endpoint + path

	var lastErr error
	for attempt := 0; attempt <= t.maxRetries; attempt++ {
		if attempt > 0 {
			delay := t.baseDelay * time.Duration(1<<(attempt-1))
			select {
			case <-ctx.Done():
				return newRetryableError(0, ctx.Err())
			case <-time.After(delay):
			}
		}

		req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(payload))
		if err != nil {
			return newPermanentError(0, fmt.Errorf("create request: %w", err))
		}
		req.Header.Set("Authorization", "Bearer "+t.apiKey)
		req.Header.Set("Content-Type", "application/json")

		resp, err := t.httpClient.Do(req)
		if err != nil {
			lastErr = newRetryableError(0, err)
			continue
		}

		body, _ := io.ReadAll(resp.Body)
		_ = resp.Body.Close()

		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			return nil
		}

		// Rate limited — respect Retry-After header
		if resp.StatusCode == 429 && attempt < t.maxRetries {
			delay := t.baseDelay * time.Duration(1<<attempt)
			if ra := resp.Header.Get("Retry-After"); ra != "" {
				if n, err := strconv.Atoi(ra); err == nil {
					delay = time.Duration(n) * time.Second
				}
			}
			lastErr = newRetryableError(429, fmt.Errorf("%s", string(body)))
			select {
			case <-ctx.Done():
				return newRetryableError(0, ctx.Err())
			case <-time.After(delay):
			}
			continue
		}

		// Server error — retryable
		if resp.StatusCode >= 500 {
			lastErr = newRetryableError(resp.StatusCode, fmt.Errorf("%s", string(body)))
			continue
		}

		// Client error (4xx except 429) — permanent, do not retry
		return newPermanentError(resp.StatusCode, fmt.Errorf("%s", string(body)))
	}

	if lastErr == nil {
		lastErr = newRetryableError(0, fmt.Errorf("request failed after %d retries", t.maxRetries))
	}
	return lastErr
}
