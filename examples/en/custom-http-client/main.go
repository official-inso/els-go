// Example: provide a custom *http.Client (proxy, timeouts, transport tuning).
package main

import (
	"errors"
	"net/http"
	"os"
	"time"

	els "github.com/official-inso/els-go"
)

func main() {
	httpClient := &http.Client{
		Timeout: 15 * time.Second,
		Transport: &http.Transport{
			MaxIdleConns:        100,
			MaxIdleConnsPerHost: 10,
			IdleConnTimeout:     90 * time.Second,
		},
	}

	client, err := els.New(els.Config{
		APIKey:     os.Getenv("ELS_API_KEY"),
		AppSlug:    "httpclient-demo",
		HTTPClient: httpClient,
	})
	if err != nil {
		panic(err)
	}
	defer client.Close()

	client.CaptureError(errors.New("sent through a custom transport"))
	client.Flush()
}
