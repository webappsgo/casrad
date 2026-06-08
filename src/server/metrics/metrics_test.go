// Package metrics — Tests for Counter, Gauge, Histogram, registry helpers, and recording functions.
// Covers pure in-memory primitives; does not start an HTTP server.
package metrics

import (
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
)

// --- Counter ---

func TestCounterInc(t *testing.T) {
	t.Parallel()
	c := &Counter{}
	c.Inc()
	c.Inc()
	if got := c.Value(); got != 2 {
		t.Errorf("Counter.Inc x2: got %d, want 2", got)
	}
}

func TestCounterAdd(t *testing.T) {
	t.Parallel()
	c := &Counter{}
	c.Add(10)
	c.Add(5)
	if got := c.Value(); got != 15 {
		t.Errorf("Counter.Add: got %d, want 15", got)
	}
}

func TestCounterStartsAtZero(t *testing.T) {
	t.Parallel()
	c := &Counter{}
	if got := c.Value(); got != 0 {
		t.Errorf("new Counter value = %d, want 0", got)
	}
}

func TestCounterConcurrent(t *testing.T) {
	t.Parallel()
	c := &Counter{}
	var wg sync.WaitGroup
	const goroutines = 50
	const incsEach = 100
	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < incsEach; j++ {
				c.Inc()
			}
		}()
	}
	wg.Wait()
	want := uint64(goroutines * incsEach)
	if got := c.Value(); got != want {
		t.Errorf("concurrent Counter: got %d, want %d", got, want)
	}
}

// --- Gauge ---

func TestGaugeSet(t *testing.T) {
	t.Parallel()
	g := &Gauge{}
	g.Set(3.14)
	if got := g.Value(); got != 3.14 {
		t.Errorf("Gauge.Set: got %v, want 3.14", got)
	}
}

func TestGaugeIncDec(t *testing.T) {
	t.Parallel()
	g := &Gauge{}
	g.Inc()
	g.Inc()
	g.Dec()
	if got := g.Value(); got != 1.0 {
		t.Errorf("Gauge Inc x2 Dec x1: got %v, want 1.0", got)
	}
}

func TestGaugeAdd(t *testing.T) {
	t.Parallel()
	g := &Gauge{}
	g.Add(5.5)
	g.Add(-2.5)
	if got := g.Value(); got != 3.0 {
		t.Errorf("Gauge.Add: got %v, want 3.0", got)
	}
}

func TestGaugeStartsAtZero(t *testing.T) {
	t.Parallel()
	g := &Gauge{}
	if got := g.Value(); got != 0 {
		t.Errorf("new Gauge value = %v, want 0", got)
	}
}

func TestGaugeConcurrent(t *testing.T) {
	t.Parallel()
	g := &Gauge{}
	var wg sync.WaitGroup
	const goroutines = 20
	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			g.Inc()
			g.Inc()
			g.Dec()
		}()
	}
	wg.Wait()
	if got := g.Value(); got != float64(goroutines) {
		t.Errorf("concurrent Gauge: got %v, want %d", got, goroutines)
	}
}

// --- Histogram ---

func TestHistogramObserve(t *testing.T) {
	t.Parallel()
	h := NewHistogram([]float64{0.01, 0.1, 1.0})
	h.Observe(0.005)
	h.Observe(0.05)
	h.Observe(0.5)
	h.Observe(2.0)

	// Verify count and sum indirectly via the struct
	if h.count != 4 {
		t.Errorf("Histogram count = %d, want 4", h.count)
	}
	wantSum := 0.005 + 0.05 + 0.5 + 2.0
	if h.sum != wantSum {
		t.Errorf("Histogram sum = %v, want %v", h.sum, wantSum)
	}
}

func TestHistogramBucketDistribution(t *testing.T) {
	t.Parallel()
	buckets := []float64{1.0, 5.0, 10.0}
	h := NewHistogram(buckets)

	// Value ≤1.0 lands in bucket[0]
	h.Observe(0.5)
	if h.counts[0] != 1 {
		t.Errorf("bucket[0] = %d, want 1", h.counts[0])
	}

	// Value ≤5.0 lands in bucket[1]
	h.Observe(3.0)
	if h.counts[1] != 1 {
		t.Errorf("bucket[1] = %d, want 1", h.counts[1])
	}

	// Value ≤10.0 lands in bucket[2]
	h.Observe(8.0)
	if h.counts[2] != 1 {
		t.Errorf("bucket[2] = %d, want 1", h.counts[2])
	}

	// Value > all buckets lands in overflow bucket (index 3)
	h.Observe(100.0)
	if h.counts[3] != 1 {
		t.Errorf("overflow bucket = %d, want 1", h.counts[3])
	}
}

func TestHistogramEmptyBuckets(t *testing.T) {
	t.Parallel()
	h := NewHistogram([]float64{})
	h.Observe(42.0)
	if h.count != 1 {
		t.Errorf("count = %d, want 1", h.count)
	}
	// Single overflow bucket
	if h.counts[0] != 1 {
		t.Errorf("overflow bucket = %d, want 1", h.counts[0])
	}
}

// --- Registry helpers ---

