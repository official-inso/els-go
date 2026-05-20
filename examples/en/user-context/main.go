// Example: attach user context + metadata to every entry.
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

	// All subsequent captures include this user under Meta["user.*"].
	client.SetUser(&els.UserContext{
		ID:    "u_123",
		Email: "ops@homedata.io",
		Name:  "Ops Bot",
		Extra: map[string]string{"plan": "enterprise"},
	})

	client.CaptureError(errors.New("export failed"),
		els.WithMeta(map[string]any{"reportId": "rep_42"}),
	)

	client.Flush()
}
