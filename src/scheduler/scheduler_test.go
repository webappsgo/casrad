// Package scheduler — Tests for the built-in task scheduler.
// Covers: parseInterval, shouldRunTask, matchCronField, calculateNextRun,
// Register/RegisterDisabled, Start/Stop, and RegisterDefaultTasks.
package scheduler

import (
	"sync/atomic"
	"testing"
	"time"
)

// --- parseInterval ---

func TestParseInterval(t *testing.T) {
	t.Parallel()

	cases := []struct {
		schedule string
		wantDur  time.Duration
		wantOK   bool
	}{
		{"@hourly", time.Hour, true},
		{"@daily", 24 * time.Hour, true},
		{"@midnight", 24 * time.Hour, true},
		{"@weekly", 7 * 24 * time.Hour, true},
		{"@every 5m", 5 * time.Minute, true},
		{"@every 30s", 30 * time.Second, true},
		{"@every 1h30m", 90 * time.Minute, true},
		{"@every 0s", 0, false},
		{"@every -1m", 0, false},
		{"@every notaduration", 0, false},
		// Standard cron expressions should not match
		{"0 * * * *", 0, false},
		{"*/5 * * * *", 0, false},
		{"", 0, false},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.schedule, func(t *testing.T) {
			t.Parallel()
			got, ok := parseInterval(tc.schedule)
			if ok != tc.wantOK {
				t.Errorf("parseInterval(%q) ok=%v, want %v", tc.schedule, ok, tc.wantOK)
			}
			if tc.wantOK && got != tc.wantDur {
				t.Errorf("parseInterval(%q) dur=%v, want %v", tc.schedule, got, tc.wantDur)
			}
		})
	}
}

// --- matchCronField ---

func TestMatchCronFieldWildcard(t *testing.T) {
	t.Parallel()
	for _, v := range []int{0, 30, 59} {
		if !matchCronField("*", v, 0, 59) {
			t.Errorf("matchCronField(*,%d) should be true", v)
		}
	}
}

func TestMatchCronFieldExact(t *testing.T) {
	t.Parallel()
	if !matchCronField("5", 5, 0, 59) {
		t.Error("matchCronField(5,5) should be true")
	}
	if matchCronField("5", 6, 0, 59) {
		t.Error("matchCronField(5,6) should be false")
	}
}

func TestMatchCronFieldStep(t *testing.T) {
	t.Parallel()
	if !matchCronField("*/5", 0, 0, 59) {
		t.Error("*/5 should match 0")
	}
	if !matchCronField("*/5", 15, 0, 59) {
		t.Error("*/5 should match 15")
	}
	if !matchCronField("*/5", 30, 0, 59) {
		t.Error("*/5 should match 30")
	}
	if matchCronField("*/5", 7, 0, 59) {
		t.Error("*/5 should not match 7")
	}
}

func TestMatchCronFieldList(t *testing.T) {
	t.Parallel()
	if !matchCronField("1,3,5", 3, 0, 59) {
		t.Error("1,3,5 should match 3")
	}
	if matchCronField("1,3,5", 2, 0, 59) {
		t.Error("1,3,5 should not match 2")
	}
}

func TestMatchCronFieldRange(t *testing.T) {
	t.Parallel()
	if !matchCronField("8-17", 12, 0, 23) {
		t.Error("8-17 should match 12")
	}
	if !matchCronField("8-17", 8, 0, 23) {
		t.Error("8-17 should match 8 (inclusive)")
	}
	if !matchCronField("8-17", 17, 0, 23) {
		t.Error("8-17 should match 17 (inclusive)")
	}
	if matchCronField("8-17", 7, 0, 23) {
		t.Error("8-17 should not match 7")
	}
	if matchCronField("8-17", 18, 0, 23) {
		t.Error("8-17 should not match 18")
	}
}

func TestMatchCronFieldInvalidStep(t *testing.T) {
	t.Parallel()
	if matchCronField("*/0", 0, 0, 59) {
		t.Error("*/0 (divide by zero) should return false")
	}
	if matchCronField("*/abc", 0, 0, 59) {
		t.Error("*/abc should return false")
	}
}

func TestMatchCronFieldInvalidRange(t *testing.T) {
	t.Parallel()
	if matchCronField("a-b", 5, 0, 59) {
		t.Error("a-b should return false")
	}
}

func TestMatchCronFieldInvalidExact(t *testing.T) {
	t.Parallel()
	if matchCronField("abc", 1, 0, 59) {
		t.Error("non-numeric field should return false")
	}
}

// --- shouldRunTask ---

