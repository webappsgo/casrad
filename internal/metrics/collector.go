package metrics

import (
	"fmt"
	"log"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/casapps/casrad/internal/database"
)

// Collector handles metrics collection
type Collector struct {
	db       *database.Engine
	interval time.Duration
	enabled  bool
	stopChan chan struct{}
	mu       sync.RWMutex

	// Current metrics
	metrics     map[string]*Metric
	aggregated  map[string]*AggregatedMetric
}

// Metric represents a single metric
type Metric struct {
	Type       string
	Name       string
	Value      float64
	Unit       string
	Dimensions map[string]string
	Timestamp  time.Time
}

// AggregatedMetric represents aggregated metrics
type AggregatedMetric struct {
	Type      string
	Name      string
	Period    string
	StartTime time.Time
	EndTime   time.Time

	Count    int
	Sum      float64
	Min      float64
	Max      float64
	Avg      float64

	// Percentiles
	P50 float64
	P90 float64
	P95 float64
	P99 float64

	Values []float64 // For percentile calculation
}

// NewCollector creates a new metrics collector
func NewCollector(db *database.Engine) *Collector {
	return &Collector{
		db:         db,
		interval:   5 * time.Minute,
		enabled:    true,
		stopChan:   make(chan struct{}),
		metrics:    make(map[string]*Metric),
		aggregated: make(map[string]*AggregatedMetric),
	}
}

// Start starts the metrics collector
func (c *Collector) Start() {
	if !c.enabled {
		return
	}

	go c.collectLoop()
	go c.aggregateLoop()

	log.Println("Metrics collector started")
}

// Stop stops the metrics collector
func (c *Collector) Stop() {
	c.enabled = false
	close(c.stopChan)
	log.Println("Metrics collector stopped")
}

// collectLoop continuously collects metrics
func (c *Collector) collectLoop() {
	ticker := time.NewTicker(c.interval)
	defer ticker.Stop()

	// Collect immediately on start
	c.collectSystemMetrics()
	c.collectApplicationMetrics()

	for {
		select {
		case <-ticker.C:
			c.collectSystemMetrics()
			c.collectApplicationMetrics()
		case <-c.stopChan:
			return
		}
	}
}

// collectSystemMetrics collects system-level metrics
func (c *Collector) collectSystemMetrics() {
	// Memory metrics
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	c.recordMetric("system", "memory.alloc", float64(memStats.Alloc), "bytes", nil)
	c.recordMetric("system", "memory.total_alloc", float64(memStats.TotalAlloc), "bytes", nil)
	c.recordMetric("system", "memory.sys", float64(memStats.Sys), "bytes", nil)
	c.recordMetric("system", "memory.heap_alloc", float64(memStats.HeapAlloc), "bytes", nil)
	c.recordMetric("system", "memory.heap_inuse", float64(memStats.HeapInuse), "bytes", nil)
	c.recordMetric("system", "memory.stack_inuse", float64(memStats.StackInuse), "bytes", nil)

	// GC metrics
	c.recordMetric("system", "gc.num_gc", float64(memStats.NumGC), "count", nil)
	c.recordMetric("system", "gc.pause_total_ns", float64(memStats.PauseTotalNs), "nanoseconds", nil)

	// Goroutine metrics
	c.recordMetric("system", "goroutines", float64(runtime.NumGoroutine()), "count", nil)

	// CPU metrics
	c.recordMetric("system", "cpu.num", float64(runtime.NumCPU()), "count", nil)
}

