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
	return &httpTransport{
		endpoint:   config.Endpoint,
		apiKey:     config.APIKey,
		httpClient: &http.Client{Timeout: config.Timeout},
		maxRetries: config.MaxRetries,
		baseDelay:  config.RetryBaseDelay,
	}
}

// sendBatch sends a batch of entries to the ELS API.
// Returns nil on success, or an error after all retries are exhausted.
func (t *httpTransport) sendBatch(ctx context.Context, entries []ErrorEntry) error {
	payload, err := json.Marshal(batchRequest{Errors: entries})
	if err != nil {
		return fmt.Errorf("els: marshal error: %w", err)
	}

	url := t.endpoint + "/errors/batch"

	var lastErr error
	for attempt := 0; attempt <= t.maxRetries; attempt++ {
		if attempt > 0 {
			delay := t.baseDelay * time.Duration(1<<(attempt-1))
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(delay):
			}
		}

		req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(payload))
		if err != nil {
			return fmt.Errorf("els: create request: %w", err)
		}
		req.Header.Set("Authorization", "Bearer "+t.apiKey)
		req.Header.Set("Content-Type", "application/json")

		resp, err := t.httpClient.Do(req)
		if err != nil {
			lastErr = err
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
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(delay):
			}
			continue
		}

		// Server error — retry
		if resp.StatusCode >= 500 {
			lastErr = fmt.Errorf("els: server error %d: %s", resp.StatusCode, string(body))
			continue
		}

		// Client error (4xx except 429) — do not retry
		return fmt.Errorf("els: client error %d: %s", resp.StatusCode, string(body))
	}

	if lastErr == nil {
		lastErr = fmt.Errorf("els: request failed after %d retries", t.maxRetries)
	}
	return lastErr
}

// sendSingle sends a single entry to the ELS API.
func (t *httpTransport) sendSingle(ctx context.Context, entry ErrorEntry) error {
	payload, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("els: marshal error: %w", err)
	}

	url := t.endpoint + "/errors"

	var lastErr error
	for attempt := 0; attempt <= t.maxRetries; attempt++ {
		if attempt > 0 {
			delay := t.baseDelay * time.Duration(1<<(attempt-1))
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(delay):
			}
		}

		req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(payload))
		if err != nil {
			return fmt.Errorf("els: create request: %w", err)
		}
		req.Header.Set("Authorization", "Bearer "+t.apiKey)
		req.Header.Set("Content-Type", "application/json")

		resp, err := t.httpClient.Do(req)
		if err != nil {
			lastErr = err
			continue
		}

		body, _ := io.ReadAll(resp.Body)
		_ = resp.Body.Close()

		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			return nil
		}

		if resp.StatusCode == 429 && attempt < t.maxRetries {
			delay := t.baseDelay * time.Duration(1<<attempt)
			if ra := resp.Header.Get("Retry-After"); ra != "" {
				if n, err := strconv.Atoi(ra); err == nil {
					delay = time.Duration(n) * time.Second
				}
			}
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(delay):
			}
			continue
		}

		if resp.StatusCode >= 500 {
			lastErr = fmt.Errorf("els: server error %d: %s", resp.StatusCode, string(body))
			continue
		}

		return fmt.Errorf("els: client error %d: %s", resp.StatusCode, string(body))
	}

	if lastErr == nil {
		lastErr = fmt.Errorf("els: request failed after %d retries", t.maxRetries)
	}
	return lastErr
}
