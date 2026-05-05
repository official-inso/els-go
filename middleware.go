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
					WithURL(r.Method+" "+r.URL.String()),
					WithUserAgent(r.UserAgent()),
					WithReferrer(r.Referer()),
					WithMeta(map[string]any{
						"method":     r.Method,
						"host":       r.Host,
						"remoteAddr": r.RemoteAddr,
						"proto":      r.Proto,
					}),
				)
				panic(rv) // re-raise for upstream recovery
			}
		}()
		next.ServeHTTP(w, r)
	})
}

// MiddlewareFunc is a convenience wrapper that accepts an http.HandlerFunc.
func (c *Client) MiddlewareFunc(next http.HandlerFunc) http.Handler {
	return c.Middleware(next)
}

// RecoverMiddleware is like Middleware but does NOT re-raise the panic.
// Instead, it returns a 500 Internal Server Error response.
// Use this as a standalone recovery middleware when you don't have another
// recovery layer above.
func (c *Client) RecoverMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rv := recover(); rv != nil {
				msg := fmt.Sprintf("panic: %v", rv)
				c.CaptureError(
					fmt.Errorf("%s", msg),
					WithLevel(LevelCritical),
					WithURL(r.Method+" "+r.URL.String()),
					WithUserAgent(r.UserAgent()),
					WithReferrer(r.Referer()),
					WithMeta(map[string]any{
						"method":     r.Method,
						"host":       r.Host,
						"remoteAddr": r.RemoteAddr,
						"proto":      r.Proto,
					}),
				)
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			}
		}()
		next.ServeHTTP(w, r)
	})
}