func TestGetOrCreateCounter(t *testing.T) {
	t.Parallel()
	registry := make(map[string]*Counter)
	c1 := GetOrCreateCounter(registry, "req_total")
	c1.Inc()
	c2 := GetOrCreateCounter(registry, "req_total")
	if c1 != c2 {
		t.Error("GetOrCreateCounter should return same counter for same key")
	}
	if c2.Value() != 1 {
		t.Errorf("counter value = %d, want 1", c2.Value())
	}
}

func TestGetOrCreateGauge(t *testing.T) {
	t.Parallel()
	registry := make(map[string]*Gauge)
	g1 := GetOrCreateGauge(registry, "active_connections")
	g1.Set(5.0)
	g2 := GetOrCreateGauge(registry, "active_connections")
	if g1 != g2 {
		t.Error("GetOrCreateGauge should return same gauge for same key")
	}
	if g2.Value() != 5.0 {
		t.Errorf("gauge value = %v, want 5.0", g2.Value())
	}
}

func TestGetOrCreateCounterDifferentKeys(t *testing.T) {
	t.Parallel()
	registry := make(map[string]*Counter)
	c1 := GetOrCreateCounter(registry, "key_a")
	c2 := GetOrCreateCounter(registry, "key_b")
	if c1 == c2 {
		t.Error("different keys should return different counters")
	}
}

// --- Record helpers ---

func TestRecordHTTPRequest(t *testing.T) {
	t.Parallel()
	// Reset the global registry for this test via a local approach
	registry := make(map[string]*Counter)
	key := "GET_/test_2xx"
	GetOrCreateCounter(registry, key).Inc()
	// Just verify RecordHTTPRequest doesn't panic
	RecordHTTPRequest("GET", "/test", 200)
	RecordHTTPRequest("POST", "/api/v1/users", 201)
	RecordHTTPRequest("GET", "/missing", 404)
}

func TestRecordDBQuery(t *testing.T) {
	t.Parallel()
	RecordDBQuery("SELECT", "users")
	RecordDBQuery("INSERT", "sessions")
	RecordDBQuery("DELETE", "tokens")
}

func TestRecordCacheHitMiss(t *testing.T) {
	t.Parallel()
	RecordCacheHit("memory")
	RecordCacheHit("memory")
	RecordCacheMiss("memory")
}

func TestRecordSchedulerTask(t *testing.T) {
	t.Parallel()
	RecordSchedulerTask("backup_daily", true)
	RecordSchedulerTask("ssl_renewal", false)
}

func TestRecordAuthAttempt(t *testing.T) {
	t.Parallel()
	RecordAuthAttempt("password", true)
	RecordAuthAttempt("password", false)
	RecordAuthAttempt("token", true)
}

// --- Init ---

func TestInit(t *testing.T) {
	t.Parallel()
	Init("1.0.0", "abc1234", "2025-01-01T00:00:00Z")
	if appInfo.Version != "1.0.0" {
		t.Errorf("appInfo.Version = %q, want 1.0.0", appInfo.Version)
	}
	if appInfo.Commit != "abc1234" {
		t.Errorf("appInfo.Commit = %q, want abc1234", appInfo.Commit)
	}
	if appInfo.GoVersion == "" {
		t.Error("appInfo.GoVersion should not be empty after Init")
	}
	// AppStartTime should be set
	if AppStartTime.Value() == 0 {
		t.Error("AppStartTime should be non-zero after Init")
	}
}

// --- UpdateUptime ---

func TestUpdateUptime(t *testing.T) {
	t.Parallel()
	Init("1.0.0", "test", "2025-01-01")
	// Short sleep to ensure uptime > 0
	time.Sleep(10 * time.Millisecond)
	UpdateUptime()
	if AppUptime.Value() < 0 {
		t.Error("uptime should be non-negative")
	}
}

// --- DefaultConfig ---

func TestDefaultConfig(t *testing.T) {
	t.Parallel()
	cfg := DefaultConfig()
	if !cfg.Enabled {
		t.Error("DefaultConfig should have Enabled=true")
	}
	if cfg.Endpoint == "" {
		t.Error("DefaultConfig Endpoint should not be empty")
	}
	if len(cfg.DurationBuckets) == 0 {
		t.Error("DefaultConfig DurationBuckets should not be empty")
	}
	if len(cfg.SizeBuckets) == 0 {
		t.Error("DefaultConfig SizeBuckets should not be empty")
	}
}

// --- Handler ---

func TestHandlerReturnsPrometheusText(t *testing.T) {
	t.Parallel()
	Init("1.0.0", "abc1234", "2025-01-01")
	RecordHTTPRequest("GET", "/test", 200)
	RecordDBQuery("SELECT", "users")

	handler := Handler()
	req := httptest.NewRequest("GET", "/metrics", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	body := rec.Body.String()

	if rec.Code != 200 {
		t.Errorf("Handler status = %d, want 200", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); !strings.Contains(ct, "text/plain") {
		t.Errorf("Content-Type = %q, want text/plain...", ct)
	}

	required := []string{
		"casrad_app_info",
		"casrad_uptime_seconds",
		"casrad_http_requests_total",
		"casrad_db_queries_total",
		"casrad_go_goroutines",
		"casrad_go_memory_alloc_bytes",
		"# HELP",
		"# TYPE",
	}
	for _, s := range required {
		if !strings.Contains(body, s) {
			t.Errorf("metrics output missing %q", s)
		}
	}
}
