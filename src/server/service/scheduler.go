// Package service - Scheduled task service with cron parsing
// See AI.md PART 19 for scheduler specification
package service

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"
)

// TaskStatus represents the current status of a task
type TaskStatus string

const (
	TaskStatusPending  TaskStatus = "pending"
	TaskStatusRunning  TaskStatus = "running"
	TaskStatusSuccess  TaskStatus = "success"
	TaskStatusFailed   TaskStatus = "failed"
	TaskStatusSkipped  TaskStatus = "skipped"
	TaskStatusDisabled TaskStatus = "disabled"
)

// TaskType determines how a task runs in cluster mode
type TaskType string

const (
	// Run on ONE node only
	TaskTypeGlobal TaskType = "global"
	// Run on EVERY node
	TaskTypeLocal TaskType = "local"
)

// TaskFunc is a function that executes a scheduled task
type TaskFunc func(ctx context.Context) error

// Task represents a scheduled task
type Task struct {
	ID       string
	Name     string
	// Cron format or @shorthand
	Schedule string
	Handler  TaskFunc
	Enabled  bool
	// global or local
	Type TaskType
	// Cannot be disabled if true
	Critical bool

	// Execution tracking
	Status       TaskStatus
	LastRun      time.Time
	NextRun      time.Time
	LastError    error
	LastDuration time.Duration
	RunCount     int64
	ErrorCount   int64

	// Retry policy
	MaxRetries  int
	RetryDelay  time.Duration
	RetryCount  int
	// Use exponential backoff
	RetryBackoff bool
}

// Scheduler is an alias for SchedulerService for backwards compatibility
type Scheduler = SchedulerService

// SchedulerService manages scheduled tasks
type SchedulerService struct {
	tasks          map[string]*Task
	running        bool
	ctx            context.Context
	cancel         context.CancelFunc
	mu             sync.RWMutex
	wg             sync.WaitGroup
	timezone       *time.Location
	// Run missed tasks if within this window
	catchUpWindow time.Duration
}

// BuiltInTask defines a built-in scheduled task per PART 19
type BuiltInTask struct {
	ID       string
	Name     string
	Schedule string
	Type     TaskType
	Critical bool     // Cannot be disabled
	Enabled  bool     // Default state
}

// BuiltInTasks are the required tasks per AI.md PART 19
var BuiltInTasks = []BuiltInTask{
	// SSL/Certificate renewal - Daily at 03:00
	{"ssl_renewal", "SSL Certificate Renewal", "0 3 * * *", TaskTypeGlobal, true, true},

	// GeoIP update - Weekly Sunday at 03:00
	{"geoip_update", "GeoIP Database Update", "0 3 * * 0", TaskTypeGlobal, false, true},

	// Blocklist update - Daily at 04:00
	{"blocklist_update", "Security Blocklist Update", "0 4 * * *", TaskTypeGlobal, false, true},

	// CVE update - Daily at 05:00
	{"cve_update", "CVE Database Update", "0 5 * * *", TaskTypeGlobal, false, true},

	// Session cleanup - Every 15 minutes
	{"session_cleanup", "Session Cleanup", "@every 15m", TaskTypeLocal, true, true},

	// Token cleanup - Every 15 minutes
	{"token_cleanup", "Token Cleanup", "@every 15m", TaskTypeLocal, true, true},

	// Log rotation - Daily at midnight
	{"log_rotation", "Log Rotation", "0 0 * * *", TaskTypeLocal, true, true},

	// Daily backup - Daily at 02:00
	{"backup_daily", "Daily Backup", "0 2 * * *", TaskTypeGlobal, false, true},

	// Hourly backup - Hourly (disabled by default)
	{"backup_hourly", "Hourly Backup", "@hourly", TaskTypeGlobal, false, false},

	// Self health check - Every 5 minutes
	{"healthcheck_self", "Self Health Check", "@every 5m", TaskTypeLocal, true, true},

	// Tor health check - Every 10 minutes (only if Tor enabled)
	{"tor_health", "Tor Health Check", "@every 10m", TaskTypeLocal, true, true},

	// Cluster heartbeat - Every 30 seconds (cluster mode only)
	{"cluster_heartbeat", "Cluster Heartbeat", "@every 30s", TaskTypeLocal, true, true},

	// CASRAD-specific tasks
	{"update_podcasts", "Podcast Feed Update", "0 */6 * * *", TaskTypeGlobal, false, true},
	{"scan_libraries", "Library Scan", "0 3 * * *", TaskTypeGlobal, false, true},
	{"aggregate_metrics", "Metrics Aggregation", "@every 5m", TaskTypeLocal, false, true},
	{"cleanup_temp", "Temp File Cleanup", "@hourly", TaskTypeLocal, false, true},
	{"cleanup_cache", "Cache Cleanup", "0 */6 * * *", TaskTypeLocal, false, true},
	{"cleanup_transcodes", "Transcode Cleanup", "0 4 * * *", TaskTypeLocal, false, true},
	{"check_quotas", "Quota Check", "*/30 * * * *", TaskTypeLocal, false, true},
}

