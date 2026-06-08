// Package service — Tests for SchedulerService and cron parsing functions.
// Covers: NewSchedulerService (built-in tasks registered), AddTask (valid schedule,
// invalid schedule returns error), EnableTask/DisableTask, GetTask (found/not-found),
// ListTasks (non-empty), IsRunning (before/after Start/Stop), RegisterHandler,
// parseCronSchedule (all fields, wildcard, step, range, comma), parseScheduleShorthand
// (@hourly, @daily, @weekly, @monthly, @every, unknown), parseField (wildcard, step,
// range, single value, out-of-range), makeRange, contains.
package service

import (
	"context"
	"errors"
	"testing"
	"time"
)

// --- NewSchedulerService ---

func TestNewSchedulerServiceNotNil(t *testing.T) {
	t.Parallel()
	s := NewSchedulerService()
	if s == nil {
		t.Fatal("NewSchedulerService returned nil")
	}
}

func TestNewSchedulerServiceHasBuiltInTasks(t *testing.T) {
	t.Parallel()
	s := NewSchedulerService()
	tasks := s.ListTasks()
	if len(tasks) == 0 {
		t.Error("NewSchedulerService should register built-in tasks")
	}
	// Verify at least one known built-in task
	found := false
	for _, task := range tasks {
		if task.ID == "ssl_renewal" {
			found = true
			break
		}
	}
	if !found {
		t.Error("built-in task ssl_renewal should be registered")
	}
}

// --- AddTask ---

func TestAddTaskValidSchedule(t *testing.T) {
	t.Parallel()
	s := NewSchedulerService()
	err := s.AddTask("my_task", "0 * * * *", func(ctx context.Context) error {
		return nil
	})
	if err != nil {
		t.Errorf("AddTask valid schedule: %v", err)
	}
	if s.GetTask("my_task") == nil {
		t.Error("AddTask should store the task")
	}
}

func TestAddTaskInvalidScheduleReturnsError(t *testing.T) {
	t.Parallel()
	s := NewSchedulerService()
	err := s.AddTask("bad_task", "not a cron", nil)
	if err == nil {
		t.Error("AddTask with invalid schedule should return error")
	}
}

func TestAddTaskIntervalSchedule(t *testing.T) {
	t.Parallel()
	s := NewSchedulerService()
	// @every intervals return nil from parseCronSchedule (not an error)
	err := s.AddTask("interval_task", "@every 5m", func(ctx context.Context) error {
		return nil
	})
	if err != nil {
		t.Errorf("AddTask @every schedule: %v", err)
	}
}

// --- EnableTask / DisableTask ---

func TestEnableDisableTask(t *testing.T) {
	t.Parallel()
	s := NewSchedulerService()
	s.AddTask("toggle_task", "0 0 * * *", func(ctx context.Context) error { return nil })

	s.DisableTask("toggle_task")
	task := s.GetTask("toggle_task")
	if task == nil {
		t.Fatal("GetTask returned nil")
	}
	if task.Enabled {
		t.Error("task should be disabled after DisableTask")
	}

	s.EnableTask("toggle_task")
	task = s.GetTask("toggle_task")
	if !task.Enabled {
		t.Error("task should be enabled after EnableTask")
	}
}

func TestDisableTaskNotFound(t *testing.T) {
	t.Parallel()
	s := NewSchedulerService()
	// Should not panic when disabling a non-existent task
	s.DisableTask("nonexistent")
}

func TestEnableTaskNotFound(t *testing.T) {
	t.Parallel()
	s := NewSchedulerService()
	s.EnableTask("nonexistent")
}

// --- GetTask ---

func TestGetTaskFound(t *testing.T) {
	t.Parallel()
	s := NewSchedulerService()
	s.AddTask("find_me", "0 3 * * *", nil)
	task := s.GetTask("find_me")
	if task == nil {
		t.Error("GetTask should return task by name")
	}
	if task.Name != "find_me" {
		t.Errorf("task.Name = %q, want find_me", task.Name)
	}
}

func TestGetTaskNotFound(t *testing.T) {
	t.Parallel()
	s := NewSchedulerService()
	task := s.GetTask("no_such_task")
	if task != nil {
		t.Error("GetTask should return nil for unknown task")
	}
}

// --- RegisterHandler ---

