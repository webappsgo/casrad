// CASRAD - Complete Audio Streaming, Radio, and Distribution
// See AI.md for specification details
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	"github.com/casapps/casrad/src/config"
	"github.com/casapps/casrad/src/mode"
	"github.com/casapps/casrad/src/server"
)

// Version information - set by ldflags during build
var (
	Version   = "dev"
	CommitID  = "unknown"
	BuildDate = "unknown"
)

func main() {
	// Initialize mode from environment first (CLI flags override later)
	mode.Init()

	// Parse command line arguments
	args := os.Args[1:]
	showHelp := false
	showVersion := false
	port := 0
	address := ""
	debug := false
	dataPath := ""
	modeFlag := ""

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "-h", "--help":
			showHelp = true
		case "-v", "--version":
			showVersion = true
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
		}
	}

	// Apply mode flag (CLI overrides environment)
	if modeFlag != "" {
		mode.Set(mode.FromString(modeFlag))
	}
	if debug {
		mode.SetDebug(true)
	}

	// Get binary name for help text
	binaryName := filepath.Base(os.Args[0])

	if showHelp {
		fmt.Printf(`%s - Complete Audio Streaming, Radio, and Distribution

Usage: %s [flags]

Flags:
  -h, --help          Show this help message
  -v, --version       Show version information
  -p, --port PORT     Override port (default: auto 64000-64999)
  -a, --address ADDR  Override bind address (default: 0.0.0.0)
  -d, --data PATH     Override data directory
  --mode MODE         Application mode (production, development, prod, dev)
  --debug             Enable debug logging

Environment Variables:
  CASRAD_PORT         Override port
  CASRAD_ADDRESS      Override bind address
  CASRAD_DATA         Override data directory
  MODE                Application mode (production/development)
  DEBUG               Enable debug logging (truthy values)

Documentation: https://github.com/casapps/casrad
`, binaryName, binaryName)
		os.Exit(0)
	}

	if showVersion {
		fmt.Printf("%s version %s\n", binaryName, Version)
		fmt.Printf("  Commit:     %s\n", CommitID)
		fmt.Printf("  Build Date: %s\n", BuildDate)
		os.Exit(0)
	}

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load config: %v\n", err)
		os.Exit(1)
	}

	// Apply command line overrides
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

	// Create and start server
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