// NewSchedulerService creates a new scheduler service with built-in tasks
func NewSchedulerService() *SchedulerService {
	s := &SchedulerService{
		tasks:         make(map[string]*Task),
		timezone:      time.Local,
		// 1 hour catch-up window
		catchUpWindow: time.Hour,
	}

	// Register all built-in tasks per PART 19
	for _, bt := range BuiltInTasks {
		s.tasks[bt.ID] = &Task{
			ID:         bt.ID,
			Name:       bt.Name,
			Schedule:   bt.Schedule,
			Enabled:    bt.Enabled,
			Type:       bt.Type,
			Critical:   bt.Critical,
			Status:     TaskStatusPending,
			MaxRetries: 3,
			RetryDelay: 5 * time.Minute,
			RetryBackoff: true,
		}
	}

	return s
}

// RegisterHandler registers a handler for a task
func (s *SchedulerService) RegisterHandler(name string, handler TaskFunc) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if task, ok := s.tasks[name]; ok {
		task.Handler = handler
	}
}

// AddTask adds a new task
func (s *SchedulerService) AddTask(name, schedule string, handler TaskFunc) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Validate schedule
	if _, err := parseCronSchedule(schedule); err != nil {
		return fmt.Errorf("invalid schedule: %w", err)
	}

	s.tasks[name] = &Task{
		Name:     name,
		Schedule: schedule,
		Handler:  handler,
		Enabled:  true,
	}

	return nil
}

// EnableTask enables a task
func (s *SchedulerService) EnableTask(name string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if task, ok := s.tasks[name]; ok {
		task.Enabled = true
	}
}

// DisableTask disables a task
func (s *SchedulerService) DisableTask(name string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if task, ok := s.tasks[name]; ok {
		task.Enabled = false
	}
}

// GetTask returns a task by name
func (s *SchedulerService) GetTask(name string) *Task {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if task, ok := s.tasks[name]; ok {
		return task
	}
	return nil
}

// ListTasks returns all tasks
func (s *SchedulerService) ListTasks() []*Task {
	s.mu.RLock()
	defer s.mu.RUnlock()

	tasks := make([]*Task, 0, len(s.tasks))
	for _, task := range s.tasks {
		tasks = append(tasks, task)
	}
	return tasks
}

// Start starts the scheduler
func (s *SchedulerService) Start() {
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		return
	}
	s.running = true
	s.ctx, s.cancel = context.WithCancel(context.Background())
	s.mu.Unlock()

	// Calculate initial next run times
	s.updateNextRunTimes()

	s.wg.Add(1)
	go s.run()
}

// Stop stops the scheduler
func (s *SchedulerService) Stop() {
	s.mu.Lock()
	if !s.running {
		s.mu.Unlock()
		return
	}
	s.running = false
	s.cancel()
	s.mu.Unlock()

	s.wg.Wait()
}

// IsRunning returns whether the scheduler is running
func (s *SchedulerService) IsRunning() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.running
}

// RunTask manually runs a task
func (s *SchedulerService) RunTask(ctx context.Context, name string) error {
	s.mu.RLock()
	task, ok := s.tasks[name]
	s.mu.RUnlock()

	if !ok {
		return fmt.Errorf("task not found: %s", name)
	}

	if task.Handler == nil {
		return fmt.Errorf("task has no handler: %s", name)
	}

	return s.executeTask(ctx, task)
}

func (s *SchedulerService) run() {
	defer s.wg.Done()

	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-s.ctx.Done():
			return
		case now := <-ticker.C:
			s.checkAndRunTasks(now)
		}
	}
}

func (s *SchedulerService) checkAndRunTasks(now time.Time) {
	s.mu.RLock()
	tasksToRun := make([]*Task, 0)

	for _, task := range s.tasks {
		if task.Enabled && task.Handler != nil && !task.NextRun.IsZero() {
			if now.After(task.NextRun) || now.Equal(task.NextRun) {
				tasksToRun = append(tasksToRun, task)
			}
		}
	}
	s.mu.RUnlock()

	// Run tasks
	for _, task := range tasksToRun {
		s.executeTask(s.ctx, task)
		s.updateTaskNextRun(task)
	}
}

