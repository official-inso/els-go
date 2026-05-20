// Пример: хук BeforeSend для маскировки PII или отбрасывания записей.
//
// BeforeSend вызывается для каждой записи прямо перед постановкой в очередь.
// Верните nil, чтобы отбросить запись, или измените её на месте, чтобы вычистить
// чувствительные данные.
package main

import (
	"errors"
	"os"
	"strings"

	els "github.com/official-inso/els-go"
)

func main() {
	client, err := els.New(els.Config{
		APIKey:  os.Getenv("ELS_API_KEY"),
		AppSlug: "redaction-demo",
		BeforeSend: func(e *els.ErrorEntry) *els.ErrorEntry {
			// Полностью отбрасываем debug-шум.
			if e.Level == els.LevelDebug {
				return nil
			}
			// Маскируем email в сообщении.
			if strings.Contains(e.Message, "@") {
				e.Message = "[email скрыт]"
			}
			// Удаляем чувствительный ключ из meta.
			delete(e.Meta, "password")
			return e
		},
	})
	if err != nil {
		panic(err)
	}
	defer client.Close()

	client.CaptureError(errors.New("вход не удался для user@example.com"),
		els.WithMeta(map[string]any{"password": "secret", "ip": "1.2.3.4"}),
	)
	client.CaptureMessage("подробная трассировка", els.LevelDebug) // отброшено BeforeSend

	client.Flush()
}
