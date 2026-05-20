// Example: batching + graceful shutdown on SIGINT/SIGTERM.
//
// The background worker batches entries. On shutdown, flush and close the
// client so nothing buffered is lost.
package main

import (
	"errors"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	els "github.com/official-inso/els-go"
)

func main() {
	client, err := els.New(els.Config{
		APIKey:        os.Getenv("ELS_API_KEY"),
		AppSlug:       "worker-demo",
		BatchSize:     20,
		BatchInterval: 2 * time.Second,
	})
	if err != nil {
		panic(err)
	}

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-stop
		fmt.Println("shutting down, flushing ELS...")
		client.FlushWithTimeout(5 * time.Second)
		_ = client.Close()
		os.Exit(0)
	}()

	for i := 0; i < 50; i++ {
		client.CaptureError(errors.New("background job failed"),
			els.WithMeta(map[string]any{"job": i}),
		)
		time.Sleep(100 * time.Millisecond)
	}

	client.Flush()
	_ = client.Close()
}
