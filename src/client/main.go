// CASRAD CLI client — casrad-cli
// See AI.md PART 33 for CLI specification
package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// Version information injected by ldflags at build time
var (
	Version      = "dev"
	CommitID     = "unknown"
	BuildDate    = "unknown"
	OfficialSite = ""
)

// CLIConfig holds cli.yml configuration
type CLIConfig struct {
	Server string `yaml:"server"`
	Token  string `yaml:"token"`
	Format string `yaml:"format"`
	Lang   string `yaml:"lang"`
}

// configDir returns the OS-specific config directory for casrad-cli per AI.md PART 33
func configDir() string {
	switch runtime.GOOS {
	case "windows":
		appData := os.Getenv("APPDATA")
		if appData == "" {
			appData = filepath.Join(os.Getenv("USERPROFILE"), "AppData", "Roaming")
		}
		return filepath.Join(appData, "casapps", "casrad")
	default:
		xdg := os.Getenv("XDG_CONFIG_HOME")
		if xdg != "" {
			return filepath.Join(xdg, "casapps", "casrad")
		}
		home, _ := os.UserHomeDir()
		return filepath.Join(home, ".config", "casapps", "casrad")
	}
}

// loadCLIConfig loads cli.yml from the config directory.
// Returns empty config (not an error) if the file doesn't exist.
// Refuses to load if file permissions are too open (not 0600) on Unix.
func loadCLIConfig() (*CLIConfig, error) {
	cfgFile := filepath.Join(configDir(), "cli.yml")
	info, err := os.Stat(cfgFile)
	if os.IsNotExist(err) {
		return &CLIConfig{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("cli.yml stat: %w", err)
	}

	// Enforce 0600 on Unix — refuse to load world/group readable files
	if runtime.GOOS != "windows" {
		perm := info.Mode().Perm()
		if perm&0o077 != 0 {
			return nil, fmt.Errorf("cli.yml has insecure permissions %04o — must be 0600", perm)
		}
	}

	data, err := os.ReadFile(cfgFile)
	if err != nil {
		return nil, fmt.Errorf("cli.yml read: %w", err)
	}

	var cfg CLIConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("cli.yml parse: %w", err)
	}
	return &cfg, nil
}

// loadToken resolves the API token per AI.md PART 33 priority:
// --token flag > --token-file flag > CASRAD_TOKEN env > cli.yml > {config_dir}/token
func loadToken(flagToken, flagTokenFile string, fileCfg *CLIConfig) (string, error) {
	// Priority 1: --token flag
	if flagToken != "" {
		return flagToken, nil
	}

	// Priority 2: --token-file flag
	if flagTokenFile != "" {
		info, err := os.Stat(flagTokenFile)
		if err != nil {
			return "", fmt.Errorf("token-file: %w", err)
		}
		if runtime.GOOS != "windows" && info.Mode().Perm()&0o077 != 0 {
			return "", fmt.Errorf("token-file %s has insecure permissions %04o — must be 0600",
				flagTokenFile, info.Mode().Perm())
		}
		data, err := os.ReadFile(flagTokenFile)
		if err != nil {
			return "", fmt.Errorf("token-file read: %w", err)
		}
		return strings.TrimSpace(string(data)), nil
	}

	// Priority 3: CASRAD_TOKEN env var
	if tok := os.Getenv("CASRAD_TOKEN"); tok != "" {
		return tok, nil
	}

	// Priority 4: cli.yml token field
	if fileCfg != nil && fileCfg.Token != "" {
		return fileCfg.Token, nil
	}

	// Priority 5: {config_dir}/token file
	tokenFile := filepath.Join(configDir(), "token")
	info, err := os.Stat(tokenFile)
	if err == nil {
		if runtime.GOOS != "windows" && info.Mode().Perm()&0o077 != 0 {
			return "", fmt.Errorf("token file %s has insecure permissions %04o — must be 0600",
				tokenFile, info.Mode().Perm())
		}
		data, err := os.ReadFile(tokenFile)
		if err == nil {
			tok := strings.TrimSpace(string(data))
			if tok != "" {
				return tok, nil
			}
		}
	}

	return "", nil
}

// colorDisabled reports whether terminal color should be suppressed.
// Priority: --color flag > NO_COLOR env > auto (TTY)
func colorDisabled(colorFlag string) bool {
	switch strings.ToLower(colorFlag) {
	case "always":
		return false
	case "never":
		return true
	}
	// auto: check NO_COLOR env per spec
	if v := os.Getenv("NO_COLOR"); v != "" {
		return true
	}
	return false
}

// doGet performs an authenticated GET request against the server and prints the result
func doGet(serverURL, path, token, format string) error {
	url := strings.TrimRight(serverURL, "/") + path
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("request: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("connect to %s: %w", serverURL, err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read response: %w", err)
	}

	switch strings.ToLower(format) {
	case "json":
		// Pretty-print if valid JSON
		var v interface{}
		if json.Unmarshal(body, &v) == nil {
			out, _ := json.MarshalIndent(v, "", "  ")
			fmt.Println(string(out))
		} else {
			fmt.Print(string(body))
		}
	default:
		fmt.Print(string(body))
	}
	return nil
}

