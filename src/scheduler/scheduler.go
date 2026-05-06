// Package scheduler provides background task scheduling
// See AI.md for scheduler specification
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
	Schedule string // Cron expression: "minute hour day month weekday"
	Handler  func() error
	Enabled  bool
	LastRun  time.Time
	NextRun  time.Time
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
	task.NextRun = calculateNextRun(schedule, time.Now())
	s.tasks[name] = task
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

func (s *Scheduler) run() {
	ticker := time.NewTicker(time.Minute)
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
		if shouldRunTask(task.Schedule, now) {
			go s.runTask(task)
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
	task.NextRun = calculateNextRun(task.Schedule, task.LastRun)
	s.mu.Unlock()
}

// shouldRunTask checks if task should run based on cron expression
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

// matchCronField checks if value matches cron field expression
func matchCronField(field string, value, min, max int) bool {
	// Any value
	if field == "*" {
		return true
	}

	// Step values: */n
	if strings.HasPrefix(field, "*/") {
		step, err := strconv.Atoi(field[2:])
		if err != nil || step <= 0 {
			return false
		}
		return value%step == 0
	}

	// List values: n,m,o
	if strings.Contains(field, ",") {
		for _, part := range strings.Split(field, ",") {
			if matchCronField(part, value, min, max) {
				return true
			}
		}
		return false
	}

	// Range values: n-m
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

	// Exact value
	n, err := strconv.Atoi(field)
	if err != nil {
		return false
	}
	return value == n
}

// calculateNextRun calculates the next run time for a schedule
func calculateNextRun(schedule string, from time.Time) time.Time {
	// Start from next minute
	next := from.Truncate(time.Minute).Add(time.Minute)

	// Search up to 1 year ahead
	maxIterations := 365 * 24 * 60
	for i := 0; i < maxIterations; i++ {
		if shouldRunTask(schedule, next) {
			return next
		}
		next = next.Add(time.Minute)
	}

	// Fallback to 1 hour from now
	return from.Add(time.Hour)
}
