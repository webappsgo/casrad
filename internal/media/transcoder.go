package media

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/casapps/casrad/internal/database"
)

// AudioFormat represents supported audio formats
type AudioFormat string

const (
	FormatMP3  AudioFormat = "mp3"
	FormatAAC  AudioFormat = "aac"
	FormatOpus AudioFormat = "opus"
	FormatOGG  AudioFormat = "ogg"
	FormatFLAC AudioFormat = "flac"
	FormatWAV  AudioFormat = "wav"
)

// TranscodeOptions contains transcoding parameters
type TranscodeOptions struct {
	Format      AudioFormat
	Bitrate     int    // kbps
	SampleRate  int    // Hz
	Channels    int    // 1 for mono, 2 for stereo
	Quality     string // low, medium, high, lossless
	StartTime   int    // seconds
	Duration    int    // seconds, 0 for full track
	Normalize   bool   // Apply loudness normalization
	ReplayGain  bool   // Apply ReplayGain
	Crossfade   int    // Crossfade duration in seconds
}

// Transcoder handles audio format conversion
type Transcoder struct {
	db            *database.Engine
	ffmpeg        *FFMPEGManager
	cachePath     string
	maxCacheSize  int64 // bytes
	workers       int
	mu            sync.RWMutex
	activeJobs    map[string]*TranscodeJob
	jobQueue      chan *TranscodeJob
	workerWg      sync.WaitGroup
}

// TranscodeJob represents a transcoding task
type TranscodeJob struct {
	ID          string
	InputPath   string
	OutputPath  string
	Options     TranscodeOptions
	Progress    float64
	Status      string
	Error       error
	StartedAt   time.Time
	CompletedAt time.Time
	ctx         context.Context
	cancel      context.CancelFunc
}

func NewTranscoder(cachePath string, db *database.Engine, ffmpeg *FFMPEGManager) *Transcoder {
	// Default to 4 transcode workers
	workers := 4
	if w, err := db.GetSetting("audio.transcode_threads"); err == nil {
		if n, err := strconv.Atoi(w); err == nil && n > 0 {
			workers = n
		}
	}

	t := &Transcoder{
		db:           db,
		ffmpeg:       ffmpeg,
		cachePath:    cachePath,
		maxCacheSize: 10 * 1024 * 1024 * 1024, // 10GB default
		workers:      workers,
		activeJobs:   make(map[string]*TranscodeJob),
		jobQueue:     make(chan *TranscodeJob, 100),
	}

	// Create cache directory
	os.MkdirAll(cachePath, 0755)

	// Start worker pool
	t.startWorkers()

	return t
}

func (t *Transcoder) startWorkers() {
	for i := 0; i < t.workers; i++ {
		t.workerWg.Add(1)
		go t.worker(i)
	}
}

func (t *Transcoder) worker(id int) {
	defer t.workerWg.Done()

	for job := range t.jobQueue {
		log.Printf("Worker %d: Starting transcode job %s", id, job.ID)
		t.processJob(job)
	}
}

func (t *Transcoder) Stop() {
	close(t.jobQueue)
	t.workerWg.Wait()

	// Cancel all active jobs
	t.mu.Lock()
	for _, job := range t.activeJobs {
		if job.cancel != nil {
			job.cancel()
		}
	}
	t.mu.Unlock()
}

// Transcode starts a transcoding job
func (t *Transcoder) Transcode(inputPath string, options TranscodeOptions) (*TranscodeJob, error) {
	if !t.ffmpeg.IsAvailable() {
		// Try to download FFMPEG if not available
		go t.ffmpeg.DownloadFFMPEG()
		return nil, fmt.Errorf("FFMPEG not available, download initiated")
	}

	// Generate job ID and output path
	jobID := t.generateJobID(inputPath, options)
	outputPath := t.generateOutputPath(inputPath, options)

	// Check if already transcoded (in cache)
	if _, err := os.Stat(outputPath); err == nil {
		// File exists in cache, return immediately
		return &TranscodeJob{
			ID:          jobID,
			InputPath:   inputPath,
			OutputPath:  outputPath,
			Options:     options,
			Progress:    100,
			Status:      "completed",
			CompletedAt: time.Now(),
		}, nil
	}

	// Create new job
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	job := &TranscodeJob{
		ID:         jobID,
		InputPath:  inputPath,
		OutputPath: outputPath,
		Options:    options,
		Status:     "queued",
		StartedAt:  time.Now(),
		ctx:        ctx,
		cancel:     cancel,
	}

	// Add to active jobs
	t.mu.Lock()
	t.activeJobs[jobID] = job
	t.mu.Unlock()

	// Queue the job
	t.jobQueue <- job

	return job, nil
}

