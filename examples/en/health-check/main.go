// Example: use Health() as a readiness probe.
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
		fmt.Printf("ELS not reachable: %v (entries will be buffered)\n", err)
		return
	}
	fmt.Println("ELS reachable")
}
