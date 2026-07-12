package views

import (
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// rateLimit is a small in-memory per-IP sliding-window limiter — enough to blunt
// OTP/SMS spam in beta (bulk requests from one source). For multi-instance prod
// this would move to a shared store (Redis), but the per-phone OTP cooldown is
// the primary guard regardless.
func rateLimit(max int, window time.Duration) gin.HandlerFunc {
	var (
		mu   sync.Mutex
		hits = make(map[string][]time.Time)
		last time.Time
	)
	return func(c *gin.Context) {
		ip := c.ClientIP()
		now := time.Now()
		cutoff := now.Add(-window)

		mu.Lock()
		// Occasionally drop empty/stale IP buckets so the map can't grow forever.
		if now.Sub(last) > window {
			for k, ts := range hits {
				if len(ts) == 0 || ts[len(ts)-1].Before(cutoff) {
					delete(hits, k)
				}
			}
			last = now
		}
		kept := hits[ip][:0]
		for _, t := range hits[ip] {
			if t.After(cutoff) {
				kept = append(kept, t)
			}
		}
		if len(kept) >= max {
			hits[ip] = kept
			mu.Unlock()
			Abort(c, CodeRateLimited)
			return
		}
		hits[ip] = append(kept, now)
		mu.Unlock()
		c.Next()
	}
}