func TestRegisterHandler(t *testing.T) {
	t.Parallel()
	s := NewSchedulerService()
	s.AddTask("handler_task", "0 0 * * *", nil)
	called := false
	s.RegisterHandler("handler_task", func(ctx context.Context) error {
		called = true
		return nil
	})
	task := s.GetTask("handler_task")
	if task == nil || task.Handler == nil {
		t.Fatal("handler not registered")
	}
	task.Handler(context.Background())
	if !called {
		t.Error("RegisterHandler should set the task handler")
	}
}

// --- IsRunning ---

func TestIsRunningInitiallyFalse(t *testing.T) {
	t.Parallel()
	s := NewSchedulerService()
	if s.IsRunning() {
		t.Error("IsRunning should be false before Start")
	}
}

func TestIsRunningAfterStart(t *testing.T) {
	t.Parallel()
	s := NewSchedulerService()
	s.Start()
	defer s.Stop()
	if !s.IsRunning() {
		t.Error("IsRunning should be true after Start")
	}
}

func TestIsRunningAfterStop(t *testing.T) {
	t.Parallel()
	s := NewSchedulerService()
	s.Start()
	s.Stop()
	if s.IsRunning() {
		t.Error("IsRunning should be false after Stop")
	}
}

// --- parseCronSchedule ---

func TestParseCronScheduleWildcard(t *testing.T) {
	t.Parallel()
	cs, err := parseCronSchedule("* * * * *")
	if err != nil {
		t.Fatalf("parseCronSchedule wildcard: %v", err)
	}
	if cs == nil {
		t.Fatal("parseCronSchedule wildcard returned nil")
	}
	if len(cs.minutes) != 60 {
		t.Errorf("minutes count = %d, want 60", len(cs.minutes))
	}
}

func TestParseCronScheduleSpecific(t *testing.T) {
	t.Parallel()
	cs, err := parseCronSchedule("30 14 1 6 0")
	if err != nil {
		t.Fatalf("parseCronSchedule specific: %v", err)
	}
	if len(cs.minutes) != 1 || cs.minutes[0] != 30 {
		t.Errorf("minutes = %v, want [30]", cs.minutes)
	}
	if len(cs.hours) != 1 || cs.hours[0] != 14 {
		t.Errorf("hours = %v, want [14]", cs.hours)
	}
}

func TestParseCronScheduleRange(t *testing.T) {
	t.Parallel()
	cs, err := parseCronSchedule("0-5 * * * *")
	if err != nil {
		t.Fatalf("parseCronSchedule range: %v", err)
	}
	if len(cs.minutes) != 6 {
		t.Errorf("minutes count = %d, want 6", len(cs.minutes))
	}
}

func TestParseCronScheduleStep(t *testing.T) {
	t.Parallel()
	cs, err := parseCronSchedule("*/15 * * * *")
	if err != nil {
		t.Fatalf("parseCronSchedule step: %v", err)
	}
	if len(cs.minutes) != 4 {
		t.Errorf("minutes count = %d, want 4 (0,15,30,45)", len(cs.minutes))
	}
}

func TestParseCronScheduleComma(t *testing.T) {
	t.Parallel()
	cs, err := parseCronSchedule("0,30 * * * *")
	if err != nil {
		t.Fatalf("parseCronSchedule comma: %v", err)
	}
	if len(cs.minutes) != 2 {
		t.Errorf("minutes count = %d, want 2", len(cs.minutes))
	}
}

func TestParseCronScheduleWrongFieldCount(t *testing.T) {
	t.Parallel()
	_, err := parseCronSchedule("0 * * *")
	if err == nil {
		t.Error("parseCronSchedule with 4 fields should return error")
	}
}

func TestParseCronScheduleInvalidMinute(t *testing.T) {
	t.Parallel()
	_, err := parseCronSchedule("70 * * * *")
	if err == nil {
		t.Error("minute 70 (out of range) should return error")
	}
}

func TestParseCronScheduleShorthandHourly(t *testing.T) {
	t.Parallel()
	cs, err := parseCronSchedule("@hourly")
	if err != nil {
		t.Fatalf("@hourly: %v", err)
	}
	if cs == nil {
		t.Fatal("@hourly should parse to a cron schedule")
	}
}

// parseCronSchedule with @every returns nil (interval schedule, not cron)
func TestParseCronScheduleEveryReturnsNil(t *testing.T) {
	t.Parallel()
	cs, err := parseCronSchedule("@every 5m")
	if err != nil {
		t.Fatalf("@every 5m error: %v", err)
	}
	if cs != nil {
		t.Error("@every should return nil cron schedule (handled as interval)")
	}
}

