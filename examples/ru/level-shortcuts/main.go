// Пример: логгер-стиль шорткатов уровней.
//
// Debug/Info/Warning/Error/Critical — тонкие обёртки над CaptureMessage.
package main

import (
	"os"

	els "github.com/official-inso/els-go"
)

func main() {
	client, err := els.New(els.Config{
		APIKey:  os.Getenv("ELS_API_KEY"),
		AppSlug: "shortcuts-demo",
	})
	if err != nil {
		panic(err)
	}
	defer client.Close()

	client.Debug("подробная деталь")
	client.Info("пользователь вошёл", els.WithMeta(map[string]any{"userId": 42}))
	client.Warning("бюджет ретраев на исходе")
	client.Error("валидация не прошла") // *сообщение* уровня error
	client.Critical("диск заполнен")

	// Чтобы захватить Go-error СО стек-трейсом, используйте CaptureError:
	//   client.CaptureError(err)

	client.Flush()
}
