// Package metrics provides Prometheus-compatible metrics per AI.md PART 21.
// Metrics are exported in Prometheus text format at /api/v1/server/metrics.
// Uses pure-Go implementation — no prometheus/client_golang dependency needed.
package metrics

import (
	"fmt"
	"net/http"
	"runtime"
	"sync"
	"time"
)

// MetricsConfig holds metrics configuration
type MetricsConfig struct {
	Enabled          bool
	Endpoint         string
	IncludeSystem    bool
	IncludeRuntime   bool
	// Optional bearer token for authentication
	Token            string
	DurationBuckets  []float64
	SizeBuckets      []float64
}

// DefaultConfig returns the default metrics configuration
func DefaultConfig() *MetricsConfig {
	return &MetricsConfig{
		Enabled:        true,
		Endpoint:       "/metrics",
		IncludeSystem:  true,
		IncludeRuntime: true,
		DurationBuckets: []float64{0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10},
		SizeBuckets:    []float64{100, 1000, 10000, 100000, 1000000, 10000000},
	}
}

// Counter is a monotonically increasing counter
type Counter struct {
	value uint64
	mu    sync.Mutex
}

// Inc increments the counter by 1
func (c *Counter) Inc() {
	c.mu.Lock()
	c.value++
	c.mu.Unlock()
}

// Add adds the given value to the counter
func (c *Counter) Add(delta uint64) {
	c.mu.Lock()
	c.value += delta
	c.mu.Unlock()
}

// Value returns the current counter value
func (c *Counter) Value() uint64 {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.value
}

// Gauge is a value that can go up and down
type Gauge struct {
	value float64
	mu    sync.Mutex
}

// Set sets the gauge to the given value
func (g *Gauge) Set(value float64) {
	g.mu.Lock()
	g.value = value
	g.mu.Unlock()
}

// Inc increments the gauge by 1
func (g *Gauge) Inc() {
	g.mu.Lock()
	g.value++
	g.mu.Unlock()
}

// Dec decrements the gauge by 1
func (g *Gauge) Dec() {
	g.mu.Lock()
	g.value--
	g.mu.Unlock()
}

// Add adds the given value to the gauge
func (g *Gauge) Add(delta float64) {
	g.mu.Lock()
	g.value += delta
	g.mu.Unlock()
}

// Value returns the current gauge value
func (g *Gauge) Value() float64 {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.value
}

// Histogram tracks the distribution of values
type Histogram struct {
	buckets []float64
	counts  []uint64
	sum     float64
	count   uint64
	mu      sync.Mutex
}

// NewHistogram creates a new histogram with the given buckets
func NewHistogram(buckets []float64) *Histogram {
	return &Histogram{
		buckets: buckets,
		counts:  make([]uint64, len(buckets)+1),
	}
}

// Observe adds a value to the histogram
func (h *Histogram) Observe(value float64) {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.sum += value
	h.count++

	for i, bucket := range h.buckets {
		if value <= bucket {
			h.counts[i]++
			return
		}
	}
	h.counts[len(h.buckets)]++
}

// Global metrics instances per PART 21
var (
	// HTTP metrics
	HTTPRequestsTotal   = make(map[string]*Counter)
	HTTPActiveRequests  = &Gauge{}

	// Database metrics
	DBQueriesTotal      = make(map[string]*Counter)
	DBConnectionsOpen   = &Gauge{}
	DBConnectionsInUse  = &Gauge{}

	// Cache metrics
	CacheHits           = make(map[string]*Counter)
	CacheMisses         = make(map[string]*Counter)
	CacheEvictions      = make(map[string]*Counter)
	CacheSize           = make(map[string]*Gauge)

	// Scheduler metrics
	SchedulerTasksTotal = make(map[string]*Counter)
	SchedulerLastRun    = make(map[string]*Gauge)

	// Auth metrics
	AuthAttemptsTotal   = make(map[string]*Counter)
	AuthSessionsActive  = &Gauge{}

	// Business metrics
	UsersTotal          = &Gauge{}
	UsersActive         = &Gauge{}
	APITokensActive     = &Gauge{}

	// Application metrics
	AppStartTime        = &Gauge{}
	AppUptime           = &Gauge{}

	// Metric registry lock
	metricsLock sync.RWMutex
)

// AppInfo holds application information
var appInfo struct {
	Version   string
	Commit    string
	BuildDate string
	GoVersion string
}

// Init initializes the metrics with application info
func Init(version, commit, buildDate string) {
	appInfo.Version = version
	appInfo.Commit = commit
	appInfo.BuildDate = buildDate
	appInfo.GoVersion = runtime.Version()
	AppStartTime.Set(float64(time.Now().Unix()))
}

// GetOrCreateCounter gets or creates a counter for the given key
func GetOrCreateCounter(registry map[string]*Counter, key string) *Counter {
	metricsLock.Lock()
	defer metricsLock.Unlock()

	if c, ok := registry[key]; ok {
		return c
	}
	c := &Counter{}
	registry[key] = c
	return c
}

// GetOrCreateGauge gets or creates a gauge for the given key
func GetOrCreateGauge(registry map[string]*Gauge, key string) *Gauge {
	metricsLock.Lock()
	defer metricsLock.Unlock()

	if g, ok := registry[key]; ok {
		return g
	}
	g := &Gauge{}
	registry[key] = g
	return g
}

// RecordHTTPRequest records an HTTP request metric
func RecordHTTPRequest(method, path string, status int) {
	key := method + "_" + path + "_" + string(rune(status/100+'0')) + "xx"
	GetOrCreateCounter(HTTPRequestsTotal, key).Inc()
}

