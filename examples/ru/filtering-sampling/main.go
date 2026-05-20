// Пример: фильтрация по MinLevel и сэмплирование SampleRate.
//
// MinLevel отбрасывает записи ниже порога до отправки. SampleRate отправляет
// лишь часть некритичных записей. Критичные записи всегда минуют сэмплинг.
package main

import (
	"fmt"
	"os"

	els "github.com/official-inso/els-go"
)

func main() {
	client, err := els.New(els.Config{
		APIKey:     os.Getenv("ELS_API_KEY"),
		AppSlug:    "sampling-demo",
		MinLevel:   els.LevelWarning, // debug/info отбрасываются до отправки
		SampleRate: 0.1,              // отправляется ~10% некритичных записей
	})
	if err != nil {
		panic(err)
	}
	defer client.Close()

	for i := 0; i < 100; i++ {
		client.CaptureMessage("шумное предупреждение", els.LevelWarning)
	}
	// Критичная запись минует сэмплинг — доставляется всегда.
	client.CaptureMessage("сбой диска", els.LevelCritical)

	client.Flush()
	fmt.Printf("статистика: %+v\n", client.GetStats())
}