// collectApplicationMetrics collects application-specific metrics
func (c *Collector) collectApplicationMetrics() {
	// Database metrics - simplified for now
	// In production, would get actual DB stats
	c.recordMetric("database", "connections.open", 0, "count", nil)
	c.recordMetric("database", "connections.in_use", 0, "count", nil)

	// User metrics
	var userCount int
	c.db.QueryRow("SELECT COUNT(*) FROM users WHERE is_active = 1").Scan(&userCount)
	c.recordMetric("application", "users.active", float64(userCount), "count", nil)

	// Session metrics
	var sessionCount int
	c.db.QueryRow("SELECT COUNT(*) FROM sessions WHERE is_active = 1").Scan(&sessionCount)
	c.recordMetric("application", "sessions.active", float64(sessionCount), "count", nil)

	// Track metrics
	var trackCount int64
	var totalSize int64
	c.db.QueryRow("SELECT COUNT(*), COALESCE(SUM(file_size), 0) FROM tracks").Scan(&trackCount, &totalSize)
	c.recordMetric("application", "tracks.count", float64(trackCount), "count", nil)
	c.recordMetric("application", "tracks.total_size", float64(totalSize), "bytes", nil)

	// Playlist metrics
	var playlistCount int
	c.db.QueryRow("SELECT COUNT(*) FROM playlists").Scan(&playlistCount)
	c.recordMetric("application", "playlists.count", float64(playlistCount), "count", nil)

	// Streaming metrics
	var activeStreams int
	c.db.QueryRow("SELECT COUNT(*) FROM broadcasts WHERE is_active = 1").Scan(&activeStreams)
	c.recordMetric("streaming", "streams.active", float64(activeStreams), "count", nil)

	var totalListeners int
	c.db.QueryRow("SELECT COALESCE(SUM(listeners_current), 0) FROM broadcasts WHERE is_active = 1").Scan(&totalListeners)
	c.recordMetric("streaming", "listeners.current", float64(totalListeners), "count", nil)

	// Podcast metrics
	var podcastCount int
	c.db.QueryRow("SELECT COUNT(*) FROM podcasts WHERE is_active = 1").Scan(&podcastCount)
	c.recordMetric("application", "podcasts.active", float64(podcastCount), "count", nil)

	// API metrics (from last hour)
	var apiRequests int
	c.db.QueryRow(`
		SELECT COUNT(*) FROM audit_log
		WHERE event_category = 'api' AND created_at > ?
	`, time.Now().Add(-time.Hour)).Scan(&apiRequests)
	c.recordMetric("api", "requests.hourly", float64(apiRequests), "count", nil)

	// Storage metrics
	c.collectStorageMetrics()

	// Cache metrics
	c.collectCacheMetrics()
}

// collectStorageMetrics collects storage usage metrics
func (c *Collector) collectStorageMetrics() {
	// User storage usage
	rows, err := c.db.Query(`
		SELECT
			SUM(storage_used_bytes) as total_used,
			SUM(storage_quota_bytes) as total_quota,
			COUNT(*) as user_count
		FROM users
	`)
	if err == nil {
		defer rows.Close()
		if rows.Next() {
			var totalUsed, totalQuota int64
			var userCount int
			rows.Scan(&totalUsed, &totalQuota, &userCount)

			c.recordMetric("storage", "users.total_used", float64(totalUsed), "bytes", nil)
			c.recordMetric("storage", "users.total_quota", float64(totalQuota), "bytes", nil)
			c.recordMetric("storage", "users.utilization", float64(totalUsed)/float64(totalQuota)*100, "percent", nil)
		}
	}

	// Per-type storage
	typeRows, err := c.db.Query(`
		SELECT
			SUM(used_music_bytes) as music,
			SUM(used_podcast_bytes) as podcasts,
			SUM(used_audiobook_bytes) as audiobooks,
			SUM(used_recording_bytes) as recordings
		FROM user_storage
	`)
	if err == nil {
		defer typeRows.Close()
		if typeRows.Next() {
			var music, podcasts, audiobooks, recordings int64
			typeRows.Scan(&music, &podcasts, &audiobooks, &recordings)

			c.recordMetric("storage", "music.bytes", float64(music), "bytes", nil)
			c.recordMetric("storage", "podcasts.bytes", float64(podcasts), "bytes", nil)
			c.recordMetric("storage", "audiobooks.bytes", float64(audiobooks), "bytes", nil)
			c.recordMetric("storage", "recordings.bytes", float64(recordings), "bytes", nil)
		}
	}
}

