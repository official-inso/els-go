// Пример: package-level (глобальный) фасад — инициализация один раз, захват где угодно.
//
// Удобно для небольших сервисов и CLI, где прокидывать *Client через каждую
// функцию неудобно.
package main

import (
	"context"
	"errors"
	"os"

	els "github.com/official-inso/els-go"
)

func main() {
	if err := els.Init(els.Config{
		APIKey:      os.Getenv("ELS_API_KEY"),
		AppSlug:     "global-demo",
		ServiceName: "cron",
	}); err != nil {
		panic(err)
	}
	defer els.Close()

	// Не нужно прокидывать клиент — используйте package-level хелперы где угодно.
	els.CaptureMessageGlobal("ночная задача началась", els.LevelInfo)
	els.CaptureErrorGlobal(errors.New("ночная задача упала"), els.WithURL("/cron/nightly"))
	_ = els.SendSyncGlobal(context.Background(), errors.New("критический сбой cron"),
		els.WithLevel(els.LevelCritical))

	els.FlushGlobal()
}