func main() {
	args := os.Args[1:]
	binaryName := filepath.Base(os.Args[0])

	// Flag parsing
	var (
		flagToken     = ""
		flagTokenFile = ""
		flagServer    = ""
		flagFormat    = "json"
		flagColor     = "auto"
		flagLang      = "en"
		flagShell     = ""
		flagDebug     = false
		flagUser      = ""
		showVersion   = false
		showHelp      = false
	)

	// Parse arguments
	for i := 0; i < len(args); i++ {
		arg := args[i]
		nextVal := func() string {
			if i+1 < len(args) {
				i++
				return args[i]
			}
			return ""
		}

		switch arg {
		case "-h", "--help":
			showHelp = true
		case "-v", "--version":
			showVersion = true
		case "--token":
			flagToken = nextVal()
		case "--token-file":
			flagTokenFile = nextVal()
		case "--server":
			flagServer = nextVal()
		case "--format":
			flagFormat = nextVal()
		case "--color":
			flagColor = nextVal()
		case "--lang":
			flagLang = nextVal()
		case "--shell":
			flagShell = nextVal()
		case "--debug":
			flagDebug = true
		case "--user":
			flagUser = nextVal()
		default:
			if strings.HasPrefix(arg, "--token=") {
				flagToken = strings.TrimPrefix(arg, "--token=")
			} else if strings.HasPrefix(arg, "--server=") {
				flagServer = strings.TrimPrefix(arg, "--server=")
			} else if strings.HasPrefix(arg, "--format=") {
				flagFormat = strings.TrimPrefix(arg, "--format=")
			} else if strings.HasPrefix(arg, "--color=") {
				flagColor = strings.TrimPrefix(arg, "--color=")
			} else if strings.HasPrefix(arg, "--lang=") {
				flagLang = strings.TrimPrefix(arg, "--lang=")
			} else if strings.HasPrefix(arg, "--shell=") {
				flagShell = strings.TrimPrefix(arg, "--shell=")
			} else if strings.HasPrefix(arg, "--user=") {
				flagUser = strings.TrimPrefix(arg, "--user=")
			}
		}
	}

	// Suppress unused-variable warnings for flags not yet wired to subcommands
	_ = flagLang
	_ = flagUser
	_ = flagDebug
	_ = colorDisabled(flagColor)

	if flagShell != "" {
		fmt.Printf("# %s completions for %s — not yet generated\n", flagShell, binaryName)
		os.Exit(0)
	}

	if showVersion {
		fmt.Printf("%s %s (%s) built %s\n", binaryName, Version, CommitID, BuildDate)
		os.Exit(0)
	}

	if showHelp {
		fmt.Printf(`%s - CASRAD command-line client

Usage: %s [flags] <command> [args]

Flags:
  -h, --help                  Show this help message
  -v, --version               Show version information
  --token TOKEN               API token (overrides all other sources)
  --token-file PATH           Read API token from file (must be 0600)
  --server URL                Server URL (default: from cli.yml or CASRAD_SERVER env)
  --format json|text          Output format (default: json)
  --color always|never|auto   Color output (default: auto)
  --lang CODE                 Language code (default: en)
  --shell bash|zsh|fish       Output shell completions and exit
  --debug                     Enable debug output
  --user USERNAME             Act as this user (requires admin token)

Token priority (highest to lowest):
  1. --token flag
  2. --token-file flag
  3. CASRAD_TOKEN environment variable
  4. token field in cli.yml
  5. %s/token file

Config file: %s/cli.yml (must be 0600)

Commands:
  health                      Show server health status
  version                     Show server version

`, binaryName, binaryName, configDir(), configDir())
		os.Exit(0)
	}

	// Load config file
	fileCfg, err := loadCLIConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: %v\n", binaryName, err)
		os.Exit(1)
	}

	// Resolve server URL
	serverURL := flagServer
	if serverURL == "" {
		serverURL = os.Getenv("CASRAD_SERVER")
	}
	if serverURL == "" && fileCfg.Server != "" {
		serverURL = fileCfg.Server
	}
	if serverURL == "" {
		serverURL = "http://localhost:64000"
	}

	// Resolve format
	format := flagFormat
	if format == "" && fileCfg.Format != "" {
		format = fileCfg.Format
	}

	// Resolve API token
	token, err := loadToken(flagToken, flagTokenFile, fileCfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: %v\n", binaryName, err)
		os.Exit(1)
	}

	// Dispatch subcommands
	if len(args) == 0 {
		fmt.Fprintf(os.Stderr, "Usage: %s [flags] <command>\nRun '%s --help' for help.\n",
			binaryName, binaryName)
		os.Exit(1)
	}

	// Find the first non-flag argument as the command
	var command string
	for _, a := range args {
		if !strings.HasPrefix(a, "-") {
			command = a
			break
		}
	}

	switch command {
	case "health":
		if err := doGet(serverURL, "/api/v1/server/healthz", token, format); err != nil {
			fmt.Fprintf(os.Stderr, "%s: %v\n", binaryName, err)
			os.Exit(1)
		}

	case "version":
		if err := doGet(serverURL, "/api/v1/server/healthz", token, format); err != nil {
			fmt.Fprintf(os.Stderr, "%s: %v\n", binaryName, err)
			os.Exit(1)
		}

	case "":
		fmt.Fprintf(os.Stderr, "Usage: %s [flags] <command>\nRun '%s --help' for help.\n",
			binaryName, binaryName)
		os.Exit(1)

	default:
		fmt.Fprintf(os.Stderr, "%s: unknown command %q\nRun '%s --help' for help.\n",
			binaryName, command, binaryName)
		os.Exit(1)
	}
}
