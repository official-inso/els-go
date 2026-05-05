package els

import (
	"fmt"
	"net/http"
)

// Middleware returns an http.Handler that recovers from panics in the
// wrapped handler and reports them as critical errors to ELS.
// The panic is re-raised after capture so upstream recovery middleware
// can handle the HTTP response.
func (c *Client) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rv := recover(); rv != nil {
				msg := fmt.Sprintf("panic: %v", rv)
				c.CaptureError(
					fmt.Errorf("%s", msg),
					WithLevel(LevelCritical),
					WithURL(r.URL.String()),
					WithUserAgent(r.UserAgent()),
					WithReferrer(r.Referer()),
				)
				panic(rv) // re-raise
			}
		}()
		next.ServeHTTP(w, r)
	})
}

// MiddlewareFunc is a convenience wrapper that accepts an http.HandlerFunc.
func (c *Client) MiddlewareFunc(next http.HandlerFunc) http.Handler {
	return c.Middleware(next)
}
