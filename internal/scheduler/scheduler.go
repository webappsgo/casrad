package scheduler

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/casapps/casrad/internal/database"
	"github.com/robfig/cron/v3"
)

type Task struct {
	Name     string
	Schedule string
	Handler  func(context.Context) error
	Type     string
}

type TaskScheduler struct {
	db       *database.Engine
	cron     *cron.Cron
	tasks    map[string]*Task
	mu       sync.RWMutex
	ctx      context.Context
	cancel   context.CancelFunc
	wg       sync.WaitGroup
}

func New(db *database.Engine) *TaskScheduler {
	return &TaskScheduler{
		db:    db,
		cron:  cron.New(),
		tasks: make(map[string]*Task),
	}
}

func (s *TaskScheduler) InitializeDefaultTasks() {
	// Register default task handlers
	s.RegisterTask("cleanup_temp", "0 * * * *", s.cleanTempFiles, "cleanup")
	s.RegisterTask("cleanup_cache", "0 */6 * * *", s.cleanCache, "cleanup")
	s.RegisterTask("rotate_logs", "0 3 * * *", s.rotateLogs, "cleanup")
	s.RegisterTask("cleanup_transcodes", "0 4 * * *", s.cleanTranscodes, "cleanup")
	s.RegisterTask("backup_database", "0 2 * * *", s.backupDatabase, "backup")
	s.RegisterTask("check_quotas", "*/30 * * * *", s.checkUserQuotas, "check")
	s.RegisterTask("renew_certificates", "0 1 * * *", s.checkCertificates, "update")
	s.RegisterTask("update_podcasts", "0 */6 * * *", s.updatePodcasts, "update")
	s.RegisterTask("scan_libraries", "0 3 * * *", s.scanLibraries, "scan")
	s.RegisterTask("update_geoip", "0 2 * * 0", s.updateGeoIP, "update")
	s.RegisterTask("update_security_lists", "0 3 * * *", s.updateSecurityLists, "update")
	s.RegisterTask("check_ffmpeg", "0 3 * * 0", s.checkFFMPEGUpdate, "update")
	s.RegisterTask("aggregate_metrics", "*/5 * * * *", s.aggregateMetrics, "metrics")
	s.RegisterTask("check_schema", "0 0 * * *", s.checkSchemaVersion, "maintenance")
}

func (s *TaskScheduler) RegisterTask(name, schedule string, handler func(context.Context) error, taskType string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.tasks[name] = &Task{
		Name:     name,
		Schedule: schedule,
		Handler:  handler,
		Type:     taskType,
	}
}

func (s *TaskScheduler) Run(ctx context.Context) {
	s.ctx, s.cancel = context.WithCancel(ctx)

	// Load and schedule enabled tasks from database
	s.loadAndScheduleTasks()

	// Start cron scheduler
	s.cron.Start()

	// Wait for context cancellation
	<-s.ctx.Done()
	s.cron.Stop()
}

func (s *TaskScheduler) Stop() {
	if s.cancel != nil {
		s.cancel()
	}
	s.wg.Wait()
}

func (s *TaskScheduler) loadAndScheduleTasks() {
	rows, err := s.db.Query(`
		SELECT name, schedule
		FROM scheduled_tasks
		WHERE is_enabled = 1
	`)
	if err != nil {
		log.Printf("Error loading scheduled tasks: %v", err)
		return
	}
	defer rows.Close()

	for rows.Next() {
		var name, schedule string
		if err := rows.Scan(&name, &schedule); err != nil {
			continue
		}

		if task, ok := s.tasks[name]; ok {
			s.scheduleTask(task)
		}
	}
}

func (s *TaskScheduler) scheduleTask(task *Task) {
	entryID, err := s.cron.AddFunc(task.Schedule, func() {
		s.runTask(task)
	})
	if err != nil {
		log.Printf("Error scheduling task %s: %v", task.Name, err)
		return
	}
	log.Printf("Scheduled task %s with ID %d", task.Name, entryID)
}

