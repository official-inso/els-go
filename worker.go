package els

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync/atomic"
	"time"
)

// worker is the background goroutine that batches entries and hands each
// completed batch to the sender pool. It never performs network I/O itself, so
// a slow or unreachable server can't block ingestion of new entries.
func (c *Client) worker() {
	defer c.wg.Done()

	// Try to flush any previously buffered entries from disk.
	c.flushDiskBuffer()

	batch := c.getBatch()
	ticker := time.NewTicker(c.config.BatchInterval)
	defer ticker.Stop()

	for {
		select {
		case entry := <-c.queue:
			if entry == nil {
				continue
			}
			batch = append(batch, entry)
			if len(batch) >= c.config.BatchSize {
				c.dispatch(batch)
				batch = c.getBatch()
			}

		case <-ticker.C:
			if len(batch) > 0 {
				c.dispatch(batch)
				batch = c.getBatch()
			}

		case <-c.flushReq:
			// Flush() asked us to hand off the partial batch immediately.
			if len(batch) > 0 {
				c.dispatch(batch)
				batch = c.getBatch()
			}

		case <-c.done:
			c.drainQueue(&batch)
			if len(batch) > 0 {
				c.dispatch(batch)
			}
			close(c.sendCh) // senders finish remaining batches, then exit
			return
		}
	}
}

// drainQueue reads all remaining entries from the queue into the batch,
// dispatching whenever the batch fills up.
func (c *Client) drainQueue(batch *[]*ErrorEntry) {
	for {
		select {
		case entry := <-c.queue:
			if entry == nil {
				continue
			}
			*batch = append(*batch, entry)
			if len(*batch) >= c.config.BatchSize {
				c.dispatch(*batch)
				*batch = c.getBatch()
			}
		default:
			return
		}
	}
}

// dispatch hands a batch to the sender pool and tracks it as in-flight.
func (c *Client) dispatch(batch []*ErrorEntry) {
	atomic.AddInt64(&c.inFlight, 1)
	c.sendCh <- batch
}

// sender drains the send channel. Multiple senders run concurrently so several
// batches can be in flight at once without blocking the worker.
func (c *Client) sender() {
	defer c.senderWg.Done()
	for batch := range c.sendCh {
		c.sendOrBuffer(batch)
		atomic.AddInt64(&c.inFlight, -1)
		c.putBatch(batch)
	}
}

// getBatch / putBatch reuse batch backing arrays via a pool to avoid an
// allocation per batch.
func (c *Client) getBatch() []*ErrorEntry {
	return c.batchPool.Get().([]*ErrorEntry)[:0]
}

func (c *Client) putBatch(b []*ErrorEntry) {
	if cap(b) == 0 {
		return
	}
	c.batchPool.Put(b[:0])
}

// sendOrBuffer attempts to send the batch; on failure, persists to disk.
func (c *Client) sendOrBuffer(batch []*ErrorEntry) {
	ctx, cancel := context.WithTimeout(context.Background(), c.config.Timeout*time.Duration(c.config.MaxRetries+1))
	defer cancel()

	if err := c.transport.sendBatch(ctx, batch); err == nil {
		atomic.AddInt64(&c.stats.Sent, int64(len(batch)))
		return
	} else {
		// Send failed — write to disk buffer.
		atomic.AddInt64(&c.stats.Failed, int64(len(batch)))
		if c.config.OnError != nil {
			c.config.OnError(fmt.Errorf("send failed, buffering to disk: %w", err))
		}
		c.writeToDisk(batch)
	}
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
// Respects MaxBufferFileSize. Guarded by diskMu — multiple senders may call it.
func (c *Client) writeToDisk(entries []*ErrorEntry) {
	c.diskMu.Lock()
	defer c.diskMu.Unlock()

	path := c.bufferFilePath()

	// Check file size limit before writing.
	if info, err := os.Stat(path); err == nil {
		if info.Size() >= c.config.MaxBufferFileSize {
			if c.config.OnError != nil {
				c.config.OnError(fmt.Errorf("els: disk buffer full (%d bytes), dropping %d entries", info.Size(), len(entries)))
			}
			return
		}
	}

	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		if c.config.OnError != nil {
			c.config.OnError(fmt.Errorf("els: open buffer file: %w", err))
		}
		return
	}
	defer f.Close()

	enc := json.NewEncoder(f)
	for _, e := range entries {
		if err := enc.Encode(e); err != nil {
			if c.config.OnError != nil {
				c.config.OnError(fmt.Errorf("els: write buffer entry: %w", err))
			}
		}
	}
}

// flushDiskBuffer reads and sends entries previously saved to disk. The file is
// read and removed under diskMu (so concurrent writes go to a fresh file);
// network sends happen without holding the lock.
func (c *Client) flushDiskBuffer() {
	c.diskMu.Lock()
	path := c.bufferFilePath()
	f, err := os.Open(path)
	if err != nil {
		c.diskMu.Unlock()
		return // nothing to flush
	}

	var entries []*ErrorEntry
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024) // 1MB line buffer
	for scanner.Scan() {
		var entry ErrorEntry
		if err := json.Unmarshal(scanner.Bytes(), &entry); err != nil {
			continue // skip malformed entries
		}
		e := entry
		entries = append(entries, &e)
	}
	f.Close()
	_ = os.Remove(path)
	c.diskMu.Unlock()

	if len(entries) == 0 {
		return
	}

	// Send in batches without holding diskMu.
	for i := 0; i < len(entries); i += c.config.BatchSize {
		end := i + c.config.BatchSize
		if end > len(entries) {
			end = len(entries)
		}

		ctx, cancel := context.WithTimeout(context.Background(), c.config.Timeout*time.Duration(c.config.MaxRetries+1))
		err := c.transport.sendBatch(ctx, entries[i:end])
		cancel()

		if err != nil {
			// Still can't send — re-buffer the unsent remainder for next time.
			c.writeToDisk(entries[i:])
			if c.config.OnError != nil {
				c.config.OnError(fmt.Errorf("els: flush disk buffer failed: %w", err))
			}
			return
		}
	}
}