// --- parseScheduleShorthand ---

func TestParseScheduleShorthandNonAt(t *testing.T) {
	t.Parallel()
	expr, dur, err := parseScheduleShorthand("0 * * * *")
	if err != nil {
		t.Fatalf("non-@ schedule: %v", err)
	}
	if expr != "0 * * * *" {
		t.Errorf("expr = %q, want passthrough", expr)
	}
	if dur != 0 {
		t.Errorf("dur = %v, want 0", dur)
	}
}

func TestParseScheduleShorthandHourly(t *testing.T) {
	t.Parallel()
	expr, dur, err := parseScheduleShorthand("@hourly")
	if err != nil {
		t.Fatalf("@hourly: %v", err)
	}
	if expr != "0 * * * *" {
		t.Errorf("@hourly expr = %q, want '0 * * * *'", expr)
	}
	if dur != 0 {
		t.Errorf("@hourly dur = %v, want 0", dur)
	}
}

func TestParseScheduleShorthandDaily(t *testing.T) {
	t.Parallel()
	expr, _, err := parseScheduleShorthand("@daily")
	if err != nil {
		t.Fatalf("@daily: %v", err)
	}
	if expr != "0 0 * * *" {
		t.Errorf("@daily expr = %q, want '0 0 * * *'", expr)
	}
}

func TestParseScheduleShorthandWeekly(t *testing.T) {
	t.Parallel()
	expr, _, err := parseScheduleShorthand("@weekly")
	if err != nil {
		t.Fatalf("@weekly: %v", err)
	}
	if expr != "0 0 * * 0" {
		t.Errorf("@weekly expr = %q, want '0 0 * * 0'", expr)
	}
}

func TestParseScheduleShorthandMonthly(t *testing.T) {
	t.Parallel()
	expr, _, err := parseScheduleShorthand("@monthly")
	if err != nil {
		t.Fatalf("@monthly: %v", err)
	}
	if expr != "0 0 1 * *" {
		t.Errorf("@monthly expr = %q, want '0 0 1 * *'", expr)
	}
}

func TestParseScheduleShorthandEverySeconds(t *testing.T) {
	t.Parallel()
	_, dur, err := parseScheduleShorthand("@every 30s")
	if err != nil {
		t.Fatalf("@every 30s: %v", err)
	}
	if dur != 30*time.Second {
		t.Errorf("dur = %v, want 30s", dur)
	}
}

func TestParseScheduleShorthandEveryMinutes(t *testing.T) {
	t.Parallel()
	_, dur, err := parseScheduleShorthand("@every 5m")
	if err != nil {
		t.Fatalf("@every 5m: %v", err)
	}
	if dur != 5*time.Minute {
		t.Errorf("dur = %v, want 5m", dur)
	}
}

func TestParseScheduleShorthandEveryHours(t *testing.T) {
	t.Parallel()
	_, dur, err := parseScheduleShorthand("@every 2h")
	if err != nil {
		t.Fatalf("@every 2h: %v", err)
	}
	if dur != 2*time.Hour {
		t.Errorf("dur = %v, want 2h", dur)
	}
}

func TestParseScheduleShorthandEveryInvalidUnit(t *testing.T) {
	t.Parallel()
	_, _, err := parseScheduleShorthand("@every 5d")
	if err == nil {
		t.Error("@every with invalid unit should return error")
	}
}

func TestParseScheduleShorthandEveryZero(t *testing.T) {
	t.Parallel()
	_, _, err := parseScheduleShorthand("@every 0m")
	if err == nil {
		t.Error("@every 0m should return error (value must be > 0)")
	}
}

func TestParseScheduleShorthandUnknown(t *testing.T) {
	t.Parallel()
	_, _, err := parseScheduleShorthand("@reboot")
	if err == nil {
		t.Error("unknown @shorthand should return error")
	}
}

// --- parseField ---

func TestParseFieldWildcard(t *testing.T) {
	t.Parallel()
	vals, err := parseField("*", 0, 59)
	if err != nil {
		t.Fatalf("parseField wildcard: %v", err)
	}
	if len(vals) != 60 {
		t.Errorf("wildcard minute count = %d, want 60", len(vals))
	}
}

