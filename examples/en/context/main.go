// Example: propagate a request/trace ID via context.Context.
//
// Set the IDs once (e.g. in HTTP middleware), then capture with the *Ctx
// methods — they copy requestId/traceId into the entry's Meta automatically.
package main

import (
	"context"
	"errors"
	"os"

	els "github.com/official-inso/els-go"
)

func main() {
	client, err := els.New(els.Config{
		APIKey:  os.Getenv("ELS_API_KEY"),
		AppSlug: "context-demo",
	})
	if err != nil {
		panic(err)
	}
	defer client.Close()

	ctx := els.ContextWithRequestID(context.Background(), "req-123")
	ctx = els.ContextWithTraceID(ctx, "trace-abc")

	// Both IDs land in Meta as "requestId" / "traceId".
	client.CaptureErrorCtx(ctx, errors.New("checkout failed"), els.WithURL("/api/checkout"))
	client.CaptureMessageCtx(ctx, "checkout step done", els.LevelInfo)

	client.Flush()
}
