package views

import (
	"strconv"
	"time"

	"coachwise/src/metrics"

	"github.com/gin-gonic/gin"
)

// Metrics records every request's count and latency into the Prometheus
// collectors. It labels by the matched route template (FullPath) rather than the
// raw URL, so /users/:id collapses to a single series instead of one per id; a
// request that matched no route is labelled "unmatched".
func Metrics() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()

		path := c.FullPath()
		if path == "" {
			path = "unmatched"
		}
		status := strconv.Itoa(c.Writer.Status())
		metrics.HTTPRequests.WithLabelValues(c.Request.Method, path, status).Inc()
		metrics.HTTPDuration.WithLabelValues(c.Request.Method, path).Observe(time.Since(start).Seconds())
	}
}
