package middleware

import (
	"context"
	"net/http"
)

type Throttler struct {
	sem chan struct{}
}

func NewThrottler(maxConcurrent int) *Throttler {
	return &Throttler{
		sem: make(chan struct{}, maxConcurrent),
	}
}

func (t *Throttler) Wrap(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithCancel(r.Context())
		defer cancel()

		select {
		case t.sem <- struct{}{}:
			defer func() { <-t.sem }()
			next.ServeHTTP(w, r.WithContext(ctx))

		case <-ctx.Done():
			http.Error(w, "Request timeout waiting for throttle slot", http.StatusTooManyRequests)
		}
	})
}
