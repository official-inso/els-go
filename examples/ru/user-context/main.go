// Пример: привязка контекста пользователя + метаданных к каждой записи.
package main

import (
	"errors"
	"os"

	els "github.com/official-inso/els-go"
)

func main() {
	client, err := els.New(els.Config{
		APIKey:  os.Getenv("ELS_API_KEY"),
		AppSlug: "user-demo",
	})
	if err != nil {
		panic(err)
	}
	defer client.Close()

	// Все последующие записи будут содержать пользователя в Meta["user.*"].
	client.SetUser(&els.UserContext{
		ID:    "u_123",
		Email: "ops@homedata.io",
		Name:  "Ops Bot",
		Extra: map[string]string{"plan": "enterprise"},
	})

	client.CaptureError(errors.New("экспорт не удался"),
		els.WithMeta(map[string]any{"reportId": "rep_42"}),
	)

	client.Flush()
}