func TestShouldRunTaskWildcard(t *testing.T) {
	t.Parallel()
	now := time.Date(2025, 1, 15, 12, 30, 0, 0, time.UTC)
	if !shouldRunTask("* * * * *", now) {
		t.Error("* * * * * should always run")
	}
}

func TestShouldRunTaskSpecific(t *testing.T) {
	t.Parallel()
	// "0 2 * * *" means minute=0, hour=2
	match := time.Date(2025, 1, 15, 2, 0, 0, 0, time.UTC)
	noMatch := time.Date(2025, 1, 15, 2, 1, 0, 0, time.UTC)

	if !shouldRunTask("0 2 * * *", match) {
		t.Error("0 2 * * * should match 02:00")
	}
	if shouldRunTask("0 2 * * *", noMatch) {
		t.Error("0 2 * * * should not match 02:01")
	}
}

func TestShouldRunTaskWeekday(t *testing.T) {
	t.Parallel()
	// Sunday = 0
	sunday := time.Date(2025, 1, 5, 3, 0, 0, 0, time.UTC)
	monday := time.Date(2025, 1, 6, 3, 0, 0, 0, time.UTC)

	if !shouldRunTask("0 3 * * 0", sunday) {
		t.Error("0 3 * * 0 should match Sunday 03:00")
	}
	if shouldRunTask("0 3 * * 0", monday) {
		t.Error("0 3 * * 0 should not match Monday")
	}
}

func TestShouldRunTaskEveryFiveMinutes(t *testing.T) {
	t.Parallel()
	match := time.Date(2025, 1, 15, 12, 15, 0, 0, time.UTC)
	noMatch := time.Date(2025, 1, 15, 12, 7, 0, 0, time.UTC)

	if !shouldRunTask("*/5 * * * *", match) {
		t.Error("*/5 * * * * should match minute=15")
	}
	if shouldRunTask("*/5 * * * *", noMatch) {
		t.Error("*/5 * * * * should not match minute=7")
	}
}

func TestShouldRunTaskInvalidFormat(t *testing.T) {
	t.Parallel()
	now := time.Now()
	if shouldRunTask("not a cron", now) {
		t.Error("invalid cron format should return false")
	}
	if shouldRunTask("* * * *", now) {
		t.Error("4-field cron should return false (need 5)")
	}
	if shouldRunTask("", now) {
		t.Error("empty cron should return false")
	}
}

// --- calculateNextRun ---

func TestCalculateNextRunBasic(t *testing.T) {
	t.Parallel()
	// Every minute: next run should be within 1 minute from now
	from := time.Date(2025, 1, 15, 12, 0, 0, 0, time.UTC)
	next := calculateNextRun("* * * * *", from)
	if next.Before(from) {
		t.Errorf("calculateNextRun returned time before 'from': %v < %v", next, from)
	}
	if next.After(from.Add(2 * time.Minute)) {
		t.Errorf("calculateNextRun for * * * * * too far ahead: %v", next)
	}
}

func TestCalculateNextRunHourly(t *testing.T) {
	t.Parallel()
	from := time.Date(2025, 1, 15, 12, 30, 0, 0, time.UTC)
	next := calculateNextRun("0 * * * *", from)
	// Should land on :00 of some hour
	if next.Minute() != 0 {
		t.Errorf("calculateNextRun for 0 * * * * should land on :00, got minute=%d", next.Minute())
	}
	if next.Before(from) {
		t.Error("next run should be after 'from'")
	}
}

// --- Register / Task lifecycle ---

func TestRegisterAndGetTask(t *testing.T) {
	t.Parallel()
	s := New()
	ran := false
	s.Register("test_task", "@every 1h", func() error {
		ran = true
		return nil
	})

	s.mu.RLock()
	task, ok := s.tasks["test_task"]
	s.mu.RUnlock()

	if !ok {
		t.Fatal("task not found after Register")
	}
	if !task.Enabled {
		t.Error("registered task should be enabled by default")
	}
	if task.interval != time.Hour {
		t.Errorf("interval = %v, want 1h", task.interval)
	}
	_ = ran
}

func TestRegisterDisabledTask(t *testing.T) {
	t.Parallel()
	s := New()
	s.RegisterDisabled("disabled_task", "@every 1h", func() error { return nil })

	s.mu.RLock()
	task, ok := s.tasks["disabled_task"]
	s.mu.RUnlock()

	if !ok {
		t.Fatal("task not found after RegisterDisabled")
	}
	if task.Enabled {
		t.Error("RegisterDisabled task should start disabled")
	}
}