// collectCacheMetrics collects cache metrics
func (c *Collector) collectCacheMetrics() {
	// This would collect actual cache metrics
	// For now, using placeholder values

	c.recordMetric("cache", "hits", 0, "count", nil)
	c.recordMetric("cache", "misses", 0, "count", nil)
	c.recordMetric("cache", "evictions", 0, "count", nil)
	c.recordMetric("cache", "size", 0, "bytes", nil)
}

// recordMetric records a single metric
func (c *Collector) recordMetric(metricType, name string, value float64, unit string, dimensions map[string]string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	key := fmt.Sprintf("%s.%s", metricType, name)

	metric := &Metric{
		Type:       metricType,
		Name:       name,
		Value:      value,
		Unit:       unit,
		Dimensions: dimensions,
		Timestamp:  time.Now(),
	}

	c.metrics[key] = metric

	// Also store in database
	go c.storeMetric(metric)
}

// storeMetric stores a metric in the database
func (c *Collector) storeMetric(metric *Metric) {
	dimensionsJSON := "{}"
	if metric.Dimensions != nil {
		// Convert dimensions to JSON (simplified)
		dimensionsJSON = fmt.Sprintf("%v", metric.Dimensions)
	}

	_, err := c.db.Exec(`
		INSERT INTO metrics (metric_type, metric_name, metric_value, metric_unit, dimensions, collected_at)
		VALUES (?, ?, ?, ?, ?, ?)
	`, metric.Type, metric.Name, metric.Value, metric.Unit, dimensionsJSON, metric.Timestamp)

	if err != nil {
		log.Printf("Failed to store metric: %v", err)
	}
}

// aggregateLoop periodically aggregates metrics
func (c *Collector) aggregateLoop() {
	// Run aggregation every hour
	ticker := time.NewTicker(time.Hour)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			c.aggregateMetrics("hour", time.Now().Add(-time.Hour), time.Now())
			c.aggregateMetrics("day", time.Now().Add(-24*time.Hour), time.Now())
		case <-c.stopChan:
			return
		}
	}
}

// aggregateMetrics aggregates metrics for a period
func (c *Collector) aggregateMetrics(period string, startTime, endTime time.Time) {
	// Get metrics from database
	rows, err := c.db.Query(`
		SELECT metric_type, metric_name, metric_value
		FROM metrics
		WHERE collected_at >= ? AND collected_at < ?
		ORDER BY metric_type, metric_name, collected_at
	`, startTime, endTime)

	if err != nil {
		log.Printf("Failed to query metrics for aggregation: %v", err)
		return
	}
	defer rows.Close()

	aggregates := make(map[string]*AggregatedMetric)

	for rows.Next() {
		var metricType, metricName string
		var value float64

		if err := rows.Scan(&metricType, &metricName, &value); err != nil {
			continue
		}

		key := fmt.Sprintf("%s.%s", metricType, metricName)

		if agg, exists := aggregates[key]; exists {
			agg.Count++
			agg.Sum += value
			if value < agg.Min {
				agg.Min = value
			}
			if value > agg.Max {
				agg.Max = value
			}
			agg.Values = append(agg.Values, value)
		} else {
			aggregates[key] = &AggregatedMetric{
				Type:      metricType,
				Name:      metricName,
				Period:    period,
				StartTime: startTime,
				EndTime:   endTime,
				Count:     1,
				Sum:       value,
				Min:       value,
				Max:       value,
				Values:    []float64{value},
			}
		}
	}

	// Calculate averages and percentiles
	for _, agg := range aggregates {
		if agg.Count > 0 {
			agg.Avg = agg.Sum / float64(agg.Count)
			agg.P50 = c.calculatePercentile(agg.Values, 50)
			agg.P90 = c.calculatePercentile(agg.Values, 90)
			agg.P95 = c.calculatePercentile(agg.Values, 95)
			agg.P99 = c.calculatePercentile(agg.Values, 99)

			// Store aggregated metric
			c.storeAggregatedMetric(agg)
		}
	}

	// Clean up old raw metrics
	c.cleanupOldMetrics()
}

