// Package metrics defines the app's Prometheus collectors and a snapshot writer.
//
// There is no HTTP /metrics endpoint. Instead the current values of every
// collector are flushed to a single file at a fixed interval, rewritten in place
// (latest snapshot only, no history). The file is Prometheus text format, so a
// node_exporter textfile collector — or a plain `cat` — can read it. The Go
// runtime and process collectors are registered by the client by default, so
// go_* and process_* metrics come along for free.
package metrics

import (
	"database/sql"
	"errors"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/common/expfmt"
)

var (
	// HTTPRequests counts handled HTTP requests by method, matched route and
	// status. The path label is the route template (not the raw URL), so its
	// cardinality is bounded by the number of routes.
	HTTPRequests = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "http_requests_total",
		Help: "Total number of HTTP requests handled.",
	}, []string{"method", "path", "status"})

	// HTTPDuration is the request latency histogram, labelled the same way minus
	// status (status doesn't change how long a request took).
	HTTPDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "http_request_duration_seconds",
		Help:    "HTTP request latency in seconds.",
		Buckets: prometheus.DefBuckets,
	}, []string{"method", "path"})

	// DBQueries counts database queries by named query and outcome ("ok" or
	// "error"). The query label is the named query, so cardinality stays bounded.
	DBQueries = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "db_queries_total",
		Help: "Total number of database queries executed.",
	}, []string{"query", "status"})

	// DBDuration is the per-query latency histogram.
	DBDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "db_query_duration_seconds",
		Help:    "Database query latency in seconds.",
		Buckets: prometheus.DefBuckets,
	}, []string{"query"})
)

// ObserveQuery records one database query's latency and outcome. A nil error —
// or sql.ErrNoRows, which is an ordinary "not found", not a fault — counts as
// "ok"; anything else is "error".
func ObserveQuery(query string, dur time.Duration, err error) {
	status := "ok"
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		status = "error"
	}
	DBQueries.WithLabelValues(query, status).Inc()
	DBDuration.WithLabelValues(query).Observe(dur.Seconds())
}

// WriteSnapshot gathers every registered collector and writes it to path in
// Prometheus text format. The write is atomic: it renders to a temp file in the
// same directory and renames it into place, so a reader never catches a
// half-written file.
func WriteSnapshot(path string) error {
	families, err := prometheus.DefaultGatherer.Gather()
	if err != nil {
		return err
	}

	tmp, err := os.CreateTemp(filepath.Dir(path), ".metrics-*.tmp")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()

	enc := expfmt.NewEncoder(tmp, expfmt.FmtText)
	for _, mf := range families {
		if encErr := enc.Encode(mf); encErr != nil {
			tmp.Close()
			os.Remove(tmpName)
			return encErr
		}
	}
	if closeErr := tmp.Close(); closeErr != nil {
		os.Remove(tmpName)
		return closeErr
	}
	return os.Rename(tmpName, path)
}

// StartSnapshotWriter writes a snapshot immediately, then once every interval
// for the life of the process. It runs in its own goroutine and only logs
// failures — losing a metrics snapshot must never take the API down. An empty
// path disables it entirely.
func StartSnapshotWriter(path string, interval time.Duration) {
	if path == "" {
		log.Print("metrics: no file configured, snapshot writer disabled")
		return
	}
	write := func() {
		if err := WriteSnapshot(path); err != nil {
			log.Printf("metrics: snapshot write failed: %v", err)
		}
	}
	write()
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for range ticker.C {
			write()
		}
	}()
}
