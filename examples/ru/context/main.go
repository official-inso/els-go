// Пример: проброс request/trace ID через context.Context.
//
// Задайте ID один раз (например, в HTTP-middleware), затем вызывайте методы
// *Ctx — они автоматически копируют requestId/traceId в Meta записи.
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

	// Оба ID попадут в Meta как "requestId" / "traceId".
	client.CaptureErrorCtx(ctx, errors.New("оформление заказа не удалось"), els.WithURL("/api/checkout"))
	client.CaptureMessageCtx(ctx, "шаг оформления выполнен", els.LevelInfo)

	client.Flush()
}
