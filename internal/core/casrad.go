package core

import (
	"context"
	"embed"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"sync"
	"syscall"
	"time"

	"github.com/casapps/casrad/internal/backup"
	"github.com/casapps/casrad/internal/cache"
	"github.com/casapps/casrad/internal/database"
	"github.com/casapps/casrad/internal/media"
	"github.com/casapps/casrad/internal/metrics"
	"github.com/casapps/casrad/internal/migration"
	"github.com/casapps/casrad/internal/protocols"
	"github.com/casapps/casrad/internal/scheduler"
	"github.com/casapps/casrad/internal/security"
	"github.com/casapps/casrad/internal/server"
	"github.com/casapps/casrad/internal/setup"
	"github.com/casapps/casrad/internal/storage"
	"github.com/casapps/casrad/internal/web"
)

// WebAssets contains embedded static files
var WebAssets embed.FS

// Templates contains embedded templates
var Templates embed.FS

type Config struct {
	Port     int
	DataPath string
	Debug    bool
}

type CASRAD struct {
	config *Config
	mu     sync.RWMutex

	// Core Components
	WebServer      *server.HTTPServer
	Database       *database.Engine
	Cache          *cache.CacheLayer
	ThemeEngine    *ThemeManager
	Scheduler      *scheduler.TaskScheduler
	SecurityMgr    *security.Manager
	StorageManager *storage.Manager

	// Protocol Servers
	MPDServer    *protocols.MPDServer
	SubsonicAPI  *protocols.SubsonicServer
	AmpacheAPI   *protocols.AmpacheServer
	WebDAVServer *protocols.WebDAVServer
	RTMPServer   *protocols.RTMPServer
	DLNAServer   *protocols.DLNAServer

	// Management Systems
	ServiceManager   *ServiceInstaller
	OSHandler        OSHandler
	FFMPEGManager    *media.FFMPEGManager
	Transcoder       *media.Transcoder
	BackupManager    *backup.BackupManager
	SetupWizard      *setup.SetupWizard
	CertManager      *security.CertificateManager
	PodcastManager   *media.PodcastManager
	MusicBrainz      *media.MusicBrainzClient
	AutoDJ           *media.AutoDJ
	Migrator         *migration.Migrator
	MetricsCollector *metrics.Collector

	// Runtime state
	ctx    context.Context
	cancel context.CancelFunc
}

func NewCASRAD(config *Config) (*CASRAD, error) {
	ctx, cancel := context.WithCancel(context.Background())

	app := &CASRAD{
		config: config,
		ctx:    ctx,
		cancel: cancel,
	}

	// Detect OS and create appropriate handler
	app.OSHandler = app.detectOS()

	// Set default data path if not provided
	if config.DataPath == "" {
		config.DataPath = app.OSHandler.GetDefaultDataPath()
	}

	// Initialize components
	if err := app.initialize(); err != nil {
		cancel()
		return nil, fmt.Errorf("initialization failed: %w", err)
	}

	return app, nil
}

func (c *CASRAD) initialize() error {
	log.Println("Initializing CASRAD...")

	// Create directory structure
	if err := c.OSHandler.CreateDirectories(c.config.DataPath); err != nil {
		return fmt.Errorf("failed to create directories: %w", err)
	}

	// Initialize database
	dbPath := c.OSHandler.GetDatabasePath(c.config.DataPath)
	db, err := database.New(database.SQLite, dbPath)
	if err != nil {
		return fmt.Errorf("failed to initialize database: %w", err)
	}
	c.Database = db

	// Initialize cache layer (memory cache by default, auto-sizing)
	c.Cache = cache.NewCacheLayer(cache.Memory, 0)

	// Initialize storage manager
	c.StorageManager = storage.NewManager(c.config.DataPath, c.Database)

	// Initialize security manager
	c.SecurityMgr = security.NewManager(c.Database)

	// Initialize scheduler
	c.Scheduler = scheduler.New(c.Database)
	c.Scheduler.InitializeDefaultTasks()

	// Determine port
	port := c.config.Port
	if port == 0 {
		port = c.OSHandler.GetDefaultPort()
	}

	// Initialize web server
	c.WebServer = server.New(port, c.Database, web.Assets, web.Templates)

	// Initialize theme manager
	c.ThemeEngine = NewThemeManager(c.Database)

	// Initialize protocol servers
	c.MPDServer = protocols.NewMPDServer(6600, c.Database)
	c.SubsonicAPI = protocols.NewSubsonicServer(c.Database)
	c.AmpacheAPI = protocols.NewAmpacheServer(c.Database)
	c.WebDAVServer = protocols.NewWebDAVServer(c.Database)
	c.RTMPServer = protocols.NewRTMPServer(1935, c.Database, c.FFMPEGManager, c.Transcoder)
	c.DLNAServer = protocols.NewDLNAServer(c.Database, port)

	// Register API protocols with web server
	c.WebServer.RegisterSubsonicAPI(c.SubsonicAPI)
	c.WebServer.RegisterAmpacheAPI(c.AmpacheAPI)
	c.WebServer.RegisterWebDAV(c.WebDAVServer)
	c.WebServer.RegisterDLNA(c.DLNAServer)

	// Initialize media components
	c.FFMPEGManager = media.NewFFMPEGManager(c.config.DataPath, c.Database)

	// Download FFMPEG automatically on first run if not available
	if !c.FFMPEGManager.IsAvailable() {
		log.Println("FFMPEG not found, initiating automatic download...")
		go func() {
			if err := c.FFMPEGManager.DownloadFFMPEG(); err != nil {
				log.Printf("Failed to download FFMPEG: %v", err)
				log.Println("Transcoding features will be unavailable until FFMPEG is installed")
			} else {
				log.Println("FFMPEG downloaded successfully")
			}
		}()
	}

	// Initialize transcoder with cache path
	cachePath := filepath.Join(c.config.DataPath, "cache", "transcode")
	c.Transcoder = media.NewTranscoder(cachePath, c.Database, c.FFMPEGManager)

	// Initialize podcast manager
	podcastPath := filepath.Join(c.config.DataPath, "podcasts")
	c.PodcastManager = media.NewPodcastManager(c.Database, podcastPath, c.FFMPEGManager)

	// Initialize MusicBrainz client
	c.MusicBrainz = media.NewMusicBrainzClient(c.Database, c.FFMPEGManager)

	// Initialize AutoDJ
	c.AutoDJ = media.NewAutoDJ(c.Database, c.Transcoder)

	// Initialize certificate manager
	certPath := filepath.Join(c.config.DataPath, "certs")
	if c.OSHandler.HasPrivileges() {
		certPath = "/etc/casrad/certs"
	}
	c.CertManager = security.NewCertificateManager(c.Database, certPath, port, 443)

	// Initialize backup manager
	backupPath := filepath.Join(c.config.DataPath, "backups")
	if c.OSHandler.HasPrivileges() {
		backupPath = "/etc/casrad/backups"
	}
	c.BackupManager = backup.NewBackupManager(backupPath, c.Database)

	// Initialize setup wizard
	c.SetupWizard = setup.NewSetupWizard(c.Database, c.StorageManager)

	// Initialize migrator
	c.Migrator = migration.NewMigrator(c.Database)

	// Initialize metrics collector
	c.MetricsCollector = metrics.NewCollector(c.Database)

	// Check if this is first run
	if isFirstRun, err := c.Database.IsFirstRun(); err == nil && isFirstRun {
		log.Println("First run detected, setup wizard will be shown")
		c.WebServer.EnableSetupMode()
	}

	// Try to install as service if running with privileges
	if c.OSHandler.HasPrivileges() && !c.OSHandler.IsRunningAsService() {
		c.attemptServiceInstallation()
	}

	return nil
}