func (s *SchedulerService) executeTask(ctx context.Context, task *Task) error {
	startTime := time.Now()
	task.LastRun = startTime
	task.Status = TaskStatusRunning

	err := task.Handler(ctx)

	task.LastDuration = time.Since(startTime)
	task.RunCount++

	if err != nil {
		task.LastError = err
		task.ErrorCount++
		task.Status = TaskStatusFailed

		// Handle retries
		if task.RetryCount < task.MaxRetries {
			task.RetryCount++
			delay := task.RetryDelay
			if task.RetryBackoff {
				// Exponential backoff: 5m, 10m, 20m, etc.
				delay = task.RetryDelay * time.Duration(1<<(task.RetryCount-1))
			}
			task.NextRun = time.Now().Add(delay)
		}
	} else {
		task.LastError = nil
		task.Status = TaskStatusSuccess
		// Reset retry count on success
		task.RetryCount = 0
	}

	return err
}

func (s *SchedulerService) updateNextRunTimes() {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	for _, task := range s.tasks {
		s.calculateNextRun(task, now)
	}
}

func (s *SchedulerService) updateTaskNextRun(task *Task) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.calculateNextRun(task, time.Now())
}

func (s *SchedulerService) calculateNextRun(task *Task, from time.Time) {
	// Check if this is an interval schedule
	_, interval, err := parseScheduleShorthand(task.Schedule)
	if err != nil {
		return
	}

	// Interval-based schedule (@every Xm)
	if interval > 0 {
		if task.LastRun.IsZero() {
			// First run - schedule immediately or from now
			task.NextRun = from.Add(interval)
		} else {
			task.NextRun = task.LastRun.Add(interval)
			// If next run is in the past, catch up
			for task.NextRun.Before(from) {
				task.NextRun = task.NextRun.Add(interval)
			}
		}
		return
	}

	// Cron-based schedule
	schedule, err := parseCronSchedule(task.Schedule)
	if err != nil || schedule == nil {
		return
	}

	// Start from next minute
	next := from.Truncate(time.Minute).Add(time.Minute)

	// Find next matching time (look up to 1 year ahead)
	// minutes in a year
	maxIterations := 525600
	for i := 0; i < maxIterations; i++ {
		if schedule.matches(next) {
			task.NextRun = next
			return
		}
		next = next.Add(time.Minute)
	}
}

// cronSchedule represents a parsed cron schedule
type cronSchedule struct {
	// 0-59
	minutes []int
	// 0-23
	hours []int
	// 1-31
	days []int
	// 1-12
	months []int
	// 0-6 (0 = Sunday)
	weekdays []int
}

// matches returns true if the given time matches the schedule
func (cs *cronSchedule) matches(t time.Time) bool {
	return contains(cs.minutes, t.Minute()) &&
		contains(cs.hours, t.Hour()) &&
		contains(cs.days, t.Day()) &&
		contains(cs.months, int(t.Month())) &&
		contains(cs.weekdays, int(t.Weekday()))
}

// parseScheduleShorthand converts @shorthand to cron format
// Supports: @hourly, @daily, @weekly, @monthly, @every Xm, @every Xh, @every Xs
func parseScheduleShorthand(schedule string) (string, time.Duration, error) {
	schedule = strings.TrimSpace(schedule)

	if !strings.HasPrefix(schedule, "@") {
		return schedule, 0, nil
	}

	switch schedule {
	case "@hourly":
		return "0 * * * *", 0, nil
	case "@daily":
		return "0 0 * * *", 0, nil
	case "@weekly":
		return "0 0 * * 0", 0, nil
	case "@monthly":
		return "0 0 1 * *", 0, nil
	}

	// @every Xm, @every Xh, @every Xs
	if strings.HasPrefix(schedule, "@every ") {
		intervalStr := strings.TrimPrefix(schedule, "@every ")
		intervalStr = strings.TrimSpace(intervalStr)

		if len(intervalStr) < 2 {
			return "", 0, fmt.Errorf("invalid @every format: %s", schedule)
		}

		unit := intervalStr[len(intervalStr)-1]
		valueStr := intervalStr[:len(intervalStr)-1]
		value, err := strconv.Atoi(valueStr)
		if err != nil || value <= 0 {
			return "", 0, fmt.Errorf("invalid @every value: %s", intervalStr)
		}

		var interval time.Duration
		switch unit {
		case 's':
			interval = time.Duration(value) * time.Second
		case 'm':
			interval = time.Duration(value) * time.Minute
		case 'h':
			interval = time.Duration(value) * time.Hour
		default:
			return "", 0, fmt.Errorf("invalid @every unit: %c (use s, m, or h)", unit)
		}

		return "", interval, nil
	}

	return "", 0, fmt.Errorf("unknown schedule shorthand: %s", schedule)
}