// StreamTranscode performs real-time transcoding for streaming
func (t *Transcoder) StreamTranscode(ctx context.Context, inputPath string, options TranscodeOptions, output io.Writer) error {
	if !t.ffmpeg.IsAvailable() {
		return fmt.Errorf("FFMPEG not available")
	}

	args := t.buildFFMPEGArgs(inputPath, "-", options)

	cmd := exec.CommandContext(ctx, t.ffmpeg.GetFFMPEGPath(), args...)
	cmd.Stdout = output

	// Create stderr pipe for progress monitoring
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return err
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start FFMPEG: %w", err)
	}

	// Monitor progress in goroutine
	go t.monitorFFMPEGOutput(stderr, nil)

	return cmd.Wait()
}

func (t *Transcoder) processJob(job *TranscodeJob) {
	job.Status = "processing"

	// Ensure output directory exists
	os.MkdirAll(filepath.Dir(job.OutputPath), 0755)

	// Build FFMPEG command
	args := t.buildFFMPEGArgs(job.InputPath, job.OutputPath, job.Options)

	cmd := exec.CommandContext(job.ctx, t.ffmpeg.GetFFMPEGPath(), args...)

	// Create stderr pipe for progress monitoring
	stderr, err := cmd.StderrPipe()
	if err != nil {
		job.Status = "failed"
		job.Error = err
		job.CompletedAt = time.Now()
		return
	}

	// Start FFMPEG
	if err := cmd.Start(); err != nil {
		job.Status = "failed"
		job.Error = err
		job.CompletedAt = time.Now()
		return
	}

	// Monitor progress
	go t.monitorFFMPEGOutput(stderr, job)

	// Wait for completion
	if err := cmd.Wait(); err != nil {
		// Check if cancelled
		select {
		case <-job.ctx.Done():
			job.Status = "cancelled"
		default:
			job.Status = "failed"
			job.Error = err
		}
	} else {
		job.Status = "completed"
		job.Progress = 100
	}

	job.CompletedAt = time.Now()

	// Clean up from active jobs
	t.mu.Lock()
	delete(t.activeJobs, job.ID)
	t.mu.Unlock()

	// Clean up old cache files if needed
	go t.cleanCache()
}

func (t *Transcoder) buildFFMPEGArgs(input, output string, options TranscodeOptions) []string {
	args := []string{
		"-i", input,
		"-y", // Overwrite output file
	}

	// Add start time if specified
	if options.StartTime > 0 {
		args = append(args, "-ss", fmt.Sprintf("%d", options.StartTime))
	}

	// Add duration if specified
	if options.Duration > 0 {
		args = append(args, "-t", fmt.Sprintf("%d", options.Duration))
	}

	// Audio codec and format
	switch options.Format {
	case FormatMP3:
		args = append(args, "-codec:a", "libmp3lame")
		if options.Bitrate > 0 {
			args = append(args, "-b:a", fmt.Sprintf("%dk", options.Bitrate))
		} else {
			// Default quality-based encoding
			args = append(args, "-q:a", t.getMP3Quality(options.Quality))
		}

	case FormatAAC:
		args = append(args, "-codec:a", "aac")
		if options.Bitrate > 0 {
			args = append(args, "-b:a", fmt.Sprintf("%dk", options.Bitrate))
		} else {
			args = append(args, "-q:a", "2") // Default quality
		}

	case FormatOpus:
		args = append(args, "-codec:a", "libopus")
		if options.Bitrate > 0 {
			args = append(args, "-b:a", fmt.Sprintf("%dk", options.Bitrate))
		} else {
			args = append(args, "-b:a", "128k") // Default bitrate for Opus
		}

	case FormatOGG:
		args = append(args, "-codec:a", "libvorbis")
		if options.Bitrate > 0 {
			args = append(args, "-b:a", fmt.Sprintf("%dk", options.Bitrate))
		} else {
			args = append(args, "-q:a", t.getOggQuality(options.Quality))
		}

	case FormatFLAC:
		args = append(args, "-codec:a", "flac")
		args = append(args, "-compression_level", t.getFLACCompression(options.Quality))

	case FormatWAV:
		args = append(args, "-codec:a", "pcm_s16le")
	}

	// Sample rate
	if options.SampleRate > 0 {
		args = append(args, "-ar", fmt.Sprintf("%d", options.SampleRate))
	}

	// Channels
	if options.Channels > 0 {
		args = append(args, "-ac", fmt.Sprintf("%d", options.Channels))
	}

	// Normalization
	if options.Normalize {
		args = append(args, "-af", "loudnorm=I=-23:LRA=7:TP=-2")
	}

	// ReplayGain (using volume filter as approximation)
	if options.ReplayGain {
		args = append(args, "-af", "volume=replaygain=track")
	}

	// Output format
	if output == "-" {
		// Streaming output
		args = append(args, "-f", string(options.Format))
		if options.Format == FormatMP3 {
			args = append(args, "-f", "mp3")
		}
	}

	args = append(args, output)

	return args
}

