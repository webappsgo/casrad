// CASRAD - Complete Audio Streaming, Radio, and Distribution
// See AI.md for specification details
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/casapps/casrad/src/config"
	"github.com/casapps/casrad/src/mode"
	"github.com/casapps/casrad/src/scheduler"
	"github.com/casapps/casrad/src/server"
	"github.com/casapps/casrad/src/server/handler"
	"github.com/casapps/casrad/src/service"
)

// Version information - set by ldflags during build
var (
	Version      = "dev"
	CommitID     = "unknown"
	BuildDate    = "unknown"
	OfficialSite = ""
)

// colorDisabled reports whether terminal color should be suppressed.
// Priority: --color flag > NO_COLOR env > auto (TTY)
func colorDisabled(colorFlag string) bool {
	switch strings.ToLower(colorFlag) {
	case "always":
		return false
	case "never":
		return true
	}
	// auto or unset: respect NO_COLOR (https://no-color.org/)
	_, set := os.LookupEnv("NO_COLOR")
	return set
}

func main() {
	// Initialize mode from environment; CLI flags override below
	mode.Init()

	args := os.Args[1:]

	showHelp := false
	showVersion := false
	port := 0
	address := ""
	debug := false
	dataPath := ""
	configPath := ""
	modeFlag := ""
	colorFlag := ""
	langFlag := ""
	shellFlag := ""

	// Service management flags
	serviceFlag := false
	serviceAction := ""

	// Maintenance flags
	maintenanceFlag := false
	maintenanceAction := ""
	maintenanceArg := ""

	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch arg {
		case "-h", "--help":
			showHelp = true

		case "-v", "--version":
			showVersion = true

		case "--config":
			if i+1 < len(args) {
				i++
				configPath = args[i]
			}

		case "-p", "--port":
			if i+1 < len(args) {
				i++
				if p, err := strconv.Atoi(args[i]); err == nil {
					port = p
				}
			}

		case "-a", "--address":
			if i+1 < len(args) {
				i++
				address = args[i]
			}

		case "-d", "--data":
			if i+1 < len(args) {
				i++
				dataPath = args[i]
			}

		case "--mode":
			if i+1 < len(args) {
				i++
				modeFlag = args[i]
			}

		case "--debug":
			debug = true

		case "--color":
			if i+1 < len(args) {
				i++
				colorFlag = args[i]
			}

		case "--lang":
			if i+1 < len(args) {
				i++
				langFlag = args[i]
			}

		case "--shell":
			if i+1 < len(args) {
				i++
				shellFlag = args[i]
			}

		case "--service":
			serviceFlag = true

		// Service sub-actions (follow --service)
		case "--install", "--uninstall", "--enable", "--disable",
			"--status", "--restart", "--start", "--stop":
			if serviceFlag {
				serviceAction = strings.TrimPrefix(arg, "--")
			}

		case "--maintenance":
			maintenanceFlag = true
			// Consume the action word that follows
			if i+1 < len(args) {
				next := args[i+1]
				switch next {
				case "backup", "restore", "update":
					i++
					maintenanceAction = next
					// backup accepts an optional filename
					// restore accepts a required filename
					if i+1 < len(args) && !strings.HasPrefix(args[i+1], "-") {
						i++
						maintenanceArg = args[i]
					}
				}
			}
		}
	}

	// Apply NO_COLOR / color preference before any terminal output
	_ = colorDisabled(colorFlag)

	// Apply mode flag (CLI overrides environment)
	if modeFlag != "" {
		mode.Set(mode.FromString(modeFlag))
	}
	if debug {
		mode.SetDebug(true)
	}

	binaryName := filepath.Base(os.Args[0])

	// --shell: print completion stub and exit
	if shellFlag != "" {
		fmt.Printf("# %s completions for %s — not yet generated\n", shellFlag, binaryName)
		os.Exit(0)
	}

	// --version: "casrad 1.0.0 (abc1234) built Sun Jan 01, 2025 at 00:00:00 UTC"
	if showVersion {
		fmt.Printf("%s %s (%s) built %s\n", binaryName, Version, CommitID, BuildDate)
		os.Exit(0)
	}

	if showHelp {
		fmt.Printf(`%s - Complete Audio Streaming, Radio, and Distribution

Usage: %s [flags]

Flags:
  -h, --help                  Show this help message
  -v, --version               Show version information
  --config PATH               Path to config file (default: {config_dir}/server.yml)
  -p, --port PORT             Override port (default: auto 64000-64999)
  -a, --address ADDR          Override bind address (default: 0.0.0.0)
  -d, --data PATH             Override data directory
  --mode MODE                 Application mode: production|development|prod|dev
  --debug                     Enable debug logging (BYPASSES admin auth)
  --color always|never|auto   Color output (default: auto)
  --lang CODE                 Language code (default: en)
  --shell bash|zsh|fish       Output shell completions

  --service ACTION            Manage system service
    Actions: --install --uninstall --enable --disable
             --status  --restart   --start  --stop

  --maintenance ACTION        Maintenance operations
    Actions: backup [filename]
             restore <file>
             update

Environment Variables:
  CASRAD_PORT        Override port
  CASRAD_ADDRESS     Override bind address
  CASRAD_DATA        Override data directory
  MODE               Application mode (production/development)
  DEBUG              Enable debug logging
  NO_COLOR           Disable color output (https://no-color.org/)
`, binaryName, binaryName)
		os.Exit(0)
	}

	// --service: manage system service
	if serviceFlag {
		if serviceAction == "" {
			fmt.Fprintf(os.Stderr, "Usage: %s --service ACTION\n", binaryName)
			fmt.Fprintf(os.Stderr, "Actions: --install --uninstall --enable --disable --status --restart --start --stop\n")
			os.Exit(1)
		}

		mgr := service.NewManager("casrad")
		switch serviceAction {
		case "install":
			if err := mgr.Install(); err != nil {
				fmt.Fprintf(os.Stderr, "Service install failed: %v\n", err)
				os.Exit(1)
			}
			fmt.Println("Service installed and started.")

		case "uninstall":
			fmt.Print("This will delete ALL data, configs, and the system user. Continue? [y/N] ")
			var confirm string
			fmt.Scanln(&confirm)
			if strings.ToLower(strings.TrimSpace(confirm)) != "y" {
				fmt.Println("Aborted.")
				os.Exit(0)
			}
			if err := mgr.Uninstall(); err != nil {
				fmt.Fprintf(os.Stderr, "Service uninstall failed: %v\n", err)
				os.Exit(1)
			}
			fmt.Println("Service uninstalled.")

		case "enable":
			if err := mgr.Enable(); err != nil {
				fmt.Fprintf(os.Stderr, "Service enable failed: %v\n", err)
				os.Exit(1)
			}
			fmt.Println("Service enabled.")

		case "disable":
			if err := mgr.Disable(); err != nil {
				fmt.Fprintf(os.Stderr, "Service disable failed: %v\n", err)
				os.Exit(1)
			}
			fmt.Println("Service disabled.")

		case "status":
			status, err := mgr.Status()
			if err != nil {
				fmt.Fprintf(os.Stderr, "Service status failed: %v\n", err)
				os.Exit(1)
			}
			fmt.Println(status)

		case "restart":
			if err := mgr.Restart(); err != nil {
				fmt.Fprintf(os.Stderr, "Service restart failed: %v\n", err)
				os.Exit(1)
			}
			fmt.Println("Service restarted.")

		case "start":
			if err := mgr.Start(); err != nil {
				fmt.Fprintf(os.Stderr, "Service start failed: %v\n", err)
				os.Exit(1)
			}
			fmt.Println("Service started.")

		case "stop":
			if err := mgr.Stop(); err != nil {
				fmt.Fprintf(os.Stderr, "Service stop failed: %v\n", err)
				os.Exit(1)
			}
			fmt.Println("Service stopped.")

		default:
			fmt.Fprintf(os.Stderr, "Unknown service action: %s\n", serviceAction)
			fmt.Fprintf(os.Stderr, "Valid actions: --install --uninstall --enable --disable --status --restart --start --stop\n")
			os.Exit(1)
		}
		os.Exit(0)
	}

	// --maintenance: backup/restore/update operations
	if maintenanceFlag {
		switch maintenanceAction {
		case "backup":
			fmt.Fprintf(os.Stderr, "Backup destination: %s\n", maintenanceArg)
			fmt.Fprintln(os.Stderr, "backup: subsystem not yet implemented in this build")
			os.Exit(1)

		case "restore":
			if maintenanceArg == "" {
				fmt.Fprintf(os.Stderr, "Usage: %s --maintenance restore <file>\n", binaryName)
				os.Exit(1)
			}
			fmt.Fprintln(os.Stderr, "restore: subsystem not yet implemented in this build")
			os.Exit(1)

		case "update":
			fmt.Fprintln(os.Stderr, "update: subsystem not yet implemented in this build")
			os.Exit(1)

		default:
			fmt.Fprintf(os.Stderr, "Usage: %s --maintenance backup|restore|update\n", binaryName)
			os.Exit(1)
		}
	}

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load config: %v\n", err)
		os.Exit(1)
	}

	// Apply command-line overrides to config
	if port > 0 {
		cfg.Server.Port = port
	}
	if address != "" {
		cfg.Server.Address = address
	}
	if debug {
		cfg.Server.Debug = true
	}
	if dataPath != "" {
		os.Setenv("CASRAD_DATA", dataPath)
	}
	if langFlag != "" {
		os.Setenv("CASRAD_LANG", langFlag)
	}
	if configPath != "" {
		os.Setenv("CASRAD_CONFIG", configPath)
	}

	// Initialize health handler with build info and mode
	handler.AppVersion = Version
	handler.BuildCommit = CommitID
	handler.BuildDate = BuildDate
	handler.SetMode(string(mode.Get()))
	handler.InitHealth()

	// Start built-in scheduler with all 12 required tasks per AI.md PART 19
	sched := scheduler.New()
	scheduler.RegisterDefaultTasks(sched)
	sched.Start()
	defer sched.Stop()

	// Create and start the HTTP server
	srv, err := server.New(cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create server: %v\n", err)
		os.Exit(1)
	}

	if err := srv.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Server error: %v\n", err)
		os.Exit(1)
	}
}