// RecordDBQuery records a database query metric
func RecordDBQuery(operation, table string) {
	key := operation + "_" + table
	GetOrCreateCounter(DBQueriesTotal, key).Inc()
}

// RecordCacheHit records a cache hit
func RecordCacheHit(cache string) {
	GetOrCreateCounter(CacheHits, cache).Inc()
}

// RecordCacheMiss records a cache miss
func RecordCacheMiss(cache string) {
	GetOrCreateCounter(CacheMisses, cache).Inc()
}

// RecordSchedulerTask records a scheduler task execution
func RecordSchedulerTask(task string, success bool) {
	status := "success"
	if !success {
		status = "failed"
	}
	key := task + "_" + status
	GetOrCreateCounter(SchedulerTasksTotal, key).Inc()
	GetOrCreateGauge(SchedulerLastRun, task).Set(float64(time.Now().Unix()))
}

// RecordAuthAttempt records an authentication attempt
func RecordAuthAttempt(method string, success bool) {
	status := "success"
	if !success {
		status = "failed"
	}
	key := method + "_" + status
	GetOrCreateCounter(AuthAttemptsTotal, key).Inc()
}

// UpdateUptime updates the uptime metric
func UpdateUptime() {
	uptime := time.Since(time.Unix(int64(AppStartTime.Value()), 0))
	AppUptime.Set(uptime.Seconds())
}

// Handler returns an HTTP handler that serves Prometheus-format metrics.
// Metrics are exposed at /api/v1/server/metrics per AI.md PART 21.
func Handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		UpdateUptime()

		var ms runtime.MemStats
		runtime.ReadMemStats(&ms)

		w.Header().Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")

		metricsLock.RLock()
		defer metricsLock.RUnlock()

		// App info gauge
		fmt.Fprintf(w, "# HELP casrad_app_info Application version info\n")
		fmt.Fprintf(w, "# TYPE casrad_app_info gauge\n")
		fmt.Fprintf(w, "casrad_app_info{version=%q,commit=%q,go_version=%q} 1\n",
			appInfo.Version, appInfo.Commit, appInfo.GoVersion)

		// Uptime
		fmt.Fprintf(w, "# HELP casrad_uptime_seconds Seconds since application start\n")
		fmt.Fprintf(w, "# TYPE casrad_uptime_seconds gauge\n")
		fmt.Fprintf(w, "casrad_uptime_seconds %.2f\n", AppUptime.Value())

		// HTTP requests
		fmt.Fprintf(w, "# HELP casrad_http_requests_total Total HTTP requests by method/path/status\n")
		fmt.Fprintf(w, "# TYPE casrad_http_requests_total counter\n")
		for k, c := range HTTPRequestsTotal {
			fmt.Fprintf(w, "casrad_http_requests_total{key=%q} %d\n", k, c.Value())
		}

		// DB queries
		fmt.Fprintf(w, "# HELP casrad_db_queries_total Total database queries\n")
		fmt.Fprintf(w, "# TYPE casrad_db_queries_total counter\n")
		for k, c := range DBQueriesTotal {
			fmt.Fprintf(w, "casrad_db_queries_total{key=%q} %d\n", k, c.Value())
		}

		// Cache hits/misses
		fmt.Fprintf(w, "# HELP casrad_cache_hits_total Total cache hits\n")
		fmt.Fprintf(w, "# TYPE casrad_cache_hits_total counter\n")
		for k, c := range CacheHits {
			fmt.Fprintf(w, "casrad_cache_hits_total{cache=%q} %d\n", k, c.Value())
		}
		fmt.Fprintf(w, "# HELP casrad_cache_misses_total Total cache misses\n")
		fmt.Fprintf(w, "# TYPE casrad_cache_misses_total counter\n")
		for k, c := range CacheMisses {
			fmt.Fprintf(w, "casrad_cache_misses_total{cache=%q} %d\n", k, c.Value())
		}

		// Scheduler tasks
		fmt.Fprintf(w, "# HELP casrad_scheduler_tasks_total Scheduler task executions\n")
		fmt.Fprintf(w, "# TYPE casrad_scheduler_tasks_total counter\n")
		for k, c := range SchedulerTasksTotal {
			fmt.Fprintf(w, "casrad_scheduler_tasks_total{task=%q} %d\n", k, c.Value())
		}

		// Auth attempts
		fmt.Fprintf(w, "# HELP casrad_auth_attempts_total Authentication attempts\n")
		fmt.Fprintf(w, "# TYPE casrad_auth_attempts_total counter\n")
		for k, c := range AuthAttemptsTotal {
			fmt.Fprintf(w, "casrad_auth_attempts_total{method=%q} %d\n", k, c.Value())
		}

		// Go runtime metrics
		fmt.Fprintf(w, "# HELP casrad_go_goroutines Number of goroutines\n")
		fmt.Fprintf(w, "# TYPE casrad_go_goroutines gauge\n")
		fmt.Fprintf(w, "casrad_go_goroutines %d\n", runtime.NumGoroutine())

		fmt.Fprintf(w, "# HELP casrad_go_memory_alloc_bytes Allocated heap bytes\n")
		fmt.Fprintf(w, "# TYPE casrad_go_memory_alloc_bytes gauge\n")
		fmt.Fprintf(w, "casrad_go_memory_alloc_bytes %d\n", ms.Alloc)

		fmt.Fprintf(w, "# HELP casrad_go_memory_sys_bytes Total memory obtained from OS\n")
		fmt.Fprintf(w, "# TYPE casrad_go_memory_sys_bytes gauge\n")
		fmt.Fprintf(w, "casrad_go_memory_sys_bytes %d\n", ms.Sys)
	})
}