func (t *Transcoder) monitorFFMPEGOutput(stderr io.ReadCloser, job *TranscodeJob) {
	defer stderr.Close()

	buf := make([]byte, 1024)
	var output strings.Builder

	for {
		n, err := stderr.Read(buf)
		if n > 0 {
			output.Write(buf[:n])

			// Parse progress if job provided
			if job != nil {
				lines := strings.Split(output.String(), "\n")
				for _, line := range lines {
					if strings.Contains(line, "time=") {
						// Parse time and calculate progress
						// This is simplified - real implementation would parse duration first
						if timeStr := t.extractTime(line); timeStr != "" {
							// Update job progress
							// job.Progress = calculateProgress(timeStr, totalDuration)
						}
					}
				}
			}
		}

		if err != nil {
			break
		}
	}
}

func (t *Transcoder) extractTime(line string) string {
	if idx := strings.Index(line, "time="); idx >= 0 {
		timeStr := line[idx+5:]
		if endIdx := strings.Index(timeStr, " "); endIdx > 0 {
			return timeStr[:endIdx]
		}
	}
	return ""
}

func (t *Transcoder) generateJobID(inputPath string, options TranscodeOptions) string {
	// Generate unique ID based on input and options
	data := fmt.Sprintf("%s-%v", inputPath, options)
	return fmt.Sprintf("%x", data)[:16]
}

func (t *Transcoder) generateOutputPath(inputPath string, options TranscodeOptions) string {
	// Generate cache file path
	baseName := filepath.Base(inputPath)
	ext := filepath.Ext(baseName)
	nameWithoutExt := strings.TrimSuffix(baseName, ext)

	// Build filename with options
	outputName := fmt.Sprintf("%s_%s_%dk.%s",
		nameWithoutExt,
		options.Format,
		options.Bitrate,
		options.Format,
	)

	return filepath.Join(t.cachePath, outputName)
}

func (t *Transcoder) getMP3Quality(quality string) string {
	switch quality {
	case "low":
		return "9"
	case "medium":
		return "5"
	case "high":
		return "2"
	case "lossless":
		return "0"
	default:
		return "5"
	}
}

func (t *Transcoder) getOggQuality(quality string) string {
	switch quality {
	case "low":
		return "1"
	case "medium":
		return "5"
	case "high":
		return "8"
	case "lossless":
		return "10"
	default:
		return "5"
	}
}

func (t *Transcoder) getFLACCompression(quality string) string {
	switch quality {
	case "low":
		return "0"
	case "medium":
		return "5"
	case "high":
		return "8"
	case "lossless":
		return "12"
	default:
		return "5"
	}
}

func (t *Transcoder) cleanCache() {
	// Calculate cache size
	var totalSize int64
	var files []os.FileInfo

	err := filepath.Walk(t.cachePath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if !info.IsDir() {
			files = append(files, info)
			totalSize += info.Size()
		}
		return nil
	})

	if err != nil || totalSize <= t.maxCacheSize {
		return
	}

	// Sort files by access time (oldest first)
	// Delete oldest files until under limit
	deleteSize := totalSize - t.maxCacheSize
	var deleted int64

	for _, file := range files {
		if deleted >= deleteSize {
			break
		}

		filePath := filepath.Join(t.cachePath, file.Name())
		if err := os.Remove(filePath); err == nil {
			deleted += file.Size()
			log.Printf("Cleaned cache file: %s", file.Name())
		}
	}
}

// GetJob returns the status of a transcode job
func (t *Transcoder) GetJob(jobID string) (*TranscodeJob, bool) {
	t.mu.RLock()
	defer t.mu.RUnlock()

	job, exists := t.activeJobs[jobID]
	return job, exists
}

// CancelJob cancels an active transcode job
func (t *Transcoder) CancelJob(jobID string) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	job, exists := t.activeJobs[jobID]
	if !exists {
		return fmt.Errorf("job not found")
	}

	if job.cancel != nil {
		job.cancel()
	}

	return nil
}

// GetActiveJobs returns all active transcode jobs
func (t *Transcoder) GetActiveJobs() []*TranscodeJob {
	t.mu.RLock()
	defer t.mu.RUnlock()

	jobs := make([]*TranscodeJob, 0, len(t.activeJobs))
	for _, job := range t.activeJobs {
		jobs = append(jobs, job)
	}

	return jobs
}