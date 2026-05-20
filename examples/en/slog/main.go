// Example: integrate ELS with Go's standard log/slog.
//
// Records logged via slog go to ELS. An slog.Error that carries an error
// attribute ("err"/"error") is captured with a full stack trace and the error
// text in Meta — exactly like CaptureError. ServiceName/AppSlug from Config are
// attached automatically, so you don't pass them on every call.
package main

import (
	"errors"
	"log/slog"
	"os"

	els "github.com/official-inso/els-go"
)

func main() {
	client, err := els.New(els.Config{
		APIKey:      os.Getenv("ELS_API_KEY"),
		AppSlug:     "homedata",
		ServiceName: "sensorloader",
	})
	if err != nil {
		slog.Error("init ELS", "err", err)
		os.Exit(1)
	}
	defer client.Close()

	logger := slog.New(els.SlogHandler(client, &els.SlogHandlerOptions{
		AddSource:           true,
		CaptureStackOnError: true,
		// ErrorKeys defaults to {"err", "error"}; override if your codebase
		// uses a different attribute key for errors.
	}))
	slog.SetDefault(logger)

	slog.Info("service started", "port", 8080)
	slog.Warn("cache miss rate high", "rate", 0.42)

	if err := doWork(); err != nil {
		// Captured as an ELS error WITH stack trace + Meta["error"].
		slog.Error("work failed", "err", err, "stage", "ingest")
	}

	client.Flush()
}

func doWork() error {
	return errors.New("sensor stream disconnected")
}