func (c *CASRAD) Run() error {
	// Start scheduler
	go c.Scheduler.Run(c.ctx)

	// Start security manager background tasks
	go c.SecurityMgr.RunBackgroundTasks(c.ctx)

	// Start metrics collector
	c.MetricsCollector.Start()

	// Start protocol servers
	if err := c.MPDServer.Start(); err != nil {
		log.Printf("Failed to start MPD server: %v", err)
	}

	if err := c.RTMPServer.Start(); err != nil {
		log.Printf("Failed to start RTMP server: %v", err)
	}

	if err := c.DLNAServer.Start(); err != nil {
		log.Printf("Failed to start DLNA server: %v", err)
	}

	// Start web server
	serverErr := make(chan error, 1)
	go func() {
		log.Printf("Starting web server on port %d", c.WebServer.Port)
		if err := c.WebServer.Start(); err != nil {
			serverErr <- err
		}
	}()

	// Display startup information
	c.displayStartupInfo()

	// Setup signal handling
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// Wait for shutdown signal or error
	select {
	case <-sigChan:
		log.Println("Shutdown signal received")
		return c.shutdown()
	case err := <-serverErr:
		return fmt.Errorf("server error: %w", err)
	case <-c.ctx.Done():
		return c.shutdown()
	}
}

func (c *CASRAD) shutdown() error {
	log.Println("Shutting down CASRAD...")

	// Cancel context to stop background tasks
	c.cancel()

	// Shutdown components with timeout
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	var wg sync.WaitGroup

	// Shutdown web server
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := c.WebServer.Shutdown(shutdownCtx); err != nil {
			log.Printf("Web server shutdown error: %v", err)
		}
	}()

	// Stop scheduler
	wg.Add(1)
	go func() {
		defer wg.Done()
		c.Scheduler.Stop()
	}()

	// Stop metrics collector
	wg.Add(1)
	go func() {
		defer wg.Done()
		c.MetricsCollector.Stop()
	}()

	// Close database
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := c.Database.Close(); err != nil {
			log.Printf("Database close error: %v", err)
		}
	}()

	// Wait for all shutdowns to complete
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		log.Println("CASRAD shutdown complete")
		return nil
	case <-shutdownCtx.Done():
		log.Println("Shutdown timeout exceeded, forcing exit")
		return fmt.Errorf("shutdown timeout")
	}
}

func (c *CASRAD) displayStartupInfo() {
	fmt.Println()
	fmt.Println("╔══════════════════════════════════════════════════════════════╗")
	fmt.Println("║                          CASRAD                              ║")
	fmt.Println("║     Complete Audio Streaming, Radio, and Distribution        ║")
	fmt.Println("╚══════════════════════════════════════════════════════════════╝")
	fmt.Println()
	fmt.Printf("  Web Interface:  http://localhost:%d\n", c.WebServer.Port)
	fmt.Printf("  MPD Port:       6600\n")
	fmt.Printf("  RTMP Port:      1935\n")
	fmt.Printf("  Data Directory: %s\n", c.config.DataPath)
	fmt.Printf("  Platform:       %s/%s\n", runtime.GOOS, runtime.GOARCH)
	fmt.Println()
	fmt.Println("  Press Ctrl+C to shutdown")
	fmt.Println()
}

func (c *CASRAD) attemptServiceInstallation() {
	log.Println("Running with privileges, attempting service installation...")
	c.ServiceManager = NewServiceInstaller(c.OSHandler)
	if err := c.ServiceManager.Install(); err != nil {
		log.Printf("Service installation failed: %v", err)
		log.Println("Continuing in standalone mode")
	} else {
		log.Println("Service installed successfully")
		// The service installer may restart the process as a service
	}
}