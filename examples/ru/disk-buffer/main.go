// Пример: дисковая буферизация для офлайн-устойчивости.
//
// Когда сервер ELS недоступен, записи сохраняются в BufferDir и отправляются
// при следующей успешной отправке, а не теряются.
package main

import (
	"errors"
	"os"

	els "github.com/official-inso/els-go"
)

func main() {
	client, err := els.New(els.Config{
		APIKey:            os.Getenv("ELS_API_KEY"),
		AppSlug:           "offline-demo",
		BufferDir:         "./.els-buffer",  // записи сохраняются сюда при офлайне
		MaxBufferFileSize: 50 * 1024 * 1024, // ограничение буфера на диске — 50 МБ
	})
	if err != nil {
		panic(err)
	}
	defer client.Close()

	client.CaptureError(errors.New("записано, даже если ELS сейчас недоступен"))
	client.Flush()
}
