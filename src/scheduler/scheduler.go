// Package scheduler provides background task scheduling
// See AI.md PART 19 for scheduler specification
package scheduler

import (
	"log"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Task represents a scheduled task
type Task struct {
	Name     string
	Schedule string // Cron expression or @every / @hourly shorthand
	Handler  func() error
	Enabled  bool
	LastRun  time.Time
	NextRun  time.Time

	// interval is non-zero when Schedule starts with "@every" or "@hourly"
	interval time.Duration
}

// Scheduler manages scheduled tasks
type Scheduler struct {
	tasks   map[string]*Task
	running bool
	stop    chan struct{}
	mu      sync.RWMutex
}

// New creates a new scheduler
func New() *Scheduler {
	return &Scheduler{
		tasks: make(map[string]*Task),
		stop:  make(chan struct{}),
	}
}

// Register registers a new task
func (s *Scheduler) Register(name, schedule string, handler func() error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	task := &Task{
		Name:     name,
		Schedule: schedule,
		Handler:  handler,
		Enabled:  true,
	}

	iv, ok := parseInterval(schedule)
	if ok {
		task.interval = iv
		task.NextRun = time.Now().Add(iv)
	} else {
		task.NextRun = calculateNextRun(schedule, time.Now())
	}

	s.tasks[name] = task
}

// RegisterDisabled registers a task that starts in disabled state
func (s *Scheduler) RegisterDisabled(name, schedule string, handler func() error) {
	s.Register(name, schedule, handler)
	s.mu.Lock()
	defer s.mu.Unlock()
	if t, ok := s.tasks[name]; ok {
		t.Enabled = false
	}
}

// Start starts the scheduler
func (s *Scheduler) Start() {
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		return
	}
	s.running = true
	s.mu.Unlock()

	go s.run()
}

// Stop stops the scheduler
func (s *Scheduler) Stop() {
	s.mu.Lock()
	if !s.running {
		s.mu.Unlock()
		return
	}
	s.running = false
	s.mu.Unlock()

	close(s.stop)
}

// run is the main scheduler loop; ticks every 10 seconds for sub-minute tasks
func (s *Scheduler) run() {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-s.stop:
			return
		case <-ticker.C:
			s.tick()
		}
	}
}

func (s *Scheduler) tick() {
	s.mu.RLock()
	defer s.mu.RUnlock()

	now := time.Now()
	for _, task := range s.tasks {
		if !task.Enabled {
			continue
		}
		if task.interval > 0 {
			// @every / @hourly: run when interval has elapsed since last run
			if task.LastRun.IsZero() || now.Sub(task.LastRun) >= task.interval {
				go s.runTask(task)
			}
		} else {
			// Standard cron: check once per minute on :00 second boundary
			if now.Second() < 10 && shouldRunTask(task.Schedule, now) {
				go s.runTask(task)
			}
		}
	}
}

func (s *Scheduler) runTask(task *Task) {
	log.Printf("scheduler: running task %s", task.Name)
	if err := task.Handler(); err != nil {
		log.Printf("scheduler: task %s failed: %v", task.Name, err)
	} else {
		log.Printf("scheduler: task %s completed", task.Name)
	}
	s.mu.Lock()
	task.LastRun = time.Now()
	if task.interval > 0 {
		task.NextRun = task.LastRun.Add(task.interval)
	} else {
		task.NextRun = calculateNextRun(task.Schedule, task.LastRun)
	}
	s.mu.Unlock()
}

// parseInterval parses @every <duration> and @hourly shorthands.
// Returns (duration, true) on success.
func parseInterval(schedule string) (time.Duration, bool) {
	s := strings.TrimSpace(schedule)
	switch s {
	case "@hourly":
		return time.Hour, true
	case "@daily", "@midnight":
		return 24 * time.Hour, true
	case "@weekly":
		return 7 * 24 * time.Hour, true
	}

	if strings.HasPrefix(s, "@every ") {
		raw := strings.TrimPrefix(s, "@every ")
		d, err := time.ParseDuration(strings.TrimSpace(raw))
		if err == nil && d > 0 {
			return d, true
		}
	}

	return 0, false
}

// shouldRunTask checks if a standard 5-field cron expression fires at now.
// Format: "minute hour day month weekday"
// Supports: * (any), */n (every n), n (exact), n,m (list), n-m (range)
func shouldRunTask(schedule string, now time.Time) bool {
	parts := strings.Fields(schedule)
	if len(parts) != 5 {
		return false
	}

	minute := now.Minute()
	hour := now.Hour()
	day := now.Day()
	month := int(now.Month())
	weekday := int(now.Weekday())

	return matchCronField(parts[0], minute, 0, 59) &&
		matchCronField(parts[1], hour, 0, 23) &&
		matchCronField(parts[2], day, 1, 31) &&
		matchCronField(parts[3], month, 1, 12) &&
		matchCronField(parts[4], weekday, 0, 6)
}