// parseCronSchedule parses a cron schedule string
// Format: minute hour day month weekday
// Also supports @shorthand format via parseScheduleShorthand
func parseCronSchedule(schedule string) (*cronSchedule, error) {
	// Handle @shorthand first
	cronExpr, _, err := parseScheduleShorthand(schedule)
	if err != nil {
		return nil, err
	}
	if cronExpr == "" {
		// This is an interval schedule, not cron
		return nil, nil
	}

	parts := strings.Fields(cronExpr)
	if len(parts) != 5 {
		return nil, fmt.Errorf("invalid cron format: expected 5 fields, got %d", len(parts))
	}

	cs := &cronSchedule{}
	var parseErr error

	cs.minutes, parseErr = parseField(parts[0], 0, 59)
	if parseErr != nil {
		return nil, fmt.Errorf("invalid minute field: %w", parseErr)
	}

	cs.hours, parseErr = parseField(parts[1], 0, 23)
	if parseErr != nil {
		return nil, fmt.Errorf("invalid hour field: %w", parseErr)
	}

	cs.days, parseErr = parseField(parts[2], 1, 31)
	if parseErr != nil {
		return nil, fmt.Errorf("invalid day field: %w", parseErr)
	}

	cs.months, parseErr = parseField(parts[3], 1, 12)
	if parseErr != nil {
		return nil, fmt.Errorf("invalid month field: %w", parseErr)
	}

	cs.weekdays, parseErr = parseField(parts[4], 0, 6)
	if parseErr != nil {
		return nil, fmt.Errorf("invalid weekday field: %w", parseErr)
	}

	return cs, nil
}

// parseField parses a cron field
func parseField(field string, min, max int) ([]int, error) {
	if field == "*" {
		return makeRange(min, max), nil
	}

	var values []int

	// Handle comma-separated values
	for _, part := range strings.Split(field, ",") {
		// Handle step values (*/n or n-m/s)
		if strings.Contains(part, "/") {
			stepParts := strings.Split(part, "/")
			if len(stepParts) != 2 {
				return nil, fmt.Errorf("invalid step format: %s", part)
			}

			step, err := strconv.Atoi(stepParts[1])
			if err != nil || step <= 0 {
				return nil, fmt.Errorf("invalid step value: %s", stepParts[1])
			}

			var rangeValues []int
			if stepParts[0] == "*" {
				rangeValues = makeRange(min, max)
			} else if strings.Contains(stepParts[0], "-") {
				rangeValues, err = parseRange(stepParts[0], min, max)
				if err != nil {
					return nil, err
				}
			} else {
				return nil, fmt.Errorf("invalid step base: %s", stepParts[0])
			}

			for i, v := range rangeValues {
				if i%step == 0 {
					values = append(values, v)
				}
			}
		} else if strings.Contains(part, "-") {
			// Handle ranges (n-m)
			rangeValues, err := parseRange(part, min, max)
			if err != nil {
				return nil, err
			}
			values = append(values, rangeValues...)
		} else {
			// Single value
			v, err := strconv.Atoi(part)
			if err != nil {
				return nil, fmt.Errorf("invalid value: %s", part)
			}
			if v < min || v > max {
				return nil, fmt.Errorf("value out of range: %d", v)
			}
			values = append(values, v)
		}
	}

	return values, nil
}

// parseRange parses a range like "1-5"
func parseRange(s string, min, max int) ([]int, error) {
	parts := strings.Split(s, "-")
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid range: %s", s)
	}

	start, err := strconv.Atoi(parts[0])
	if err != nil {
		return nil, fmt.Errorf("invalid range start: %s", parts[0])
	}

	end, err := strconv.Atoi(parts[1])
	if err != nil {
		return nil, fmt.Errorf("invalid range end: %s", parts[1])
	}

	if start < min || end > max || start > end {
		return nil, fmt.Errorf("invalid range: %d-%d", start, end)
	}

	return makeRange(start, end), nil
}

// makeRange creates a slice of integers from start to end (inclusive)
func makeRange(start, end int) []int {
	result := make([]int, end-start+1)
	for i := range result {
		result[i] = start + i
	}
	return result
}

// contains checks if a slice contains a value
func contains(slice []int, val int) bool {
	for _, v := range slice {
		if v == val {
			return true
		}
	}
	return false
}