func TestParseFieldSingleValue(t *testing.T) {
	t.Parallel()
	vals, err := parseField("30", 0, 59)
	if err != nil {
		t.Fatalf("parseField single: %v", err)
	}
	if len(vals) != 1 || vals[0] != 30 {
		t.Errorf("vals = %v, want [30]", vals)
	}
}

func TestParseFieldOutOfRange(t *testing.T) {
	t.Parallel()
	_, err := parseField("60", 0, 59)
	if err == nil {
		t.Error("value 60 for minutes (max 59) should return error")
	}
}

func TestParseFieldRange(t *testing.T) {
	t.Parallel()
	vals, err := parseField("1-5", 0, 59)
	if err != nil {
		t.Fatalf("parseField range: %v", err)
	}
	if len(vals) != 5 {
		t.Errorf("range 1-5 count = %d, want 5", len(vals))
	}
}

func TestParseFieldStep(t *testing.T) {
	t.Parallel()
	vals, err := parseField("*/10", 0, 59)
	if err != nil {
		t.Fatalf("parseField step: %v", err)
	}
	// 0, 10, 20, 30, 40, 50
	if len(vals) != 6 {
		t.Errorf("*/10 count = %d, want 6", len(vals))
	}
}

func TestParseFieldComma(t *testing.T) {
	t.Parallel()
	vals, err := parseField("0,15,30,45", 0, 59)
	if err != nil {
		t.Fatalf("parseField comma: %v", err)
	}
	if len(vals) != 4 {
		t.Errorf("comma count = %d, want 4", len(vals))
	}
}

func TestParseFieldInvalidStep(t *testing.T) {
	t.Parallel()
	_, err := parseField("*/0", 0, 59)
	if err == nil {
		t.Error("step 0 should return error")
	}
}

// --- makeRange ---

func TestMakeRangeSingle(t *testing.T) {
	t.Parallel()
	r := makeRange(5, 5)
	if len(r) != 1 || r[0] != 5 {
		t.Errorf("makeRange(5,5) = %v, want [5]", r)
	}
}

func TestMakeRangeMultiple(t *testing.T) {
	t.Parallel()
	r := makeRange(0, 2)
	if len(r) != 3 || r[0] != 0 || r[1] != 1 || r[2] != 2 {
		t.Errorf("makeRange(0,2) = %v, want [0 1 2]", r)
	}
}

// --- contains ---

func TestContainsTrue(t *testing.T) {
	t.Parallel()
	if !contains([]int{1, 2, 3}, 2) {
		t.Error("contains should return true when value is in slice")
	}
}

func TestContainsFalse(t *testing.T) {
	t.Parallel()
	if contains([]int{1, 2, 3}, 5) {
		t.Error("contains should return false when value is not in slice")
	}
}

func TestContainsEmptySlice(t *testing.T) {
	t.Parallel()
	if contains([]int{}, 0) {
		t.Error("contains on empty slice should return false")
	}
}

// --- RunTask ---

func TestRunTaskNotFound(t *testing.T) {
	t.Parallel()
	s := NewSchedulerService()
	err := s.RunTask(context.Background(), "nonexistent")
	if err == nil {
		t.Error("RunTask on nonexistent task should return error")
	}
}

func TestRunTaskSuccess(t *testing.T) {
	t.Parallel()
	s := NewSchedulerService()
	called := false
	s.AddTask("run_me", "0 * * * *", func(ctx context.Context) error {
		called = true
		return nil
	})
	if err := s.RunTask(context.Background(), "run_me"); err != nil {
		t.Errorf("RunTask: %v", err)
	}
	if !called {
		t.Error("handler should have been called by RunTask")
	}
}

func TestRunTaskHandlerError(t *testing.T) {
	t.Parallel()
	s := NewSchedulerService()
	s.AddTask("failing_task", "0 * * * *", func(ctx context.Context) error {
		return errors.New("task failed")
	})
	if err := s.RunTask(context.Background(), "failing_task"); err == nil {
		t.Error("RunTask should propagate handler error")
	}
}

func TestRunTaskDisabled(t *testing.T) {
	t.Parallel()
	s := NewSchedulerService()
	s.AddTask("disabled_task", "0 * * * *", func(ctx context.Context) error {
		return nil
	})
	s.DisableTask("disabled_task")
	// Disabled tasks: RunTask should either return error or skip
	// The key property is it must not panic
	_ = s.RunTask(context.Background(), "disabled_task")
}
