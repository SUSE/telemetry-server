package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/SUSE/telemetry-server/app/middleware"
)

// TestThrottler_LimitsConcurrentRequests checks that the Throttler enforces max concurrency
func TestThrottler_LimitsConcurrentRequests(t *testing.T) {
	// Only allow 2 concurrent requests
	throttler := middleware.NewThrottler(2)

	var (
		active     int
		maxActive  int
		mu         sync.Mutex
		wg         sync.WaitGroup
		totalCalls = 5
	)

	// Fake handler simulating 100ms processing
	handler := throttler.Wrap(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		active++
		if active > maxActive {
			maxActive = active
		}
		mu.Unlock()

		time.Sleep(100 * time.Millisecond) // simulate work

		mu.Lock()
		active--
		mu.Unlock()

		w.WriteHeader(http.StatusOK)
	}))

	// Start 5 concurrent requests
	for i := 0; i < totalCalls; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			req := httptest.NewRequest("GET", "/", nil)
			rr := httptest.NewRecorder()
			handler.ServeHTTP(rr, req)

			if rr.Code != http.StatusOK {
				t.Errorf("unexpected status code: got %d, want %d", rr.Code, http.StatusOK)
			}
		}()
	}

	wg.Wait()

	if maxActive > 2 {
		t.Errorf("too many concurrent requests: maxActive = %d, want <= 2", maxActive)
	}
}
