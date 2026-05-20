// Пример: типизированные уровни и маппинг slog<->ELS.
//
// Level — типизированная строка с методами (String, Valid, Priority, ToSlog),
// плюс пакет даёт LevelFromSlog для обратного направления.
package main

import (
	"fmt"
	"log/slog"
	"os"

	els "github.com/official-inso/els-go"
)

func main() {
	client, err := els.New(els.Config{
		APIKey:   os.Getenv("ELS_API_KEY"),
		AppSlug:  "levels-demo",
		MinLevel: els.LevelInfo, // записи debug отбрасываются до отправки
	})
	if err != nil {
		panic(err)
	}
	defer client.Close()

	levels := []els.Level{
		els.LevelDebug, els.LevelInfo, els.LevelWarning, els.LevelError, els.LevelCritical,
	}
	for _, lvl := range levels {
		fmt.Printf("%-8s valid=%v priority=%d slog=%v\n",
			lvl, lvl.Valid(), lvl.Priority(), lvl.ToSlog())
		client.CaptureMessage("демо уровня: "+lvl.String(), lvl)
	}

	// Маппинг slog.Level в уровень ELS (удобно в кастомных хендлерах).
	fmt.Println("slog.LevelError ->", els.LevelFromSlog(slog.LevelError))

	client.Flush()
}
