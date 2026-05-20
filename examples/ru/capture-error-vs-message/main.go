// Пример: CaptureError vs CaptureMessage vs SendSync.
package main

import (
	"context"
	"errors"
	"log"
	"os"

	els "github.com/official-inso/els-go"
)

func main() {
	client, err := els.New(els.Config{
		APIKey:  os.Getenv("ELS_API_KEY"),
		AppSlug: "capture-demo",
	})
	if err != nil {
		panic(err)
	}
	defer client.Close()

	// 1) CaptureError — асинхронно, автоматический стек-трейс с места вызова.
	client.CaptureError(errors.New("таймаут апстрима"),
		els.WithURL("/api/orders"),
		els.WithLevel(els.LevelError),
	)

	// 2) CaptureMessage — асинхронно, без стека, явный уровень.
	client.CaptureMessage("повтор запроса", els.LevelWarning,
		els.WithMeta(map[string]any{"attempt": 2}),
	)

	// 3) SendSync — блокирует до доставки. Для критичных событий, которые нельзя потерять.
	ctx := context.Background()
	if err := client.SendSync(ctx, errors.New("не удалось списать оплату"),
		els.WithURL("/api/payments"),
		els.WithLevel(els.LevelCritical),
	); err != nil {
		if els.IsRetryableErr(err) {
			log.Printf("повторяемая ошибка ELS: %v", err)
		} else {
			log.Printf("постоянная ошибка ELS: %v", err)
		}
	}

	client.Flush()
}