func TestRegisterOverwritesPreviousTask(t *testing.T) {
	t.Parallel()
	s := New()
	s.Register("dup", "@every 1h", func() error { return nil })
	s.Register("dup", "@every 2h", func() error { return nil })

	s.mu.RLock()
	task := s.tasks["dup"]
	s.mu.RUnlock()

	if task.interval != 2*time.Hour {
		t.Errorf("second Register should overwrite first, got interval=%v", task.interval)
	}
}

func TestStartStop(t *testing.T) {
	t.Parallel()
	s := New()
	s.Register("noop", "@every 10h", func() error { return nil })
	s.Start()

	s.mu.RLock()
	running := s.running
	s.mu.RUnlock()

	if !running {
		t.Error("scheduler should be running after Start()")
	}

	s.Stop()
	time.Sleep(50 * time.Millisecond)

	s.mu.RLock()
	running = s.running
	s.mu.RUnlock()

	if running {
		t.Error("scheduler should not be running after Stop()")
	}
}

func TestStartIdempotent(t *testing.T) {
	t.Parallel()
	s := New()
	// Double-start should not panic or block
	s.Start()
	s.Start()
	s.Stop()
}

func TestStopIdempotent(t *testing.T) {
	t.Parallel()
	s := New()
	// Stop on non-running scheduler should not panic
	s.Stop()
}

// --- RegisterDefaultTasks ---

func TestRegisterDefaultTasksRegistersAll(t *testing.T) {
	t.Parallel()
	s := New()
	RegisterDefaultTasks(s)

	required := []string{
		"ssl_renewal", "geoip_update", "blocklist_update", "cve_update",
		"session_cleanup", "token_cleanup", "log_rotation", "backup_daily",
		"backup_hourly", "healthcheck_self", "tor_health", "cluster_heartbeat",
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, name := range required {
		if _, ok := s.tasks[name]; !ok {
			t.Errorf("task %q not registered", name)
		}
	}
}

func TestRegisterDefaultTasksBackupHourlyDisabled(t *testing.T) {
	t.Parallel()
	s := New()
	RegisterDefaultTasks(s)

	s.mu.RLock()
	task, ok := s.tasks["backup_hourly"]
	s.mu.RUnlock()

	if !ok {
		t.Fatal("backup_hourly not registered")
	}
	if task.Enabled {
		t.Error("backup_hourly should be disabled by default per spec")
	}
}

func TestRegisterDefaultTasksHandlersRunWithoutPanic(t *testing.T) {
	t.Parallel()
	s := New()
	RegisterDefaultTasks(s)

	s.mu.RLock()
	tasks := make([]*Task, 0, len(s.tasks))
	for _, task := range s.tasks {
		tasks = append(tasks, task)
	}
	s.mu.RUnlock()

	for _, task := range tasks {
		task := task
		t.Run(task.Name, func(t *testing.T) {
			t.Parallel()
			if err := task.Handler(); err != nil {
				t.Errorf("task %q handler returned error: %v", task.Name, err)
			}
		})
	}
}

// --- Interval-based scheduling ---

func TestIntervalTaskRunsWhenElapsed(t *testing.T) {
	t.Parallel()

	s := New()
	var runCount int32

	// Register with very short interval
	s.Register("fast_task", "@every 50ms", func() error {
		atomic.AddInt32(&runCount, 1)
		return nil
	})

	// Manually set LastRun to zero so it fires immediately on first tick
	s.mu.Lock()
	s.tasks["fast_task"].LastRun = time.Time{}
	s.mu.Unlock()

	// Run tick directly (bypasses the 10-second ticker for testing)
	s.tick()

	time.Sleep(100 * time.Millisecond)

	// Task should have been dispatched at least once
	if atomic.LoadInt32(&runCount) == 0 {
		// Give goroutine time to complete
		time.Sleep(200 * time.Millisecond)
	}

	if atomic.LoadInt32(&runCount) == 0 {
		t.Error("interval task should have run at least once after tick with zero LastRun")
	}
}

func TestCronTaskNextRunIsInFuture(t *testing.T) {
	t.Parallel()
	s := New()
	s.Register("cron_task", "0 3 * * *", func() error { return nil })

	s.mu.RLock()
	task := s.tasks["cron_task"]
	s.mu.RUnlock()

	if task.NextRun.IsZero() {
		t.Error("cron task NextRun should not be zero")
	}
	if !task.NextRun.After(time.Now()) {
		t.Errorf("cron task NextRun=%v should be in the future", task.NextRun)
	}
}
