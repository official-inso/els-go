// Пример: использование Health() как readiness-пробы.
package main

import (
	"context"
	"fmt"
	"os"
	"time"

	els "github.com/official-inso/els-go"
)

func main() {
	client, err := els.New(els.Config{
		APIKey:  os.Getenv("ELS_API_KEY"),
		AppSlug: "health-demo",
	})
	if err != nil {
		panic(err)
	}
	defer client.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	if err := client.Health(ctx); err != nil {
		fmt.Printf("ELS недоступен: %v (записи будут буферизованы)\n", err)
		return
	}
	fmt.Println("ELS доступен")
}
