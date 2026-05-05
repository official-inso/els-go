package els

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// worker is the background goroutine that batches entries and sends them.
func (c *Client) worker() {
	defer c.wg.Done()

	// Try to flush any previously buffered entries from disk
	c.flushDiskBuffer()

	batch := make([]ErrorEntry, 0, c.config.BatchSize)
	ticker := time.NewTicker(c.config.BatchInterval)
	defer ticker.Stop()

	for {
		select {
		case entry, ok := <-c.queue:
			if !ok {
				// Channel closed during shutdown — drain remaining
				if len(batch) > 0 {
					c.sendOrBuffer(batch)
				}
				return
			}
			batch = append(batch, *entry)
			if len(batch) >= c.config.BatchSize {
				c.sendOrBuffer(batch)
				batch = make([]ErrorEntry, 0, c.config.BatchSize)
			}

		case <-ticker.C:
			if len(batch) > 0 {
				c.sendOrBuffer(batch)
				batch = make([]ErrorEntry, 0, c.config.BatchSize)
			}

		case <-c.done:
			// Drain queue
			for {
				select {
				case entry, ok := <-c.queue:
					if !ok {
						if len(batch) > 0 {
							c.sendOrBuffer(batch)
						}
						return
					}
					batch = append(batch, *entry)
					if len(batch) >= c.config.BatchSize {
						c.sendOrBuffer(batch)
						batch = make([]ErrorEntry, 0, c.config.BatchSize)
					}
				default:
					if len(batch) > 0 {
						c.sendOrBuffer(batch)
					}
					return
				}
			}
		}
	}
}

// sendOrBuffer attempts to send the batch; on failure, persists to disk.
func (c *Client) sendOrBuffer(batch []ErrorEntry) {
	ctx, cancel := context.WithTimeout(context.Background(), c.config.Timeout*time.Duration(c.config.MaxRetries+1))
	defer cancel()

	err := c.transport.sendBatch(ctx, batch)
	if err == nil {
		return
	}

	// Send failed — write to disk buffer
	if c.config.OnError != nil {
		c.config.OnError(fmt.Errorf("send failed, buffering to disk: %w", err))
	}
	c.writeToDisk(batch)
}

// bufferFilePath returns the path for the disk buffer file.
func (c *Client) bufferFilePath() string {
	dir := c.config.BufferDir
	if dir == "" {
		dir = os.TempDir()
	}
	return filepath.Join(dir, ".els-buffer.jsonl")
}

// writeToDisk appends entries to the disk buffer file in JSONL format.
func (c *Client) writeToDisk(entries []ErrorEntry) {
	path := c.bufferFilePath()

	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		if c.config.OnError != nil {
			c.config.OnError(fmt.Errorf("els: open buffer file: %w", err))
		}
		return
	}
	defer f.Close()

	enc := json.NewEncoder(f)
	for i := range entries {
		if err := enc.Encode(&entries[i]); err != nil {
			if c.config.OnError != nil {
				c.config.OnError(fmt.Errorf("els: write buffer entry: %w", err))
			}
		}
	}
}

// flushDiskBuffer reads and sends entries previously saved to disk.
func (c *Client) flushDiskBuffer() {
	path := c.bufferFilePath()

	f, err := os.Open(path)
	if err != nil {
		return // File doesn't exist or can't be read — nothing to flush
	}

	var entries []ErrorEntry
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024) // 1MB line buffer
	for scanner.Scan() {
		var entry ErrorEntry
		if err := json.Unmarshal(scanner.Bytes(), &entry); err != nil {
			continue // Skip malformed entries
		}
		entries = append(entries, entry)
	}
	f.Close()

	if len(entries) == 0 {
		_ = os.Remove(path)
		return
	}

	// Send in batches
	for i := 0; i < len(entries); i += c.config.BatchSize {
		end := i + c.config.BatchSize
		if end > len(entries) {
			end = len(entries)
		}

		ctx, cancel := context.WithTimeout(context.Background(), c.config.Timeout*time.Duration(c.config.MaxRetries+1))
		err := c.transport.sendBatch(ctx, entries[i:end])
		cancel()

		if err != nil {
			// Still can't send — leave the file intact for next time
			if c.config.OnError != nil {
				c.config.OnError(fmt.Errorf("els: flush disk buffer failed: %w", err))
			}
			return
		}
	}

	// All sent successfully — remove the buffer file
	_ = os.Remove(path)
}