func (s *TaskScheduler) runTask(task *Task) {
	s.wg.Add(1)
	defer s.wg.Done()

	startTime := time.Now()
	log.Printf("Running task: %s", task.Name)

	// Update task status to running
	s.updateTaskStatus(task.Name, "running", nil)

	// Run the task
	err := task.Handler(s.ctx)

	duration := time.Since(startTime)

	if err != nil {
		log.Printf("Task %s failed: %v", task.Name, err)
		s.updateTaskStatus(task.Name, "failed", err)
	} else {
		log.Printf("Task %s completed in %v", task.Name, duration)
		s.updateTaskStatus(task.Name, "success", nil)
	}

	// Update execution statistics
	s.updateTaskStats(task.Name, duration)
}

func (s *TaskScheduler) updateTaskStatus(name, status string, err error) {
	var errStr *string
	if err != nil {
		e := err.Error()
		errStr = &e
	}

	s.db.Exec(`
		UPDATE scheduled_tasks
		SET last_status = ?,
		    last_error = ?,
		    last_run = ?
		WHERE name = ?
	`, status, errStr, time.Now(), name)
}

func (s *TaskScheduler) updateTaskStats(name string, duration time.Duration) {
	ms := int(duration.Milliseconds())
	s.db.Exec(`
		UPDATE scheduled_tasks
		SET run_count = run_count + 1,
		    average_duration_ms = (average_duration_ms * run_count + ?) / (run_count + 1),
		    max_duration_ms = MAX(max_duration_ms, ?)
		WHERE name = ?
	`, ms, ms, name)
}

// Task implementations

func (s *TaskScheduler) cleanTempFiles(ctx context.Context) error {
	// TODO: Implement temp file cleanup
	log.Println("Cleaning temporary files...")
	return nil
}

func (s *TaskScheduler) cleanCache(ctx context.Context) error {
	// TODO: Implement cache cleanup
	log.Println("Cleaning cache...")
	return nil
}

func (s *TaskScheduler) rotateLogs(ctx context.Context) error {
	// TODO: Implement log rotation
	log.Println("Rotating logs...")
	return nil
}

func (s *TaskScheduler) cleanTranscodes(ctx context.Context) error {
	// TODO: Implement transcode cleanup
	log.Println("Cleaning transcodes...")
	return nil
}

func (s *TaskScheduler) backupDatabase(ctx context.Context) error {
	// TODO: Implement database backup
	log.Println("Backing up database...")
	return nil
}

func (s *TaskScheduler) checkUserQuotas(ctx context.Context) error {
	// TODO: Implement quota checking
	log.Println("Checking user quotas...")
	return nil
}

func (s *TaskScheduler) checkCertificates(ctx context.Context) error {
	// TODO: Implement certificate renewal
	log.Println("Checking certificates...")
	return nil
}

func (s *TaskScheduler) updatePodcasts(ctx context.Context) error {
	// TODO: Implement podcast updates
	log.Println("Updating podcasts...")
	return nil
}

func (s *TaskScheduler) scanLibraries(ctx context.Context) error {
	// TODO: Implement library scanning
	log.Println("Scanning libraries...")
	return nil
}

func (s *TaskScheduler) updateGeoIP(ctx context.Context) error {
	// TODO: Implement GeoIP update
	log.Println("Updating GeoIP databases...")
	return nil
}

func (s *TaskScheduler) updateSecurityLists(ctx context.Context) error {
	// TODO: Implement security list updates
	log.Println("Updating security lists...")
	return nil
}

func (s *TaskScheduler) checkFFMPEGUpdate(ctx context.Context) error {
	// TODO: Implement FFMPEG update check
	log.Println("Checking FFMPEG updates...")
	return nil
}

func (s *TaskScheduler) aggregateMetrics(ctx context.Context) error {
	// TODO: Implement metrics aggregation
	return nil
}

func (s *TaskScheduler) checkSchemaVersion(ctx context.Context) error {
	// TODO: Implement schema version check
	return nil
}