// matchCronField checks if value matches a single cron field expression
func matchCronField(field string, value, min, max int) bool {
	if field == "*" {
		return true
	}

	if strings.HasPrefix(field, "*/") {
		step, err := strconv.Atoi(field[2:])
		if err != nil || step <= 0 {
			return false
		}
		return value%step == 0
	}

	if strings.Contains(field, ",") {
		for _, part := range strings.Split(field, ",") {
			if matchCronField(part, value, min, max) {
				return true
			}
		}
		return false
	}

	if strings.Contains(field, "-") {
		rangeParts := strings.SplitN(field, "-", 2)
		if len(rangeParts) == 2 {
			start, err1 := strconv.Atoi(rangeParts[0])
			end, err2 := strconv.Atoi(rangeParts[1])
			if err1 == nil && err2 == nil {
				return value >= start && value <= end
			}
		}
		return false
	}

	n, err := strconv.Atoi(field)
	if err != nil {
		return false
	}
	return value == n
}

// calculateNextRun calculates the next scheduled run time for standard cron expressions
func calculateNextRun(schedule string, from time.Time) time.Time {
	next := from.Truncate(time.Minute).Add(time.Minute)

	maxIterations := 365 * 24 * 60
	for i := 0; i < maxIterations; i++ {
		if shouldRunTask(schedule, next) {
			return next
		}
		next = next.Add(time.Minute)
	}

	return from.Add(time.Hour)
}

// RegisterDefaultTasks registers all 12 required tasks per AI.md PART 19.
// Each handler is a stub that will be wired to real implementations as
// subsystems are built.
func RegisterDefaultTasks(s *Scheduler) {
	// ssl_renewal: daily at 03:00 — check and renew expiring SSL certificates
	s.Register("ssl_renewal", "0 3 * * *", func() error {
		log.Printf("scheduler: ssl_renewal: stub — SSL subsystem not yet implemented")
		return nil
	})

	// geoip_update: weekly Sunday at 03:00 — download ip-location-db updates
	s.Register("geoip_update", "0 3 * * 0", func() error {
		log.Printf("scheduler: geoip_update: stub — GeoIP subsystem not yet implemented")
		return nil
	})

	// blocklist_update: daily at 04:00 — refresh IP/UA/referrer blocklists
	s.Register("blocklist_update", "0 4 * * *", func() error {
		log.Printf("scheduler: blocklist_update: stub — blocklist subsystem not yet implemented")
		return nil
	})

	// cve_update: daily at 05:00 — update CVE / Trivy databases
	s.Register("cve_update", "0 5 * * *", func() error {
		log.Printf("scheduler: cve_update: stub — CVE subsystem not yet implemented")
		return nil
	})

	// session_cleanup: every 15 minutes — remove expired sessions from DB
	s.Register("session_cleanup", "@every 15m", func() error {
		log.Printf("scheduler: session_cleanup: stub — session store not yet implemented")
		return nil
	})

	// token_cleanup: every 15 minutes — remove expired API tokens from DB
	s.Register("token_cleanup", "@every 15m", func() error {
		log.Printf("scheduler: token_cleanup: stub — token store not yet implemented")
		return nil
	})

	// log_rotation: daily at midnight — rotate and compress log files
	s.Register("log_rotation", "0 0 * * *", func() error {
		log.Printf("scheduler: log_rotation: stub — log rotation not yet implemented")
		return nil
	})

	// backup_daily: daily at 02:00 — create automatic database backup
	s.Register("backup_daily", "0 2 * * *", func() error {
		log.Printf("scheduler: backup_daily: stub — backup subsystem not yet implemented")
		return nil
	})

	// backup_hourly: hourly — disabled by default per spec
	s.RegisterDisabled("backup_hourly", "@hourly", func() error {
		log.Printf("scheduler: backup_hourly: stub — backup subsystem not yet implemented")
		return nil
	})

	// healthcheck_self: every 5 minutes — verify own endpoints are responding
	s.Register("healthcheck_self", "@every 5m", func() error {
		log.Printf("scheduler: healthcheck_self: stub — self-check not yet implemented")
		return nil
	})

	// tor_health: every 10 minutes — verify Tor hidden service is reachable
	s.Register("tor_health", "@every 10m", func() error {
		log.Printf("scheduler: tor_health: stub — Tor subsystem not yet implemented")
		return nil
	})

	// cluster_heartbeat: every 30 seconds — broadcast presence to cluster peers
	s.Register("cluster_heartbeat", "@every 30s", func() error {
		log.Printf("scheduler: cluster_heartbeat: stub — cluster subsystem not yet implemented")
		return nil
	})
}
