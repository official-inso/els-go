// Пример: минимально возможная интеграция с ELS.
//
// Обязательны только APIKey и AppSlug — endpoint захардкожен в SDK.
package main

import (
	"log"
	"os"

	els "github.com/official-inso/els-go"
)

func main() {
	client, err := els.New(els.Config{
		APIKey:  os.Getenv("ELS_API_KEY"),
		AppSlug: "quickstart",
	})
	if err != nil {
		log.Fatalf("инициализация ELS: %v", err)
	}
	defer client.Close()

	client.CaptureMessage("привет из els-go", els.LevelInfo)
	client.Flush()
}
