package metrics

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestWriteSnapshot checks the writer emits a Prometheus-text file containing our
// own collectors plus the free runtime metrics, and leaves no temp file behind
// (the write must be atomic: temp + rename).
func TestWriteSnapshot(t *testing.T) {
	HTTPRequests.WithLabelValues("GET", "/health", "200").Inc()
	HTTPDuration.WithLabelValues("GET", "/health").Observe(0.01)
	ObserveQuery("users_get", 5*time.Millisecond, nil)

	dir := t.TempDir()
	path := filepath.Join(dir, "metrics.prom")
	if err := WriteSnapshot(path); err != nil {
		t.Fatalf("WriteSnapshot: %v", err)
	}

	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read snapshot: %v", err)
	}
	out := string(b)
	for _, want := range []string{
		"http_requests_total",
		"http_request_duration_seconds",
		"db_queries_total",
		"db_query_duration_seconds",
		"go_goroutines", // registered by the client by default
	} {
		if !strings.Contains(out, want) {
			t.Errorf("snapshot missing metric %q", want)
		}
	}

	entries, _ := os.ReadDir(dir)
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), ".metrics-") {
			t.Errorf("leftover temp file not cleaned up: %s", e.Name())
		}
	}
}
