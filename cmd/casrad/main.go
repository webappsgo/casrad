package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"runtime"

	"github.com/casapps/casrad/internal/core"
)

var (
	Version   = "1.0.0"
	BuildTime = "unknown"
	GitCommit = "unknown"
)

func main() {
	var (
		showHelp    = flag.Bool("h", false, "Show help")
		showVersion = flag.Bool("v", false, "Show version")
		port        = flag.Int("p", 0, "Override default port (default: auto 64000-64999)")
		dataPath    = flag.String("d", "", "Override data directory (default: OS-specific)")
		debug       = flag.Bool("debug", false, "Enable debug logging")
	)

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "CASRAD - Complete Audio Streaming, Radio, and Distribution Server\n\n")
		fmt.Fprintf(os.Stderr, "Usage: %s [flags]\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Flags:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nCASRAD is designed to work with zero configuration.\n")
		fmt.Fprintf(os.Stderr, "Simply run the binary and open your browser to the displayed URL.\n")
	}

	flag.Parse()

	if *showHelp {
		flag.Usage()
		os.Exit(0)
	}

	if *showVersion {
		fmt.Printf("CASRAD %s\n", Version)
		fmt.Printf("Build Time: %s\n", BuildTime)
		fmt.Printf("Git Commit: %s\n", GitCommit)
		fmt.Printf("Go Version: %s\n", runtime.Version())
		fmt.Printf("OS/Arch: %s/%s\n", runtime.GOOS, runtime.GOARCH)
		os.Exit(0)
	}

	config := &core.Config{
		Port:     *port,
		DataPath: *dataPath,
		Debug:    *debug,
	}

	app, err := core.NewCASRAD(config)
	if err != nil {
		log.Fatalf("Failed to initialize CASRAD: %v", err)
	}

	if err := app.Run(); err != nil {
		log.Fatalf("Failed to run CASRAD: %v", err)
	}
}