// calculatePercentile calculates a percentile value
func (c *Collector) calculatePercentile(values []float64, percentile float64) float64 {
	if len(values) == 0 {
		return 0
	}

	// Sort values
	for i := 0; i < len(values)-1; i++ {
		for j := i + 1; j < len(values); j++ {
			if values[j] < values[i] {
				values[i], values[j] = values[j], values[i]
			}
		}
	}

	// Calculate percentile index
	index := int(float64(len(values)-1) * percentile / 100)
	return values[index]
}

// storeAggregatedMetric stores an aggregated metric
func (c *Collector) storeAggregatedMetric(agg *AggregatedMetric) {
	_, err := c.db.Exec(`
		INSERT INTO metrics_aggregated (
			metric_type, metric_name, period, period_start, period_end,
			min_value, max_value, avg_value, sum_value, count_value,
			p50_value, p90_value, p95_value, p99_value
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, agg.Type, agg.Name, agg.Period, agg.StartTime, agg.EndTime,
		agg.Min, agg.Max, agg.Avg, agg.Sum, agg.Count,
		agg.P50, agg.P90, agg.P95, agg.P99)

	if err != nil {
		log.Printf("Failed to store aggregated metric: %v", err)
	}
}

// cleanupOldMetrics removes old raw metrics
func (c *Collector) cleanupOldMetrics() {
	retentionDays := 7 // Keep raw metrics for 7 days

	if val, err := c.db.GetSetting("metrics.retention_days"); err == nil {
		fmt.Sscanf(val, "%d", &retentionDays)
	}

	cutoff := time.Now().AddDate(0, 0, -retentionDays)

	result, err := c.db.Exec(`
		DELETE FROM metrics
		WHERE collected_at < ?
	`, cutoff)

	if err == nil {
		if affected, _ := result.RowsAffected(); affected > 0 {
			log.Printf("Cleaned up %d old metrics", affected)
		}
	}
}

// GetCurrentMetrics returns current metrics
func (c *Collector) GetCurrentMetrics() map[string]*Metric {
	c.mu.RLock()
	defer c.mu.RUnlock()

	// Return copy to avoid race conditions
	metrics := make(map[string]*Metric)
	for k, v := range c.metrics {
		metrics[k] = v
	}

	return metrics
}

// GetMetricsJSON returns metrics in JSON format
func (c *Collector) GetMetricsJSON() map[string]interface{} {
	metrics := c.GetCurrentMetrics()

	result := make(map[string]interface{})
	for key, metric := range metrics {
		result[key] = map[string]interface{}{
			"value":     metric.Value,
			"unit":      metric.Unit,
			"timestamp": metric.Timestamp,
		}
	}

	return result
}

// GetPrometheusMetrics returns metrics in Prometheus format
func (c *Collector) GetPrometheusMetrics() string {
	metrics := c.GetCurrentMetrics()

	output := ""
	for key, metric := range metrics {
		// Convert metric name to Prometheus format
		promName := "casrad_" + strings.ReplaceAll(key, ".", "_")

		// Add HELP and TYPE comments
		output += fmt.Sprintf("# HELP %s %s in %s\n", promName, metric.Name, metric.Unit)
		output += fmt.Sprintf("# TYPE %s gauge\n", promName)

		// Add metric value
		if metric.Dimensions != nil && len(metric.Dimensions) > 0 {
			labels := ""
			for k, v := range metric.Dimensions {
				if labels != "" {
					labels += ","
				}
				labels += fmt.Sprintf(`%s="%s"`, k, v)
			}
			output += fmt.Sprintf("%s{%s} %f\n", promName, labels, metric.Value)
		} else {
			output += fmt.Sprintf("%s %f\n", promName, metric.Value)
		}
		output += "\n"
	}

	return output
}

// Enable enables metrics collection
func (c *Collector) Enable() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.enabled = true
}

// Disable disables metrics collection
func (c *Collector) Disable() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.enabled = false
}

// IsEnabled returns whether metrics collection is enabled
func (c *Collector) IsEnabled() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.enabled
}

// SetInterval sets the collection interval
func (c *Collector) SetInterval(interval time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.interval = interval
}