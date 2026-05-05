package els

import "sync/atomic"

// Stats holds runtime metrics about the SDK's operation.
// All fields are updated atomically and safe to read concurrently.
type Stats struct {
	// Enqueued is the total number of entries that were accepted into the queue.
	Enqueued int64

	// Dropped is the number of entries dropped due to queue overflow.
	Dropped int64

	// Sent is the total number of entries successfully delivered to the server.
	Sent int64

	// Failed is the number of entries that failed to send (buffered to disk).
	Failed int64

	// Sampled is the number of entries dropped due to sampling.
	Sampled int64
}

// GetStats returns a snapshot of the SDK's runtime metrics.
// Useful for observability dashboards and debugging.
//
//	stats := client.GetStats()
//	log.Printf("ELS: queued=%d sent=%d dropped=%d failed=%d",
//	    stats.Enqueued, stats.Sent, stats.Dropped, stats.Failed)
func (c *Client) GetStats() Stats {
	return Stats{
		Enqueued: atomic.LoadInt64(&c.stats.Enqueued),
		Dropped:  atomic.LoadInt64(&c.stats.Dropped),
		Sent:     atomic.LoadInt64(&c.stats.Sent),
		Failed:   atomic.LoadInt64(&c.stats.Failed),
		Sampled:  atomic.LoadInt64(&c.stats.Sampled),
	}
}

// QueueSize returns the current number of entries waiting in the queue.
func (c *Client) QueueSize() int {
	return len(c.queue)
}
